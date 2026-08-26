#!/usr/bin/env python3
import argparse
import csv
import json
import os
import signal
import subprocess
import sys
import tempfile
import time
from datetime import datetime, timezone
from statistics import median
from typing import Any

os.environ.setdefault("MPLCONFIGDIR", os.path.join(tempfile.gettempdir(), "jaops-matplotlib-cache"))
os.makedirs(os.environ["MPLCONFIGDIR"], exist_ok=True)

import matplotlib.pyplot as plt
from matplotlib.axes import Axes
from matplotlib.ticker import FuncFormatter, NullFormatter

from .metrics import METRIC_UNITS, MICRO_METRICS, SIMULATOR_METRICS, SIMULATOR_THRESHOLDS
from .plotting import (
    PLOT_COLORS,
    add_density_band,
    add_threshold_bands,
    add_threshold_line_label,
    apply_log_y_axis,
    apply_percentage_y_axis,
    apply_x_tick_formatter,
    apply_y_axis_floor,
    format_axis_tick,
    percent_change,
    plot_series,
    range_values,
    raw_percentile_columns,
    split_change_extremes,
)

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
BENCHMARK_DIR = os.path.dirname(SCRIPT_DIR)
LONG_TERM_BASELINE_DIR = os.path.join(BENCHMARK_DIR, "baselines", "long-term")
LONG_TERM_SIMULATOR_RESULTS = os.path.join(LONG_TERM_BASELINE_DIR, "simulator.json")
LONG_TERM_MICRO_RESULTS = os.path.join(LONG_TERM_BASELINE_DIR, "microbenchmarks.json")
METRIC_SEMANTICS_VERSION = 2
DEFAULT_STREAMS = "1,5,10,25,50,100,250,500,750,1000"
DEFAULT_PARAMETERS = ",".join(
    [
        "/myproject/CCSDS_Packet_ID",
        "/myproject/CCSDS_Packet_Sequence",
        "/myproject/CCSDS_Packet_Length",
        "/myproject/EpochUSNO",
        "/myproject/OrbitNumberCumulative",
        "/myproject/ElapsedSeconds",
        "/myproject/A",
        "/myproject/Height",
        "/myproject/Position",
        "/myproject/Velocity",
        "/myproject/Latitude",
        "/myproject/Longitude",
        "/myproject/Battery1_Voltage",
        "/myproject/Battery2_Voltage",
        "/myproject/Battery1_Temp",
        "/myproject/Battery2_Temp",
        "/myproject/Magnetometer",
        "/myproject/Sunsensor",
        "/myproject/Sunsensor_Beta",
        "/myproject/Gyro",
        "/myproject/Detector_Temp",
        "/myproject/Shadow",
        "/myproject/Contact_Golbasi_GS",
        "/myproject/Contact_Svalbard",
        "/myproject/Payload_Status",
        "/myproject/Payload_Error_Flag",
        "/myproject/ADCS_Error_Flag",
        "/myproject/CDHS_Error_Flag",
        "/myproject/COMMS_Error_Flag",
        "/myproject/EPS_Error_Flag",
        "/myproject/COMMS_Status",
        "/myproject/CDHS_Status",
        "/myproject/Mode_Night",
        "/myproject/Mode_Day",
        "/myproject/Mode_Payload",
        "/myproject/Mode_XBand",
        "/myproject/Mode_SBand",
        "/myproject/Mode_Safe",
        "/myproject/Enum_Para_1",
        "/myproject/Enum_Para_2",
        "/myproject/Enum_Para_3",
    ]
)
METRIC_LABELS = {key: value["label"] for source in (SIMULATOR_METRICS, MICRO_METRICS) for key, value in source.items()}
PLOT_TITLES = {key: value["title"] for source in (SIMULATOR_METRICS, MICRO_METRICS) for key, value in source.items()}
PLOT_FILE_NAMES = {key: os.path.splitext(value["file"])[0] for source in (SIMULATOR_METRICS, MICRO_METRICS) for key, value in source.items()}
PERFORMANCE_PLOT_KEYS = [
    "live_memory_growth_bytes",
    "total_allocated_bytes",
    "values_read_per_sec",
    "values_read_fresh_pct",
    "median_tick_runstream_busy",
    "setup",
]
MICRO_PLOT_KEYS = [
    "frame_numeric_full",
    "frame_numeric_average",
    "frame_numeric_average_minmax",
    "frame_discrete",
    "process_stream_10_values",
]
LOG_Y_KEYS = {
    "median_tick_runstream_busy",
    "setup",
    "live_memory_growth_bytes",
    "total_allocated_bytes",
    "values_read_per_sec",
    "frame_numeric_full",
    "frame_numeric_average",
    "frame_numeric_average_minmax",
    "frame_discrete",
    "process_stream_10_values",
}
TIME_KEYS = {
    "setup",
    "median_tick_runstream_busy",
    "frame_numeric_full",
    "frame_numeric_average",
    "frame_numeric_average_minmax",
    "frame_discrete",
    "process_stream_10_values",
}
BYTE_KEYS = {"live_memory_growth_bytes", "total_allocated_bytes"}
THRESHOLDS = SIMULATOR_THRESHOLDS
def parse_streams(value: str) -> list[int]:
    streams = sorted({int(part.strip()) for part in value.split(",") if part.strip()})
    if not streams or any(n <= 0 for n in streams):
        raise argparse.ArgumentTypeError("streams must be positive integers")
    return streams


def run_simulator(args: argparse.Namespace) -> subprocess.Popen[str] | None:
    if args.no_simulator:
        return None

    simulator = os.path.join(args.quickstart_dir, "simulator.py")
    testdata = os.path.join(args.quickstart_dir, "testdata.ccsds")
    cmd = [
        "python3",
        simulator,
        "--tm_host",
        args.simulator_host,
        "--tm_port",
        str(args.simulator_port),
        "--rate",
        str(args.simulator_rate),
        "--testdata",
        testdata,
    ]
    return subprocess.Popen(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
        preexec_fn=os.setsid,
    )


def stop_process(process: subprocess.Popen[str] | None) -> None:
    if process is None or process.poll() is not None:
        return
    os.killpg(os.getpgid(process.pid), signal.SIGTERM)
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        os.killpg(os.getpgid(process.pid), signal.SIGKILL)
        process.wait(timeout=5)


def run_go_scenario(args: argparse.Namespace, streams: list[int]) -> dict[str, Any]:
    env = os.environ.copy()
    env.setdefault("GOCACHE", os.path.join(tempfile.gettempdir(), "jaops-go-build-cache"))
    os.makedirs(env["GOCACHE"], exist_ok=True)

    cmd = [
        "go",
        "run",
        "./scripts/benchmarks/simulator/scenario.go",
        "--address",
        args.yamcs_address,
        "--instance",
        args.instance,
        "--processor",
        args.processor,
        "--streams",
        ",".join(str(n) for n in streams),
        "--parameters",
        args.parameters,
        "--duration",
        args.duration,
        "--warmup",
        args.warmup,
        "--warmup-scenario-streams",
        str(args.warmup_scenario_streams),
        "--warmup-scenario-duration",
        args.warmup_scenario_duration,
        "--read-interval",
        args.read_interval,
        "--freshness-window",
        args.freshness_window,
    ]
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    if result.returncode != 0:
        raise RuntimeError(
            "Go Yamcs stream scenario failed with exit code "
            f"{result.returncode}\nSTDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}"
        )
    return json.loads(result.stdout)


def run_go_microbenchmarks(args: argparse.Namespace) -> list[dict[str, Any]]:
    env = os.environ.copy()
    env.setdefault("GOCACHE", os.path.join(tempfile.gettempdir(), "jaops-go-build-cache"))
    os.makedirs(env["GOCACHE"], exist_ok=True)
    with tempfile.NamedTemporaryFile(prefix="jaops-microbench-", suffix=".json", delete=False) as fp:
        output_path = fp.name
    env["BENCHMARK_MICRO_OUTPUT"] = output_path
    cmd = [
        "go",
        "test",
        "./pkg/source",
        "-tags",
        "benchmark",
        "-run",
        "TestBenchmarkMicroCurves",
        "-count",
        "1",
        "-timeout",
        "10m",
    ]
    result = subprocess.run(cmd, capture_output=True, text=True, env=env)
    if result.returncode != 0:
        raise RuntimeError(
            "Go benchmark micro curves failed with exit code "
            f"{result.returncode}\nSTDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}"
        )
    try:
        with open(output_path, encoding="utf-8") as fp:
            payload = json.load(fp)
    finally:
        try:
            os.remove(output_path)
        except OSError:
            pass
    metrics = payload.get("metrics", [])
    return metrics if isinstance(metrics, list) else []


def write_csv(path: str, rows: list[dict[str, Any]]) -> None:
    if not rows:
        return
    fields = ["streams"] + sorted({key for row in rows for key in row.keys()} - {"streams"})
    with open(path, "w", newline="", encoding="utf-8") as fp:
        writer = csv.DictWriter(fp, fieldnames=fields)
        writer.writeheader()
        writer.writerows(rows)


def load_baseline_results(path: str) -> tuple[dict[str, Any] | None, str]:
    if not path:
        return None, "no baseline result path provided"
    if not os.path.exists(path):
        return None, f"baseline result file does not exist: {path}"
    try:
        with open(path, encoding="utf-8") as fp:
            result = json.load(fp)
    except (OSError, json.JSONDecodeError) as err:
        return None, f"could not read baseline result file: {err}"
    scenarios = result.get("scenarios")
    if not isinstance(scenarios, list):
        return None, "baseline result file does not contain a scenarios array"
    if not all(isinstance(row, dict) and isinstance(row.get("streams"), int) for row in scenarios):
        return None, "baseline scenarios do not contain compatible stream counts"
    return result, "baseline loaded"


def load_json_file(path: str) -> dict[str, Any] | None:
    try:
        with open(path, encoding="utf-8") as fp:
            value = json.load(fp)
    except (OSError, json.JSONDecodeError):
        return None
    return value if isinstance(value, dict) else None


def metric_semantics_compatible(result: dict[str, Any] | None) -> bool:
    if not result:
        return False
    return result.get("metric_semantics_version") == METRIC_SEMANTICS_VERSION


def write_long_term_metadata(args: argparse.Namespace, result: dict[str, Any]) -> None:
    metadata = {
        "name": "long-term",
        "description": "Checked-in benchmark baseline used as a stable reference curve in benchmark plots.",
        "source_environment": "github-actions" if os.environ.get("GITHUB_ACTIONS") == "true" else "local",
        "created_from": "scripts/benchmarks/baselines/long-term/simulator.json",
        "created_at": datetime.now(timezone.utc).isoformat(),
        "github": {
            "repository": os.environ.get("GITHUB_REPOSITORY", ""),
            "workflow": os.environ.get("GITHUB_WORKFLOW", ""),
            "run_id": os.environ.get("GITHUB_RUN_ID", ""),
            "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT", ""),
            "sha": os.environ.get("GITHUB_SHA", ""),
        },
        "yamcs_quickstart": "jaops-space/yamcs-quickstart",
        "yamcs_instance": result.get("instance", args.instance),
        "yamcs_processor": result.get("processor", args.processor),
        "simulator_rate_hz": args.simulator_rate,
        "stream_read_interval": args.read_interval,
        "freshness_window": args.freshness_window,
        "warmup_scenario": {
            "streams": args.warmup_scenario_streams,
            "duration": args.warmup_scenario_duration,
        },
        "system": result.get("system", {}),
        "streams": [row.get("streams") for row in result.get("scenarios", []) if isinstance(row, dict)],
        "parameter_count": len(result.get("parameters", [])),
        "refresh_command": "Run the `Refresh benchmark long-term baseline` GitHub Actions workflow.",
        "metric_semantics_version": METRIC_SEMANTICS_VERSION,
    }
    with open(os.path.join(LONG_TERM_BASELINE_DIR, "metadata.json"), "w", encoding="utf-8") as fp:
        json.dump(metadata, fp, indent=2)
        fp.write("\n")


def scaled_series(key: str, values: list[float], scale_reference: float | None = None) -> tuple[list[float], str]:
    label = METRIC_LABELS.get(key, key.replace("_", " "))
    max_abs = scale_reference if scale_reference is not None else max(abs(value) for value in values) if values else 0
    if key in TIME_KEYS:
        if max_abs >= 1_000_000:
            return [value / 1_000_000 for value in values], f"{label} (ms)"
        if max_abs >= 1_000:
            return [value / 1_000 for value in values], f"{label} (us)"
        return values, f"{label} (ns)"
    if key in BYTE_KEYS:
        if max_abs >= 1024 * 1024:
            return [value / (1024 * 1024) for value in values], f"{label} (MiB)"
        if max_abs >= 1024:
            return [value / 1024 for value in values], f"{label} (KiB)"
        return values, f"{label} (bytes)"
    if key.endswith("_pct"):
        return values, f"{label} (%)"
    if key in METRIC_UNITS:
        return values, f"{label} ({METRIC_UNITS[key]})"
    return values, label


def split_axis_label(label: str) -> tuple[str, str]:
    if label.endswith(")") and " (" in label:
        name, unit = label.rsplit(" (", 1)
        return name, unit[:-1]
    return label, ""


def y_axis_label(key: str, default_label: str, unit: str) -> str:
    if key in {
        "frame_numeric_full",
        "frame_numeric_average",
        "frame_numeric_average_minmax",
        "frame_discrete",
    }:
        return "Median conversion time"
    if key == "process_stream_10_values":
        return "Median processing time for 10 values"
    if key == "median_tick_runstream_busy":
        return "Median busy time per 1s tick"
    if key == "setup":
        return "Setup time"
    return default_label


def thresholds_for_plot(key: str, xs: list[int], scale_reference: float) -> list[tuple[str, str, str, list[int], list[float]]]:
    lines = []
    for threshold in THRESHOLDS.values():
        if threshold.get("plot_key") != key:
            continue
        operator = str(threshold.get("operator", "max"))
        for level, color in [("warn", PLOT_COLORS["warn"]), ("fail", PLOT_COLORS["fail"])]:
            raw_value = float(threshold[level])
            if threshold.get("scale") == "per_stream":
                values = [raw_value * x for x in xs]
            else:
                values = [raw_value for _ in xs]
            if operator == "max" and max(values) > scale_reference * 5:
                continue
            if operator == "min" and max(values) < scale_reference / 5:
                continue
            scaled_values, _ = scaled_series(key, values, scale_reference)
            lines.append((level, operator, color, xs, scaled_values))
    return lines


def threshold_level(threshold_lines: list[tuple[str, str, str, list[int], list[float]]], level: str) -> list[float] | None:
    for threshold_level_name, _operator, _color, _xs, values in threshold_lines:
        if threshold_level_name == level:
            return values
    return None


def style_metric_axis(ax: Axes, title: str, x_label: str = "Number of concurrent Grafana streams") -> None:
    ax.set_title(title, loc="left", fontsize=13, fontweight="normal", pad=18)
    ax.set_xlabel(x_label)
    ax.spines[["top", "right"]].set_visible(False)
    for spine in ax.spines.values():
        spine.set_color(PLOT_COLORS["grid"])
    ax.grid(True, which="major", color=PLOT_COLORS["grid"], alpha=0.78)
    ax.grid(True, which="minor", color=PLOT_COLORS["minor_grid"], alpha=0.9)


def apply_y_tick_formatter(ax: Axes, axis_unit: str) -> None:
    if not axis_unit:
        return
    formatter = FuncFormatter(lambda value, _position: format_axis_tick(value, axis_unit))
    ax.yaxis.set_major_formatter(formatter)
    ax.yaxis.set_minor_formatter(NullFormatter())
    ax.yaxis.offsetText.set_visible(False)


def plot_metric(
    output_dir: str,
    rows: list[dict[str, Any]],
    key: str,
    baseline_rows: list[dict[str, Any]] | None = None,
    long_term_rows: list[dict[str, Any]] | None = None,
) -> str | None:
    points = [(row["streams"], row.get(key)) for row in rows if row.get(key) is not None]
    points = [(x, y) for x, y in points if isinstance(y, (int, float))]
    if not points:
        return None

    points.sort(key=lambda item: item[0])
    xs = [point[0] for point in points]
    raw_ys = [float(point[1]) for point in points]
    rows_by_x = {row["streams"]: row for row in rows if "streams" in row}
    raw_mins: list[float] = []
    raw_maxs: list[float] = []
    for x, y in points:
        row = next(row for row in rows if row.get("streams") == x and row.get(key) == y)
        min_value, max_value = range_values(row, key)
        raw_mins.append(float(y) if min_value is None else min_value)
        raw_maxs.append(float(y) if max_value is None else max_value)
    raw_percentiles = raw_percentile_columns(rows_by_x, xs, key)
    baseline_points = []
    if baseline_rows:
        baseline_points = [(row["streams"], row.get(key)) for row in baseline_rows if row.get(key) is not None]
        baseline_points = [(x, y) for x, y in baseline_points if isinstance(y, (int, float))]
        baseline_points.sort(key=lambda item: item[0])
    baseline_raw_ys = [float(point[1]) for point in baseline_points]
    long_term_points = []
    if long_term_rows:
        long_term_points = [(row["streams"], row.get(key)) for row in long_term_rows if row.get(key) is not None]
        long_term_points = [(x, y) for x, y in long_term_points if isinstance(y, (int, float))]
        long_term_points.sort(key=lambda item: item[0])
    long_term_raw_ys = [float(point[1]) for point in long_term_points]
    all_raw_ys = raw_ys + baseline_raw_ys + long_term_raw_ys
    all_raw_ys += raw_mins + raw_maxs
    scale_reference = max(abs(value) for value in all_raw_ys) if all_raw_ys else 0
    ys, label = scaled_series(key, raw_ys, scale_reference)
    ys_min, _ = scaled_series(key, raw_mins, scale_reference)
    ys_max, _ = scaled_series(key, raw_maxs, scale_reference)
    percentile_columns = {suffix: scaled_series(key, values, scale_reference)[0] for suffix, values in raw_percentiles.items()}
    y_label, y_unit = split_axis_label(label)
    baseline_xs = [point[0] for point in baseline_points]
    baseline_ys, _ = scaled_series(key, baseline_raw_ys, scale_reference)
    long_term_xs = [point[0] for point in long_term_points]
    long_term_ys, _ = scaled_series(key, long_term_raw_ys, scale_reference)
    path = os.path.join(output_dir, f"{PLOT_FILE_NAMES.get(key, key)}.png")
    threshold_lines = thresholds_for_plot(key, xs, scale_reference)

    fig, ax = plt.subplots(figsize=(9.6, 5.4))
    fig.patch.set_facecolor("#f8fafc")
    ax.set_facecolor("white")
    plot_series(ax, xs, ys, "head", zorder=3)
    if baseline_points:
        plot_series(ax, baseline_xs, baseline_ys, "pr_base", zorder=2)
    if long_term_points:
        plot_series(ax, long_term_xs, long_term_ys, "long_term", zorder=2)
    ax.set_xscale("log")
    apply_x_tick_formatter(ax)
    ax.set_ylabel(y_axis_label(key, y_label, y_unit))
    style_metric_axis(ax, PLOT_TITLES.get(key, f"{label} by concurrent Grafana streams"))
    # Deliberately excludes threshold values: thresholds are drawn as
    # overlays and must not stretch the axis when they sit far from the
    # actual data, which would otherwise squash the real curve.
    axis_values = ys + ys_min + ys_max + baseline_ys + long_term_ys
    if key.endswith("_pct"):
        apply_percentage_y_axis(ax, axis_values)
    elif key in LOG_Y_KEYS:
        apply_log_y_axis(ax, axis_values)
    else:
        apply_y_axis_floor(ax, axis_values)
    apply_y_tick_formatter(ax, y_unit)
    add_density_band(ax, xs, ys, ys_min, ys_max, percentile_columns, PLOT_COLORS["head"])
    for level, _operator, color, threshold_xs, threshold_values in threshold_lines:
        ax.plot(threshold_xs, threshold_values, color=color, linewidth=1.25, alpha=0.95, zorder=2, clip_on=True)
    add_threshold_bands(ax, threshold_lines)
    warn_values = threshold_level(threshold_lines, "warn")
    fail_values = threshold_level(threshold_lines, "fail")
    if warn_values is not None:
        add_threshold_line_label(ax, xs, warn_values, "warn", PLOT_COLORS["warn_text"])
    if fail_values is not None:
        add_threshold_line_label(ax, xs, fail_values, "fail", PLOT_COLORS["fail_text"])
    ax.legend(frameon=False, loc="upper left", ncols=3)
    fig.tight_layout()
    fig.savefig(path, dpi=180, facecolor="white")
    plt.close()
    return path


def plot_micro_metric(
    output_dir: str,
    rows: list[dict[str, Any]],
    key: str,
    baseline_rows: list[dict[str, Any]] | None = None,
    long_term_rows: list[dict[str, Any]] | None = None,
) -> str | None:
    metric_rows = [row for row in rows if row.get("metric") == key and isinstance(row.get("x"), int)]
    points = [(row["x"], row.get("median_ns")) for row in metric_rows if isinstance(row.get("median_ns"), (int, float))]
    if not points:
        return None

    points.sort(key=lambda item: item[0])
    xs = [point[0] for point in points]
    raw_ys = [float(point[1]) for point in points]
    metric_rows_by_x = {row["x"]: row for row in metric_rows if isinstance(row.get("x"), int)}
    raw_mins: list[float] = []
    raw_maxs: list[float] = []
    for x, y in points:
        row = metric_rows_by_x.get(x, {})
        min_value, max_value = range_values(row, key, "ns")
        raw_mins.append(float(y) if min_value is None else min_value)
        raw_maxs.append(float(y) if max_value is None else max_value)
    raw_percentiles = raw_percentile_columns(metric_rows_by_x, xs, key, "ns")
    x_label = next((str(row.get("x_label")) for row in metric_rows if row.get("x_label")), "N")

    baseline_points = []
    if baseline_rows:
        baseline_points = [
            (row["x"], row.get("median_ns"))
            for row in baseline_rows
            if row.get("metric") == key and isinstance(row.get("x"), int) and isinstance(row.get("median_ns"), (int, float))
        ]
        baseline_points.sort(key=lambda item: item[0])
    baseline_raw_ys = [float(point[1]) for point in baseline_points]

    long_term_points = []
    if long_term_rows:
        long_term_points = [
            (row["x"], row.get("median_ns"))
            for row in long_term_rows
            if row.get("metric") == key and isinstance(row.get("x"), int) and isinstance(row.get("median_ns"), (int, float))
        ]
        long_term_points.sort(key=lambda item: item[0])
    long_term_raw_ys = [float(point[1]) for point in long_term_points]

    all_raw_ys = raw_ys + raw_mins + raw_maxs + baseline_raw_ys + long_term_raw_ys
    scale_reference = max(abs(value) for value in all_raw_ys) if all_raw_ys else 0
    ys, label = scaled_series(key, raw_ys, scale_reference)
    ys_min, _ = scaled_series(key, raw_mins, scale_reference)
    ys_max, _ = scaled_series(key, raw_maxs, scale_reference)
    percentile_columns = {suffix: scaled_series(key, values, scale_reference)[0] for suffix, values in raw_percentiles.items()}
    y_label, y_unit = split_axis_label(label)
    baseline_xs = [point[0] for point in baseline_points]
    baseline_ys, _ = scaled_series(key, baseline_raw_ys, scale_reference)
    long_term_xs = [point[0] for point in long_term_points]
    long_term_ys, _ = scaled_series(key, long_term_raw_ys, scale_reference)
    path = os.path.join(output_dir, f"{PLOT_FILE_NAMES.get(key, key)}.png")

    fig, ax = plt.subplots(figsize=(9.6, 5.4))
    fig.patch.set_facecolor("#f8fafc")
    ax.set_facecolor("white")
    plot_series(ax, xs, ys, "head", zorder=3)
    if baseline_points:
        plot_series(ax, baseline_xs, baseline_ys, "pr_base", zorder=2)
    if long_term_points:
        plot_series(ax, long_term_xs, long_term_ys, "long_term", zorder=2)
    ax.set_xscale("log")
    apply_x_tick_formatter(ax)
    ax.set_ylabel(y_axis_label(key, y_label, y_unit))
    style_metric_axis(ax, PLOT_TITLES.get(key, f"{label} by {x_label.lower()}"), x_label)
    axis_values = ys + ys_min + ys_max + baseline_ys + long_term_ys
    if key in LOG_Y_KEYS:
        apply_log_y_axis(ax, axis_values)
    else:
        apply_y_axis_floor(ax, axis_values)
    apply_y_tick_formatter(ax, y_unit)
    add_density_band(ax, xs, ys, ys_min, ys_max, percentile_columns, PLOT_COLORS["head"])
    ax.legend(frameon=False, loc="upper left", ncols=3)
    fig.tight_layout()
    fig.savefig(path, dpi=180, facecolor="white")
    plt.close()
    return path


def plot_all_metrics(
    output_dir: str,
    rows: list[dict[str, Any]],
    baseline_rows: list[dict[str, Any]] | None = None,
    long_term_rows: list[dict[str, Any]] | None = None,
) -> list[str]:
    plots_dir = os.path.join(output_dir, "plots")
    os.makedirs(plots_dir, exist_ok=True)
    for filename in os.listdir(plots_dir):
        if filename.endswith(".png"):
            os.remove(os.path.join(plots_dir, filename))
    return [path for key in PERFORMANCE_PLOT_KEYS if (path := plot_metric(plots_dir, rows, key, baseline_rows, long_term_rows))]


def plot_all_micro_metrics(
    output_dir: str,
    rows: list[dict[str, Any]],
    baseline_rows: list[dict[str, Any]] | None = None,
    long_term_rows: list[dict[str, Any]] | None = None,
) -> list[str]:
    plots_dir = os.path.join(output_dir, "plots")
    os.makedirs(plots_dir, exist_ok=True)
    return [path for key in MICRO_PLOT_KEYS if (path := plot_micro_metric(plots_dir, rows, key, baseline_rows, long_term_rows))]


def threshold_value(row: dict[str, Any], key: str) -> float:
    if key == "setup_per_stream":
        return float(row["setup"]) / max(float(row["streams"]), 1)
    if key == "live_memory_growth_bytes_per_stream":
        return float(row["live_memory_growth_bytes"]) / max(float(row["streams"]), 1)
    if key == "values_read_per_sec_per_stream":
        return float(row["values_read_per_sec"]) / max(float(row["streams"]), 1)
    return float(row[key])


def evaluate_thresholds(rows: list[dict[str, Any]]) -> list[dict[str, Any]]:
    results = []
    for key, threshold in THRESHOLDS.items():
        values = [threshold_value(row, key) for row in rows]
        operator = threshold["operator"]
        observed = max(values) if operator == "max" else min(values)
        warn = float(threshold["warn"])
        fail = float(threshold["fail"])
        if operator == "max":
            status = "fail" if observed > fail else "warn" if observed > warn else "pass"
        else:
            status = "fail" if observed < fail else "warn" if observed < warn else "pass"
        results.append(
            {
                "metric": key,
                "operator": operator,
                "observed": observed,
                "warn": warn,
                "fail": fail,
                "unit": threshold["unit"],
                "status": status,
            }
        )
    return results


def summarize_baseline_changes(
    rows: list[dict[str, Any]],
    baseline_rows: list[dict[str, Any]] | None,
    label: str,
) -> list[dict[str, Any]]:
    if not baseline_rows:
        return []

    baseline_by_streams = {row["streams"]: row for row in baseline_rows}
    summaries = []
    for key in PERFORMANCE_PLOT_KEYS:
        changes = []
        for row in rows:
            streams = row["streams"]
            baseline = baseline_by_streams.get(streams)
            if not baseline:
                continue
            current_value = row.get(key)
            baseline_value = baseline.get(key)
            if not isinstance(current_value, (int, float)) or not isinstance(baseline_value, (int, float)):
                continue
            if baseline_value == 0:
                continue

            change = percent_change(float(current_value), float(baseline_value))
            if change is not None:
                changes.append(change)
        if changes:
            max_negative_change, max_positive_change = split_change_extremes(changes)
            summaries.append(
                {
                    "baseline": label,
                    "metric": key,
                    "samples": len(changes),
                    "median_change_pct": median(changes),
                    "max_negative_change_pct": max_negative_change,
                    "max_positive_change_pct": max_positive_change,
                    "unit": METRIC_UNITS.get(key, ""),
                }
            )
    return summaries


def summarize_micro_baseline_changes(
    rows: list[dict[str, Any]],
    baseline_rows: list[dict[str, Any]] | None,
    label: str,
) -> list[dict[str, Any]]:
    if not baseline_rows:
        return []

    baseline_by_metric_x = {
        (row.get("metric"), row.get("x")): row
        for row in baseline_rows
        if isinstance(row, dict) and isinstance(row.get("metric"), str) and isinstance(row.get("x"), int)
    }
    summaries = []
    for key in MICRO_PLOT_KEYS:
        changes = []
        for row in rows:
            if row.get("metric") != key or not isinstance(row.get("x"), int):
                continue
            baseline = baseline_by_metric_x.get((key, row["x"]))
            if not baseline:
                continue
            current_value = row.get("median_ns")
            baseline_value = baseline.get("median_ns")
            if not isinstance(current_value, (int, float)) or not isinstance(baseline_value, (int, float)):
                continue
            change = percent_change(float(current_value), float(baseline_value))
            if change is not None:
                changes.append(change)
        if changes:
            max_negative_change, max_positive_change = split_change_extremes(changes)
            summaries.append(
                {
                    "baseline": label,
                    "metric": key,
                    "samples": len(changes),
                    "median_change_pct": median(changes),
                    "max_negative_change_pct": max_negative_change,
                    "max_positive_change_pct": max_positive_change,
                    "unit": METRIC_UNITS.get(key, ""),
                }
            )
    return summaries

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

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
LONG_TERM_BASELINE_DIR = os.path.join(SCRIPT_DIR, "baselines", "long-term")
LONG_TERM_BASELINE_RESULTS = os.path.join(LONG_TERM_BASELINE_DIR, "yamcs-stream-results.json")
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
METRIC_LABELS = {
    "live_memory_growth_bytes": "Live memory used during simulator run",
    "total_allocated_bytes": "Total memory allocated during simulator run",
    "values_read_per_sec": "Values read per second from simulator buffers",
    "values_read_fresh_pct": "Values read within 1s simulator tick",
    "median_tick_runstream_busy": "RunStream busy time per 1s simulator tick",
    "setup": "Time to set up simulator streams",
    "frame_numeric_full": "Time to convert numeric buffer to full frame",
    "frame_numeric_average": "Time to convert numeric buffer to average frame",
    "frame_numeric_average_minmax": "Time to convert numeric buffer to average/min/max frame",
    "frame_discrete": "Time to convert discrete buffer to frame",
    "process_stream_10_values": "Time to process 10 values into stream buffers",
}
PLOT_TITLES = {
    "live_memory_growth_bytes": "Simulator - Live memory used while N Grafana streams run",
    "total_allocated_bytes": "Simulator - Total memory allocated while N Grafana streams run",
    "values_read_per_sec": "Simulator - Values read per second by N Grafana streams",
    "values_read_fresh_pct": "Simulator - Values read within the same 1s simulator tick",
    "median_tick_runstream_busy": "Simulator - Time spent doing RunStream work per 1s tick",
    "setup": "Simulator - Time to set up N Grafana streams",
    "frame_numeric_full": "Microbenchmark - Time to convert a numeric buffer to a full frame",
    "frame_numeric_average": "Microbenchmark - Time to convert a numeric buffer to an average frame",
    "frame_numeric_average_minmax": "Microbenchmark - Time to convert a numeric buffer to an average/min/max frame",
    "frame_discrete": "Microbenchmark - Time to convert a discrete buffer to a frame",
    "process_stream_10_values": "Microbenchmark - Time to process 10 values into N stream buffers",
}
PLOT_FILE_NAMES = {
    "median_tick_runstream_busy": "median_tick_runstream_busy",
    "setup": "setup",
    "frame_numeric_full": "frame_numeric_full",
    "frame_numeric_average": "frame_numeric_average",
    "frame_numeric_average_minmax": "frame_numeric_average_minmax",
    "frame_discrete": "frame_discrete",
    "process_stream_10_values": "process_stream_10_values",
}
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
THRESHOLDS = {
    "setup_per_stream": {
        "warn": 50_000_000,
        "fail": 100_000_000,
        "operator": "max",
        "unit": "ns/stream",
        "plot_key": "setup",
        "scale": "per_stream",
    },
    "live_memory_growth_bytes_per_stream": {
        "warn": 200_000,
        "fail": 1_000_000,
        "operator": "max",
        "unit": "bytes/stream",
        "plot_key": "live_memory_growth_bytes",
        "scale": "per_stream",
    },
    "values_read_per_sec_per_stream": {
        "warn": 0.8,
        "fail": 0.5,
        "operator": "min",
        "unit": "values/sec/stream",
        "plot_key": "values_read_per_sec",
        "scale": "per_stream",
    },
    "values_read_fresh_pct": {
        "warn": 99,
        "fail": 95,
        "operator": "min",
        "unit": "%",
        "plot_key": "values_read_fresh_pct",
        "scale": "constant",
    },
    "median_tick_runstream_busy": {
        "warn": 1_000_000_000,
        "fail": 1_200_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "median_tick_runstream_busy",
        "scale": "constant",
    },
}
METRIC_UNITS = {
    "live_memory_growth_bytes": "bytes",
    "total_allocated_bytes": "bytes",
    "values_read_per_sec": "values/sec",
    "values_read_fresh_pct": "%",
    "avg_tick_runstream": "ns",
    "median_tick_runstream_busy": "ns",
    "setup": "ns",
    "frame_numeric_full": "ns",
    "frame_numeric_average": "ns",
    "frame_numeric_average_minmax": "ns",
    "frame_discrete": "ns",
    "process_stream_10_values": "ns",
}
PLOT_COLORS = {
    "head": "#1d4ed8",
    "pr_base": "#64748b",
    "long_term": "#16a34a",
    "warn": "#f59e0b",
    "warn_text": "#b45309",
    "fail": "#ef4444",
    "fail_text": "#b91c1c",
    "grid": "#e2e8f0",
    "minor_grid": "#f1f5f9",
}


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
        "./scripts/benchmarks/yamcs_stream_scenario.go",
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
        "created_from": "scripts/benchmarks/baselines/long-term/yamcs-stream-results.json",
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
        return f"Median conversion time ({unit})" if unit else "Median conversion time"
    if key == "process_stream_10_values":
        return f"Median processing time for 10 values ({unit})" if unit else "Median processing time for 10 values"
    if key == "median_tick_runstream_busy":
        return f"Median busy time per 1s tick ({unit})" if unit else "Median busy time per 1s tick"
    if key == "setup":
        return f"Setup time ({unit})" if unit else "Setup time"
    return f"{default_label} ({unit})" if unit else default_label


def format_number(value: float) -> str:
    abs_value = abs(value)
    if abs_value == 0:
        return "0"
    if abs_value >= 100:
        return f"{value:.0f}"
    if abs_value >= 10:
        return f"{value:.1f}".rstrip("0").rstrip(".")
    if abs_value >= 1:
        return f"{value:.2f}".rstrip("0").rstrip(".")
    return f"{value:.3f}".rstrip("0").rstrip(".")


def format_time_tick(value: float, axis_unit: str) -> str:
    multipliers = {"ns": 1, "us": 1_000, "ms": 1_000_000, "s": 1_000_000_000}
    ns = value * multipliers.get(axis_unit, 1)
    abs_ns = abs(ns)
    if abs_ns >= 1_000_000_000:
        return f"{format_number(ns / 1_000_000_000)} s"
    if abs_ns >= 1_000_000:
        return f"{format_number(ns / 1_000_000)} ms"
    if abs_ns >= 1_000:
        return f"{format_number(ns / 1_000)} us"
    return f"{format_number(ns)} ns"


def format_byte_tick(value: float, axis_unit: str) -> str:
    multipliers = {"bytes": 1, "KiB": 1024, "MiB": 1024 * 1024, "GiB": 1024 * 1024 * 1024}
    bytes_value = value * multipliers.get(axis_unit, 1)
    abs_bytes = abs(bytes_value)
    if abs_bytes >= 1024 * 1024 * 1024:
        return f"{format_number(bytes_value / (1024 * 1024 * 1024))} GiB"
    if abs_bytes >= 1024 * 1024:
        return f"{format_number(bytes_value / (1024 * 1024))} MiB"
    if abs_bytes >= 1024:
        return f"{format_number(bytes_value / 1024)} KiB"
    return f"{format_number(bytes_value)} bytes"


def format_rate_tick(value: float, axis_unit: str) -> str:
    if axis_unit == "values/sec":
        if abs(value) >= 1_000_000:
            return f"{format_number(value / 1_000_000)}M values/s"
        if abs(value) >= 1_000:
            return f"{format_number(value / 1_000)}k values/s"
        return f"{format_number(value)} values/s"
    return f"{format_number(value)} {axis_unit}".rstrip()


def format_axis_tick(value: float, axis_unit: str) -> str:
    if axis_unit in {"ns", "us", "ms", "s"}:
        return format_time_tick(value, axis_unit)
    if axis_unit in {"bytes", "KiB", "MiB", "GiB"}:
        return format_byte_tick(value, axis_unit)
    if axis_unit == "%":
        return f"{format_number(value)}%"
    return format_rate_tick(value, axis_unit)


def apply_y_axis_floor(values: list[float]) -> None:
    if not values:
        return
    ymin = min(values)
    ymax = max(values)
    if ymin >= 0:
        upper = ymax * 1.12 if ymax > 0 else 1
        plt.ylim(bottom=0, top=upper)
        return

    span = ymax - ymin
    min_span = max(abs(ymax), abs(ymin), 1) * 0.2
    if span < min_span:
        midpoint = (ymax + ymin) / 2
        half = min_span / 2
        plt.ylim(midpoint - half, midpoint + half)


def apply_log_y_axis(values: list[float]) -> None:
    positive = [value for value in values if value > 0]
    if not positive:
        return
    plt.yscale("log")
    plt.ylim(bottom=min(positive) * 0.8, top=max(positive) * 1.2)


def apply_percentage_y_axis(values: list[float]) -> None:
    if not values:
        return
    ymin = max(0, min(values) - 5)
    ymax = min(105, max(values) + 2)
    if ymax - ymin < 10:
        midpoint = (ymin + ymax) / 2
        ymin = max(0, midpoint - 5)
        ymax = min(105, midpoint + 5)
    plt.ylim(bottom=ymin, top=ymax)


def thresholds_for_plot(key: str, xs: list[int], scale_reference: float) -> list[tuple[str, str, str, list[float]]]:
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
            lines.append((level, operator, color, scaled_values))
    return lines


def threshold_level(threshold_lines: list[tuple[str, str, str, list[float]]], level: str) -> float | None:
    for threshold_level_name, _operator, _color, values in threshold_lines:
        if threshold_level_name == level and values and all(value == values[0] for value in values):
            return float(values[0])
    return None


def threshold_operator(threshold_lines: list[tuple[str, str, str, list[float]]]) -> str:
    for _level, operator, _color, _values in threshold_lines:
        return operator
    return "max"


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


def add_threshold_regions(ax: Axes, threshold_lines: list[tuple[str, str, str, list[float]]]) -> None:
    warn = threshold_level(threshold_lines, "warn")
    fail = threshold_level(threshold_lines, "fail")
    if warn is None or fail is None:
        return

    ymin, ymax = ax.get_ylim()
    operator = threshold_operator(threshold_lines)
    if operator == "min":
        ax.axhspan(max(ymin, fail), min(ymax, warn), color=PLOT_COLORS["warn"], alpha=0.07, zorder=0)
        ax.axhspan(ymin, min(ymax, fail), color=PLOT_COLORS["fail"], alpha=0.055, zorder=0)
        return

    ax.axhspan(max(ymin, warn), min(ymax, fail), color=PLOT_COLORS["warn"], alpha=0.07, zorder=0)
    ax.axhspan(max(ymin, fail), ymax, color=PLOT_COLORS["fail"], alpha=0.055, zorder=0)


def add_threshold_label(ax: Axes, label: str, value: float, color: str) -> None:
    xmin, xmax = ax.get_xlim()
    ymin, ymax = ax.get_ylim()
    if not ymin <= value <= ymax:
        return
    text_y = value * 1.04 if ax.get_yscale() == "log" else value + (ymax - ymin) * 0.018
    text_y = min(max(text_y, ymin), ymax)
    ax.text(xmax, text_y, label, ha="right", va="bottom", color=color, fontsize=9)


def add_head_fill(ax: Axes, xs: list[int], ys: list[float], axis_values: list[float]) -> None:
    if not xs or not ys:
        return
    if ax.get_yscale() == "log":
        positive = [value for value in axis_values if value > 0]
        if not positive:
            return
        fill_bottom = min(positive) * 0.8
    else:
        fill_bottom = max(0, ax.get_ylim()[0])
    ax.fill_between(xs, fill_bottom, ys, color=PLOT_COLORS["head"], alpha=0.045, zorder=1)


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
    scale_reference = max(abs(value) for value in all_raw_ys) if all_raw_ys else 0
    ys, label = scaled_series(key, raw_ys, scale_reference)
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
    ax.plot(xs, ys, color=PLOT_COLORS["head"], linewidth=3.0, label="HEAD", zorder=3)
    ax.scatter(xs, ys, s=34, color=PLOT_COLORS["head"], edgecolor="white", linewidth=1.1, zorder=4)
    if baseline_points:
        ax.plot(baseline_xs, baseline_ys, color=PLOT_COLORS["pr_base"], linewidth=2.3, label="PR base", zorder=2)
    if long_term_points:
        ax.plot(
            long_term_xs,
            long_term_ys,
            color=PLOT_COLORS["long_term"],
            linewidth=2.1,
            linestyle=(0, (5, 4)),
            label="Long-term baseline",
            zorder=2,
        )
    for level, _operator, color, threshold_values in threshold_lines:
        ax.plot(xs, threshold_values, color=color, linewidth=1.25, alpha=0.95, zorder=2)
    ax.set_xscale("log")
    ax.set_ylabel(y_axis_label(key, y_label, y_unit))
    style_metric_axis(ax, PLOT_TITLES.get(key, f"{label} by concurrent Grafana streams"))
    axis_values = ys + baseline_ys + long_term_ys + [value for _, _, _, threshold_values in threshold_lines for value in threshold_values]
    if key.endswith("_pct"):
        plt.sca(ax)
        apply_percentage_y_axis(axis_values)
    elif key in LOG_Y_KEYS:
        plt.sca(ax)
        apply_log_y_axis(axis_values)
    else:
        plt.sca(ax)
        apply_y_axis_floor(axis_values)
    apply_y_tick_formatter(ax, y_unit)
    add_head_fill(ax, xs, ys, axis_values)
    add_threshold_regions(ax, threshold_lines)
    warn = threshold_level(threshold_lines, "warn")
    fail = threshold_level(threshold_lines, "fail")
    if warn is not None:
        add_threshold_label(ax, "warn", warn, PLOT_COLORS["warn_text"])
    if fail is not None:
        add_threshold_label(ax, "fail", fail, PLOT_COLORS["fail_text"])
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

    all_raw_ys = raw_ys + baseline_raw_ys + long_term_raw_ys
    scale_reference = max(abs(value) for value in all_raw_ys) if all_raw_ys else 0
    ys, label = scaled_series(key, raw_ys, scale_reference)
    y_label, y_unit = split_axis_label(label)
    baseline_xs = [point[0] for point in baseline_points]
    baseline_ys, _ = scaled_series(key, baseline_raw_ys, scale_reference)
    long_term_xs = [point[0] for point in long_term_points]
    long_term_ys, _ = scaled_series(key, long_term_raw_ys, scale_reference)
    path = os.path.join(output_dir, f"{PLOT_FILE_NAMES.get(key, key)}.png")

    fig, ax = plt.subplots(figsize=(9.6, 5.4))
    fig.patch.set_facecolor("#f8fafc")
    ax.set_facecolor("white")
    ax.plot(xs, ys, color=PLOT_COLORS["head"], linewidth=3.0, label="HEAD", zorder=3)
    ax.scatter(xs, ys, s=34, color=PLOT_COLORS["head"], edgecolor="white", linewidth=1.1, zorder=4)
    if baseline_points:
        ax.plot(baseline_xs, baseline_ys, color=PLOT_COLORS["pr_base"], linewidth=2.3, label="PR base", zorder=2)
    if long_term_points:
        ax.plot(
            long_term_xs,
            long_term_ys,
            color=PLOT_COLORS["long_term"],
            linewidth=2.1,
            linestyle=(0, (5, 4)),
            label="Long-term baseline",
            zorder=2,
        )
    ax.set_xscale("log")
    ax.set_ylabel(y_axis_label(key, y_label, y_unit))
    style_metric_axis(ax, PLOT_TITLES.get(key, f"{label} by {x_label.lower()}"), x_label)
    axis_values = ys + baseline_ys + long_term_ys
    if key in LOG_Y_KEYS:
        plt.sca(ax)
        apply_log_y_axis(axis_values)
    else:
        plt.sca(ax)
        apply_y_axis_floor(axis_values)
    apply_y_tick_formatter(ax, y_unit)
    add_head_fill(ax, xs, ys, axis_values)
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

            current = float(current_value)
            base = float(baseline_value)
            changes.append(100 * (current - base) / abs(base))
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


def split_change_extremes(changes: list[float]) -> tuple[float, float]:
    negative_changes = [change for change in changes if change < 0]
    positive_changes = [change for change in changes if change > 0]
    return (
        min(negative_changes) if negative_changes else 0,
        max(positive_changes) if positive_changes else 0,
    )


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
            if baseline_value == 0:
                continue
            changes.append(100 * (float(current_value) - float(baseline_value)) / abs(float(baseline_value)))
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


def main() -> None:
    parser = argparse.ArgumentParser(description="Benchmark N concurrent Grafana streams with live Yamcs quickstart data.")
    parser.add_argument("--output-dir", default="benchmark-output")
    parser.add_argument("--yamcs-address", default="localhost:8090")
    parser.add_argument("--instance", default="myproject")
    parser.add_argument("--processor", default="realtime")
    parser.add_argument("--streams", type=parse_streams, default=parse_streams(DEFAULT_STREAMS))
    parser.add_argument("--parameters", default=DEFAULT_PARAMETERS)
    parser.add_argument("--duration", default="10s")
    parser.add_argument("--warmup", default="3s")
    parser.add_argument("--warmup-scenario-streams", type=int, default=25)
    parser.add_argument("--warmup-scenario-duration", default="3s")
    parser.add_argument("--read-interval", default="1s")
    parser.add_argument("--freshness-window", default="1s")
    parser.add_argument("--quickstart-dir", default="/tmp/yamcs-quickstart")
    parser.add_argument("--no-simulator", action="store_true", help="Do not start simulator.py before running the scenario")
    parser.add_argument("--simulator-host", default="127.0.0.1")
    parser.add_argument("--simulator-port", type=int, default=10015)
    parser.add_argument("--simulator-rate", type=int, default=1)
    parser.add_argument("--baseline-results", default="", help="Optional previous benchmark JSON to plot and compare against")
    parser.add_argument("--baseline-commit", default="", help="Commit hash used to produce the PR base benchmark")
    parser.add_argument("--allow-local-baseline", action="store_true", help="Allow writing the checked-in long-term baseline outside CI")
    parser.add_argument("--fail-on-threshold", action="store_true", help="Exit non-zero when any benchmark threshold fails")
    argv = sys.argv[1:]
    if argv and argv[0] == "--":
        argv = argv[1:]
    args = parser.parse_args(argv)

    refreshing_long_term = os.path.abspath(args.output_dir) == os.path.abspath(LONG_TERM_BASELINE_DIR)
    if refreshing_long_term and os.environ.get("CI") != "true" and not args.allow_local_baseline:
        raise SystemExit(
            "Refusing to refresh the checked-in long-term baseline outside CI. "
            "Use the GitHub Actions baseline refresh workflow, or pass --allow-local-baseline for a local diagnostic run."
        )

    os.makedirs(args.output_dir, exist_ok=True)
    started_at = datetime.now(timezone.utc).isoformat()
    simulator = run_simulator(args)
    try:
        if simulator is not None:
            time.sleep(2)
        result = run_go_scenario(args, args.streams)
        result["microbenchmarks"] = run_go_microbenchmarks(args)
    finally:
        stop_process(simulator)

    result["python_started_at"] = started_at
    result["python_finished_at"] = datetime.now(timezone.utc).isoformat()
    result["simulator_rate"] = args.simulator_rate
    result["metric_semantics_version"] = METRIC_SEMANTICS_VERSION

    json_path = os.path.join(args.output_dir, "yamcs-stream-results.json")
    csv_path = os.path.join(args.output_dir, "yamcs-stream-results.csv")

    threshold_results = evaluate_thresholds(result["scenarios"])
    baseline_result, baseline_message = load_baseline_results(args.baseline_results)
    baseline_compatible = metric_semantics_compatible(baseline_result)
    if baseline_result and not baseline_compatible:
        baseline_message = "baseline loaded but metric semantics version is incompatible"
    baseline_rows = baseline_result.get("scenarios", []) if baseline_compatible else None
    baseline_micro_rows = baseline_result.get("microbenchmarks", []) if baseline_compatible else None
    if os.path.abspath(args.output_dir) == os.path.abspath(LONG_TERM_BASELINE_DIR):
        long_term_result = None
        long_term_message = "long-term baseline skipped while refreshing it"
    else:
        long_term_result, long_term_message = load_baseline_results(LONG_TERM_BASELINE_RESULTS)
    long_term_compatible = metric_semantics_compatible(long_term_result)
    if long_term_result and not long_term_compatible:
        long_term_message = "long-term baseline loaded but metric semantics version is incompatible"
    long_term_rows = long_term_result.get("scenarios", []) if long_term_compatible else None
    long_term_micro_rows = long_term_result.get("microbenchmarks", []) if long_term_compatible else None
    baseline_change_summaries = summarize_baseline_changes(result["scenarios"], baseline_rows, "PR base")
    baseline_change_summaries += summarize_baseline_changes(result["scenarios"], long_term_rows, "Long-term baseline")
    baseline_change_summaries += summarize_micro_baseline_changes(result.get("microbenchmarks", []), baseline_micro_rows, "PR base")
    baseline_change_summaries += summarize_micro_baseline_changes(
        result.get("microbenchmarks", []), long_term_micro_rows, "Long-term baseline"
    )
    result["baseline"] = {
        "compatible": baseline_compatible,
        "path": args.baseline_results,
        "commit": args.baseline_commit,
        "message": baseline_message,
        "metric_semantics_version": baseline_result.get("metric_semantics_version") if baseline_result else None,
    }
    result["long_term_baseline"] = {
        "compatible": long_term_compatible,
        "path": os.path.relpath(LONG_TERM_BASELINE_RESULTS, os.getcwd()),
        "metadata": load_json_file(os.path.join(LONG_TERM_BASELINE_DIR, "metadata.json")) or {},
        "system": long_term_result.get("system", {}) if long_term_result else {},
        "environment": {
            "yamcs_address": long_term_result.get("yamcs_address", "") if long_term_result else "",
            "instance": long_term_result.get("instance", "") if long_term_result else "",
            "processor": long_term_result.get("processor", "") if long_term_result else "",
        },
        "message": long_term_message,
        "metric_semantics_version": long_term_result.get("metric_semantics_version") if long_term_result else None,
    }
    result["baseline_change_summaries"] = baseline_change_summaries
    result["thresholds"] = threshold_results
    with open(json_path, "w", encoding="utf-8") as fp:
        json.dump(result, fp, indent=2)
    if refreshing_long_term:
        write_long_term_metadata(args, result)
    write_csv(csv_path, result["scenarios"])
    plot_paths = plot_all_metrics(args.output_dir, result["scenarios"], baseline_rows, long_term_rows)
    plot_paths += plot_all_micro_metrics(
        args.output_dir,
        result.get("microbenchmarks", []),
        baseline_micro_rows,
        long_term_micro_rows,
    )

    print("=== Yamcs Stream Scenario Benchmark ===")
    print(f"scenarios: {len(result['scenarios'])}")
    print(f"plots generated: {len(plot_paths)}")
    print(f"baseline: {baseline_message}")
    print(f"long-term baseline: {long_term_message}")
    print(f"thresholds: {', '.join(t['metric'] + '=' + t['status'] for t in threshold_results)}")
    print(f"Artifacts written to: {os.path.abspath(args.output_dir)}")
    if args.fail_on_threshold and any(t["status"] == "fail" for t in threshold_results):
        raise SystemExit(1)


if __name__ == "__main__":
    main()

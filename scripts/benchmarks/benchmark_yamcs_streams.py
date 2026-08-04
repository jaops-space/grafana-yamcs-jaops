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
from typing import Any

os.environ.setdefault("MPLCONFIGDIR", os.path.join(tempfile.gettempdir(), "jaops-matplotlib-cache"))
os.makedirs(os.environ["MPLCONFIGDIR"], exist_ok=True)

import matplotlib.pyplot as plt

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
LONG_TERM_BASELINE_DIR = os.path.join(SCRIPT_DIR, "baselines", "long-term")
LONG_TERM_BASELINE_RESULTS = os.path.join(LONG_TERM_BASELINE_DIR, "yamcs-stream-results.json")
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
    "avg_read_clear": "Average read and clear time",
    "avg_process": "Average Yamcs listener processing time",
    "live_memory_growth_bytes": "Live memory used during run",
    "total_allocated_bytes": "Total memory allocated during run",
    "values_read_per_sec": "Values read per second from buffers",
    "values_read_fresh_pct": "Values read within 1s tick",
    "avg_value_read_age": "Average value age when read",
    "avg_tick_runstream": "Average RunStream wall time per 1s tick",
    "setup": "Stream setup time",
}
PLOT_TITLES = {
    "avg_read_clear": "Average time to read and clear one stream buffer",
    "avg_process": "Average time to process one Yamcs parameter update",
    "live_memory_growth_bytes": "Live memory used while N Grafana streams run",
    "total_allocated_bytes": "Total memory allocated while N Grafana streams run",
    "values_read_per_sec": "Values read per second from buffers by N Grafana streams",
    "values_read_fresh_pct": "Values read within the same 1s simulator tick",
    "avg_value_read_age": "Average age of values when Grafana stream reads them",
    "avg_tick_runstream": "Average RunStream wall time with N Grafana streams on 1s tickers",
    "setup": "Time to set up N Grafana streams",
}
PLOT_FILE_NAMES = {
    "avg_read_clear": "avg_read_clear",
    "avg_process": "avg_process",
    "avg_value_read_age": "avg_value_read_age",
    "avg_tick_runstream": "avg_tick_runstream",
    "setup": "setup",
}
PERFORMANCE_PLOT_KEYS = [
    "avg_read_clear",
    "avg_process",
    "live_memory_growth_bytes",
    "total_allocated_bytes",
    "values_read_per_sec",
    "values_read_fresh_pct",
    "avg_value_read_age",
    "avg_tick_runstream",
    "setup",
]
LOG_Y_KEYS = {
    "avg_read_clear",
    "avg_process",
    "avg_value_read_age",
    "avg_tick_runstream",
    "setup",
    "live_memory_growth_bytes",
    "total_allocated_bytes",
    "values_read_per_sec",
}
TIME_KEYS = {
    "avg_read_clear",
    "avg_process",
    "setup",
    "avg_value_read_age",
    "avg_tick_runstream",
}
BYTE_KEYS = {"live_memory_growth_bytes", "total_allocated_bytes"}
THRESHOLDS = {
    "avg_read_clear": {
        "warn": 1_000_000,
        "fail": 10_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "avg_read_clear",
        "scale": "constant",
    },
    "avg_process": {
        "warn": 1_000_000,
        "fail": 10_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "avg_process",
        "scale": "constant",
    },
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
    "avg_tick_runstream": {
        "warn": 1_000_000_000,
        "fail": 1_200_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "avg_tick_runstream",
        "scale": "constant",
    },
}
BASELINE_COMPARISON_RULES = {
    "avg_read_clear": {"direction": "lower", "warn_ratio": 1.30, "unit": "ns"},
    "avg_process": {"direction": "lower", "warn_ratio": 1.30, "unit": "ns"},
    "live_memory_growth_bytes": {"direction": "lower", "warn_ratio": 1.30, "unit": "bytes"},
    "total_allocated_bytes": {"direction": "lower", "warn_ratio": 1.30, "unit": "bytes"},
    "values_read_per_sec": {"direction": "higher", "warn_ratio": 0.70, "unit": "values/sec"},
    "values_read_fresh_pct": {"direction": "higher", "warn_ratio": 0.95, "unit": "%"},
    "avg_value_read_age": {"direction": "lower", "warn_ratio": 1.30, "unit": "ns"},
    "avg_tick_runstream": {"direction": "lower", "warn_ratio": 1.30, "unit": "ns"},
    "setup": {"direction": "lower", "warn_ratio": 1.30, "unit": "ns"},
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
    return values, label


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


def thresholds_for_plot(key: str, xs: list[int], scale_reference: float) -> list[tuple[str, str, list[float]]]:
    lines = []
    for threshold in THRESHOLDS.values():
        if threshold.get("plot_key") != key:
            continue
        for level, color in [("warn", "#c58a00"), ("fail", "#c53030")]:
            raw_value = float(threshold[level])
            if threshold.get("scale") == "per_stream":
                values = [raw_value * x for x in xs]
            else:
                values = [raw_value for _ in xs]
            if threshold.get("operator") == "max" and max(values) > scale_reference * 5:
                continue
            if threshold.get("operator") == "min" and max(values) < scale_reference / 5:
                continue
            scaled_values, _ = scaled_series(key, values, scale_reference)
            lines.append((level, color, scaled_values))
    return lines


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
    baseline_xs = [point[0] for point in baseline_points]
    baseline_ys, _ = scaled_series(key, baseline_raw_ys, scale_reference)
    long_term_xs = [point[0] for point in long_term_points]
    long_term_ys, _ = scaled_series(key, long_term_raw_ys, scale_reference)
    path = os.path.join(output_dir, f"{PLOT_FILE_NAMES.get(key, key)}.png")
    threshold_lines = thresholds_for_plot(key, xs, scale_reference)

    plt.figure(figsize=(10, 6))
    plt.plot(xs, ys, color="#2563eb", marker="o", label="PR")
    if baseline_points:
        plt.plot(baseline_xs, baseline_ys, color="#16a34a", marker="o", label="Base")
    if long_term_points:
        plt.plot(long_term_xs, long_term_ys, color="#f97316", marker="o", label="Long-term baseline")
    for level, color, threshold_values in threshold_lines:
        plt.plot(xs, threshold_values, linestyle="--", color=color, linewidth=1.2, label=f"{level} threshold")
    plt.xscale("log")
    plt.xlabel("Concurrent Grafana streams (N, log scale)")
    plt.ylabel(label)
    plt.title(PLOT_TITLES.get(key, f"{label} by concurrent Grafana streams"))
    axis_values = ys + baseline_ys + long_term_ys + [value for _, _, threshold_values in threshold_lines for value in threshold_values]
    if key in LOG_Y_KEYS:
        apply_log_y_axis(axis_values)
    else:
        apply_y_axis_floor(axis_values)
    plt.legend()
    plt.grid(True, which="both", alpha=0.25)
    plt.tight_layout()
    plt.savefig(path)
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


def evaluate_baseline_comparisons(rows: list[dict[str, Any]], baseline_rows: list[dict[str, Any]] | None) -> list[dict[str, Any]]:
    if not baseline_rows:
        return []

    baseline_by_streams = {row["streams"]: row for row in baseline_rows}
    comparisons = []
    for key, rule in BASELINE_COMPARISON_RULES.items():
        worst: dict[str, Any] | None = None
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
            ratio = current / base
            if rule["direction"] == "lower":
                status = "warn" if ratio >= float(rule["warn_ratio"]) else "pass"
                severity = ratio
            else:
                status = "warn" if ratio <= float(rule["warn_ratio"]) else "pass"
                severity = 1 / ratio if ratio > 0 else float("inf")

            change_pct = 100 * (current - base) / abs(base)
            candidate = {
                "metric": key,
                "plot": f"{PLOT_FILE_NAMES.get(key, key)}.png",
                "streams": streams,
                "status": status,
                "current": current,
                "baseline": base,
                "change_pct": change_pct,
                "ratio": ratio,
                "unit": rule["unit"],
                "direction": rule["direction"],
                "warn_ratio": rule["warn_ratio"],
            }
            if worst is None or severity > worst["_severity"]:
                worst = {**candidate, "_severity": severity}
        if worst:
            worst.pop("_severity", None)
            comparisons.append(worst)
    return comparisons


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
    parser.add_argument("--read-interval", default="1s")
    parser.add_argument("--freshness-window", default="1s")
    parser.add_argument("--quickstart-dir", default="/tmp/yamcs-quickstart")
    parser.add_argument("--no-simulator", action="store_true", help="Do not start simulator.py before running the scenario")
    parser.add_argument("--simulator-host", default="127.0.0.1")
    parser.add_argument("--simulator-port", type=int, default=10015)
    parser.add_argument("--simulator-rate", type=int, default=1)
    parser.add_argument("--baseline-results", default="", help="Optional previous benchmark JSON to plot and compare against")
    parser.add_argument("--fail-on-threshold", action="store_true", help="Exit non-zero when any benchmark threshold fails")
    argv = sys.argv[1:]
    if argv and argv[0] == "--":
        argv = argv[1:]
    args = parser.parse_args(argv)

    os.makedirs(args.output_dir, exist_ok=True)
    started_at = datetime.now(timezone.utc).isoformat()
    simulator = run_simulator(args)
    try:
        if simulator is not None:
            time.sleep(2)
        result = run_go_scenario(args, args.streams)
    finally:
        stop_process(simulator)

    result["python_started_at"] = started_at
    result["python_finished_at"] = datetime.now(timezone.utc).isoformat()
    result["simulator_rate"] = args.simulator_rate

    json_path = os.path.join(args.output_dir, "yamcs-stream-results.json")
    csv_path = os.path.join(args.output_dir, "yamcs-stream-results.csv")

    threshold_results = evaluate_thresholds(result["scenarios"])
    baseline_result, baseline_message = load_baseline_results(args.baseline_results)
    baseline_rows = baseline_result.get("scenarios", []) if baseline_result else None
    if os.path.abspath(args.output_dir) == os.path.abspath(LONG_TERM_BASELINE_DIR):
        long_term_result = None
        long_term_message = "long-term baseline skipped while refreshing it"
    else:
        long_term_result, long_term_message = load_baseline_results(LONG_TERM_BASELINE_RESULTS)
    long_term_rows = long_term_result.get("scenarios", []) if long_term_result else None
    baseline_comparisons = evaluate_baseline_comparisons(result["scenarios"], baseline_rows)
    result["baseline"] = {
        "compatible": baseline_result is not None,
        "path": args.baseline_results,
        "message": baseline_message,
    }
    result["long_term_baseline"] = {
        "compatible": long_term_result is not None,
        "path": os.path.relpath(LONG_TERM_BASELINE_RESULTS, os.getcwd()),
        "message": long_term_message,
    }
    result["baseline_comparisons"] = baseline_comparisons
    result["thresholds"] = threshold_results
    with open(json_path, "w", encoding="utf-8") as fp:
        json.dump(result, fp, indent=2)
    write_csv(csv_path, result["scenarios"])
    plot_paths = plot_all_metrics(args.output_dir, result["scenarios"], baseline_rows, long_term_rows)

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

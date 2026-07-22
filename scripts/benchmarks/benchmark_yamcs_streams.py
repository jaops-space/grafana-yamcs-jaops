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
    "avg_read_clear_ns": "Average read and clear latency",
    "avg_process_ns": "Average Yamcs listener process time",
    "live_memory_growth_bytes": "Live memory growth",
    "total_allocated_bytes": "Total allocated",
    "values_read_per_sec": "Values read per second",
    "values_read_fresh_pct": "Values read within freshness window",
    "values_read_stale_pct": "Values read after freshness window",
    "avg_value_read_age_ns": "Average value buffer age",
    "max_value_read_age_ns": "Max value buffer age",
    "max_value_stall_ns": "Max stall beyond freshness window",
    "avg_tick_runstream_ns": "Average RunStream tick wall time",
    "max_tick_runstream_ns": "Worst RunStream tick wall time",
    "max_tick_runstream_pct": "Worst RunStream tick wall time",
    "avg_tick_process_ns": "Average 1s tick Yamcs process workload",
    "max_tick_process_ns": "Worst 1s tick Yamcs process workload",
    "avg_tick_read_send_ns": "Average read/frame/send wall time per tick",
    "max_tick_read_send_ns": "Worst read/frame/send wall time per tick",
    "setup_ns": "Stream setup time",
}
PLOT_FILE_NAMES = {
    "avg_read_clear_ns": "avg_read_clear",
    "avg_process_ns": "avg_process",
    "avg_value_read_age_ns": "avg_value_read_age",
    "avg_tick_runstream_ns": "avg_tick_runstream",
    "setup_ns": "setup",
}
PERFORMANCE_PLOT_KEYS = [
    "avg_read_clear_ns",
    "avg_process_ns",
    "live_memory_growth_bytes",
    "total_allocated_bytes",
    "values_read_per_sec",
    "values_read_fresh_pct",
    "avg_value_read_age_ns",
    "avg_tick_runstream_ns",
    "setup_ns",
]
LOG_Y_KEYS = {"values_read_per_sec"}
TIME_KEYS = {
    "avg_read_clear_ns",
    "avg_process_ns",
    "setup_ns",
    "avg_value_read_age_ns",
    "max_value_read_age_ns",
    "max_value_stall_ns",
    "avg_tick_runstream_ns",
    "max_tick_runstream_ns",
}
BYTE_KEYS = {"live_memory_growth_bytes", "total_allocated_bytes"}
THRESHOLDS = {
    "avg_read_clear_ns": {
        "warn": 1_000_000,
        "fail": 10_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "avg_read_clear_ns",
        "scale": "constant",
    },
    "avg_process_ns": {
        "warn": 1_000_000,
        "fail": 10_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "avg_process_ns",
        "scale": "constant",
    },
    "setup_ns_per_stream": {
        "warn": 50_000_000,
        "fail": 100_000_000,
        "operator": "max",
        "unit": "ns/stream",
        "plot_key": "setup_ns",
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
    "max_value_read_age_ns": {
        "warn": 1_000_000_000,
        "fail": 2_000_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "max_value_read_age_ns",
        "scale": "constant",
    },
    "avg_tick_runstream_ns": {
        "warn": 1_000_000_000,
        "fail": 1_200_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "avg_tick_runstream_ns",
        "scale": "constant",
    },
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


def plot_metric(output_dir: str, rows: list[dict[str, Any]], key: str) -> str | None:
    points = [(row["streams"], row.get(key)) for row in rows if row.get(key) is not None]
    points = [(x, y) for x, y in points if isinstance(y, (int, float))]
    if not points:
        return None

    points.sort(key=lambda item: item[0])
    xs = [point[0] for point in points]
    raw_ys = [float(point[1]) for point in points]
    scale_reference = max(abs(value) for value in raw_ys) if raw_ys else 0
    ys, label = scaled_series(key, raw_ys, scale_reference)
    path = os.path.join(output_dir, f"{PLOT_FILE_NAMES.get(key, key)}.png")
    threshold_lines = thresholds_for_plot(key, xs, scale_reference)

    plt.figure(figsize=(10, 6))
    plt.plot(xs, ys, marker="o", label="observed")
    for level, color, threshold_values in threshold_lines:
        plt.plot(xs, threshold_values, linestyle="--", color=color, linewidth=1.2, label=f"{level} threshold")
    plt.xscale("log")
    plt.xlabel("Concurrent Grafana streams (N, log scale)")
    plt.ylabel(label)
    plt.title(f"{label} by concurrent Grafana streams")
    axis_values = ys + [value for _, _, threshold_values in threshold_lines for value in threshold_values]
    if key in LOG_Y_KEYS:
        apply_log_y_axis(axis_values)
    else:
        apply_y_axis_floor(axis_values)
    if threshold_lines:
        plt.legend()
    plt.grid(True, which="both", alpha=0.25)
    plt.tight_layout()
    plt.savefig(path)
    plt.close()
    return path


def plot_all_metrics(output_dir: str, rows: list[dict[str, Any]]) -> list[str]:
    plots_dir = os.path.join(output_dir, "plots")
    os.makedirs(plots_dir, exist_ok=True)
    for filename in os.listdir(plots_dir):
        if filename.endswith(".png"):
            os.remove(os.path.join(plots_dir, filename))
    return [path for key in PERFORMANCE_PLOT_KEYS if (path := plot_metric(plots_dir, rows, key))]


def threshold_value(row: dict[str, Any], key: str) -> float:
    if key == "setup_ns_per_stream":
        return float(row["setup_ns"]) / max(float(row["streams"]), 1)
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
    result["thresholds"] = threshold_results
    with open(json_path, "w", encoding="utf-8") as fp:
        json.dump(result, fp, indent=2)
    write_csv(csv_path, result["scenarios"])
    plot_paths = plot_all_metrics(args.output_dir, result["scenarios"])

    print("=== Yamcs Stream Scenario Benchmark ===")
    print(f"scenarios: {len(result['scenarios'])}")
    print(f"plots generated: {len(plot_paths)}")
    print(f"thresholds: {', '.join(t['metric'] + '=' + t['status'] for t in threshold_results)}")
    print(f"Artifacts written to: {os.path.abspath(args.output_dir)}")
    if args.fail_on_threshold and any(t["status"] == "fail" for t in threshold_results):
        raise SystemExit(1)


if __name__ == "__main__":
    main()

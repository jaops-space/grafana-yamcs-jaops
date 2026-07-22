#!/usr/bin/env python3
import argparse
import json
import os
import shutil
import sys
from typing import Any

COMMENT_MARKER = "<!-- jaops-yamcs-benchmark-report -->"
METRIC_NAMES = {
    "avg_read_clear_ns": "Average read/clear latency",
    "avg_process_ns": "Average Yamcs listener processing time",
    "setup_ns_per_stream": "Stream setup time per stream",
    "live_memory_growth_bytes_per_stream": "Live memory growth per stream",
    "values_read_per_sec_per_stream": "Values read per second per stream",
    "values_read_fresh_pct": "Values read within one tick",
    "avg_tick_runstream_ns": "Average RunStream tick wall time",
}
METRIC_DETAILS = {
    "avg_read_clear_ns": "Time spent clearing one stream buffer.",
    "avg_process_ns": "Time spent processing one Yamcs parameter update.",
    "setup_ns_per_stream": "Time spent creating stream demand state and Yamcs subscriptions.",
    "live_memory_growth_bytes_per_stream": "Additional live memory retained per stream during the scenario.",
    "values_read_per_sec_per_stream": "Per-stream throughput against the 1 Hz simulator cadence.",
    "values_read_fresh_pct": "Share of values read within the 1 second freshness window.",
    "avg_tick_runstream_ns": "Wall-clock time from the first RunStream starting read/frame/send work to the last RunStream finishing in one 1s tick.",
}
THRESHOLD_TO_PLOT = {
    "avg_read_clear_ns": "avg_read_clear.png",
    "avg_process_ns": "avg_process.png",
    "setup_ns_per_stream": "setup.png",
    "live_memory_growth_bytes_per_stream": "live_memory_growth_bytes.png",
    "values_read_per_sec_per_stream": "values_read_per_sec.png",
    "values_read_fresh_pct": "values_read_fresh_pct.png",
    "avg_tick_runstream_ns": "avg_tick_runstream.png",
}


def format_value(value: float, unit: str) -> str:
    if unit in {"ns", "ns/stream"}:
        suffix = "/stream" if unit.endswith("/stream") else ""
        if abs(value) >= 1_000_000_000:
            return f"{value / 1_000_000_000:.2f} s{suffix}"
        if abs(value) >= 1_000_000:
            return f"{value / 1_000_000:.2f} ms{suffix}"
        if abs(value) >= 1_000:
            return f"{value / 1_000:.2f} us{suffix}"
        return f"{value:.2f} ns{suffix}"
    if unit == "bytes/stream":
        if abs(value) >= 1024 * 1024:
            return f"{value / (1024 * 1024):.2f} MiB/stream"
        if abs(value) >= 1024:
            return f"{value / 1024:.2f} KiB/stream"
    if unit == "%":
        return f"{value:.2f}%"
    return f"{value:.2f} {unit}"


def status_for(thresholds: list[dict[str, Any]]) -> str:
    if any(t["status"] == "fail" for t in thresholds):
        return "fail"
    if any(t["status"] == "warn" for t in thresholds):
        return "warn"
    return "pass"


def status_label(status: str) -> str:
    return {"pass": "PASS", "warn": "WARN", "fail": "FAIL"}[status]


def metric_name(metric: str) -> str:
    return METRIC_NAMES.get(metric, metric.replace("_", " "))


def copy_relevant_plots(output_dir: str, thresholds: list[dict[str, Any]]) -> list[tuple[str, str]]:
    plots_dir = os.path.join(output_dir, "plots")
    selected_dir = os.path.join(output_dir, "regression-plots")
    if os.path.isdir(selected_dir):
        shutil.rmtree(selected_dir)
    os.makedirs(selected_dir, exist_ok=True)

    copied = []
    seen = set()
    for threshold in thresholds:
        if threshold["status"] == "pass":
            continue
        plot_name = THRESHOLD_TO_PLOT.get(threshold["metric"])
        if not plot_name or plot_name in seen:
            continue
        source = os.path.join(plots_dir, plot_name)
        if not os.path.exists(source):
            continue
        destination = os.path.join(selected_dir, plot_name)
        shutil.copy2(source, destination)
        copied.append((threshold["metric"], plot_name))
        seen.add(plot_name)
    return copied


def plot_url(args: argparse.Namespace, plot_name: str) -> str:
    if args.plots_base_url:
        return args.plots_base_url.rstrip("/") + f"/{plot_name}"
    artifact_url = args.artifact_url
    if not artifact_url:
        return f"benchmark-output/regression-plots/{plot_name}"
    return artifact_url.rstrip("/") + f"/regression-plots/{plot_name}"


def status_sentence(status: str) -> str:
    if status == "fail":
        return "At least one benchmark metric crossed a failure threshold. This job should block the PR until the regression is understood or the threshold is intentionally updated."
    if status == "warn":
        return "One or more benchmark metrics crossed a warning threshold. The job stays green, but the metrics below need attention."
    return "All benchmark thresholds passed."


def build_comment(result: dict[str, Any], thresholds: list[dict[str, Any]], copied_plots: list[tuple[str, str]], args: argparse.Namespace) -> str:
    status = status_for(thresholds)
    interesting = [t for t in thresholds if t["status"] != "pass"]
    threshold_by_metric = {t["metric"]: t for t in thresholds}
    scenarios = result.get("scenarios", [])
    streams = [str(s["streams"]) for s in scenarios]
    parameters = result.get("parameters", [])
    system = result.get("system", {})
    system_arch = "unknown"
    if system:
        os_name = system.get("os", "unknown")
        arch = system.get("arch", "unknown")
        cpus = system.get("cpus", "unknown")
        go_version = system.get("go_version", "unknown")
        system_arch = f"{os_name}/{arch}, {cpus} CPU(s), {go_version}"

    lines = [
        COMMENT_MARKER,
        "## Yamcs Stream Benchmark",
        "",
        f"**Status:** {status_label(status)}",
        "",
        status_sentence(status),
        "",
        "<details>",
        "<summary>Benchmark configuration</summary>",
        "",
        "| Setting | Value |",
        "|---|---:|",
        f"| Streams | `{', '.join(streams)}` |",
        f"| Parameters | `{len(parameters)}` |",
        f"| Duration | `{result.get('duration_seconds', 0):.2f}s` |",
        f"| Warmup | `{result.get('warmup_seconds', 0):.2f}s` |",
        f"| Read interval | `{result.get('read_interval_ms', 0)}ms` |",
        f"| Freshness window | `{result.get('freshness_ms', 0)}ms` |",
        f"| Simulator rate | `{result.get('simulator_rate', 'unknown')} Hz` |",
        f"| Yamcs | `{result.get('yamcs_address', 'unknown')}` |",
        f"| Instance / processor | `{result.get('instance', 'unknown')}` / `{result.get('processor', 'unknown')}` |",
        f"| System architecture | `{system_arch}` |",
    ]
    if args.run_url:
        lines.append(f"| Workflow run | [open run]({args.run_url}) |")
    if args.artifact_url:
        lines.append(f"| Full artifact | [download benchmark-output]({args.artifact_url}) |")
    lines.extend(["", "</details>"])

    if interesting:
        lines.extend(
            [
                "",
                "### Thresholds Needing Attention",
                "",
                "| Metric | Status | Observed | Warning threshold | Failure threshold | What it means |",
                "|---|---|---:|---:|---:|---|",
            ]
        )
        for threshold in interesting:
            lines.append(
                "| {metric} | {status} | {observed} | {warn} | {fail} | {detail} |".format(
                    metric=metric_name(threshold["metric"]),
                    status=threshold["status"].upper(),
                    observed=format_value(float(threshold["observed"]), threshold["unit"]),
                    warn=format_value(float(threshold["warn"]), threshold["unit"]),
                    fail=format_value(float(threshold["fail"]), threshold["unit"]),
                    detail=METRIC_DETAILS.get(threshold["metric"], ""),
                )
            )

        if copied_plots:
            lines.extend(["", "### Relevant Plots", ""])
            for metric, plot_name in copied_plots:
                threshold = threshold_by_metric.get(metric, {})
                image_url = plot_url(args, plot_name)
                lines.extend(
                    [
                        f"#### {metric_name(metric)}",
                        "",
                        "| Status | Observed | Warning threshold | Failure threshold |",
                        "|---|---:|---:|---:|",
                        "| {status} | {observed} | {warn} | {fail} |".format(
                            status=str(threshold.get("status", "")).upper(),
                            observed=format_value(float(threshold.get("observed", 0)), threshold.get("unit", "")),
                            warn=format_value(float(threshold.get("warn", 0)), threshold.get("unit", "")),
                            fail=format_value(float(threshold.get("fail", 0)), threshold.get("unit", "")),
                        ),
                        "",
                        f"[Open plot]({image_url})",
                        "",
                        f"![{metric_name(metric)}]({image_url})",
                        "",
                    ]
                )
            if not args.plots_base_url:
                lines.append("_If GitHub does not render artifact images inline, use the full artifact link above and open `regression-plots/`._")
    else:
        lines.extend(["", "All benchmark thresholds passed."])

    return "\n".join(lines).rstrip() + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate CI status/comment files for Yamcs benchmark results.")
    parser.add_argument("--output-dir", default="benchmark-output")
    parser.add_argument("--artifact-url", default="")
    parser.add_argument("--plots-base-url", default="")
    parser.add_argument("--run-url", default="")
    argv = sys.argv[1:]
    if argv and argv[0] == "--":
        argv = argv[1:]
    args = parser.parse_args(argv)

    results_path = os.path.join(args.output_dir, "yamcs-stream-results.json")
    with open(results_path, encoding="utf-8") as fp:
        result = json.load(fp)

    thresholds = result.get("thresholds", [])
    status = status_for(thresholds)
    copied_plots = copy_relevant_plots(args.output_dir, thresholds)
    comment = build_comment(result, thresholds, copied_plots, args)

    with open(os.path.join(args.output_dir, "benchmark-comment.md"), "w", encoding="utf-8") as fp:
        fp.write(comment)

    status_payload = {
        "status": status,
        "should_comment": status in {"warn", "fail"},
        "should_fail": status == "fail",
        "regression_plots": [plot for _, plot in copied_plots],
    }
    with open(os.path.join(args.output_dir, "benchmark-status.json"), "w", encoding="utf-8") as fp:
        json.dump(status_payload, fp, indent=2)

    print(json.dumps(status_payload))


if __name__ == "__main__":
    main()

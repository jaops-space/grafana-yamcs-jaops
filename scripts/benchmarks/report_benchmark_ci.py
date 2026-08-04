#!/usr/bin/env python3
import argparse
import json
import os
import shutil
import sys
from typing import Any

COMMENT_MARKER = "<!-- jaops-yamcs-benchmark-report -->"
METRIC_NAMES = {
    "avg_read_clear": "Average read and clear time",
    "avg_process": "Average Yamcs listener processing time",
    "setup": "Stream setup time",
    "setup_per_stream": "Setup time per stream",
    "live_memory_growth_bytes": "Live memory used during run",
    "live_memory_growth_bytes_per_stream": "Live memory per stream",
    "total_allocated_bytes": "Total memory allocated during run",
    "values_read_per_sec": "Values read per second from buffers",
    "values_read_per_sec_per_stream": "Values read per second from buffers per stream",
    "values_read_fresh_pct": "Values read within the same 1s tick",
    "avg_value_read_age": "Average value age when read",
    "avg_tick_runstream": "Average RunStream wall time per 1s tick",
}
METRIC_DETAILS = {
    "avg_read_clear": "Average time spent clearing one stream buffer.",
    "avg_process": "Average time spent processing one Yamcs parameter update.",
    "setup_per_stream": "Time spent creating stream demand state and Yamcs subscriptions.",
    "live_memory_growth_bytes_per_stream": "Additional live memory used per stream during the run.",
    "values_read_per_sec_per_stream": "Per-stream buffer throughput against the 1 Hz simulator cadence.",
    "values_read_fresh_pct": "Share of values read before the next 1 second simulator update.",
    "avg_tick_runstream": "Average wall-clock time for all RunStream read/frame/send work during each 1 second tick.",
}
THRESHOLD_TO_PLOT = {
    "avg_read_clear": "avg_read_clear.png",
    "avg_process": "avg_process.png",
    "setup": "setup.png",
    "setup_per_stream": "setup.png",
    "live_memory_growth_bytes": "live_memory_growth_bytes.png",
    "live_memory_growth_bytes_per_stream": "live_memory_growth_bytes.png",
    "total_allocated_bytes": "total_allocated_bytes.png",
    "values_read_per_sec": "values_read_per_sec.png",
    "values_read_per_sec_per_stream": "values_read_per_sec.png",
    "values_read_fresh_pct": "values_read_fresh_pct.png",
    "avg_value_read_age": "avg_value_read_age.png",
    "avg_tick_runstream": "avg_tick_runstream.png",
}
PLOT_TO_METRIC = {
    "avg_read_clear.png": "avg_read_clear",
    "avg_process.png": "avg_process",
    "live_memory_growth_bytes.png": "live_memory_growth_bytes",
    "total_allocated_bytes.png": "total_allocated_bytes",
    "values_read_per_sec.png": "values_read_per_sec",
    "values_read_fresh_pct.png": "values_read_fresh_pct",
    "avg_value_read_age.png": "avg_value_read_age",
    "avg_tick_runstream.png": "avg_tick_runstream",
    "setup.png": "setup",
}
PLOT_ORDER = list(PLOT_TO_METRIC.keys())


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
    if unit in {"bytes", "bytes/stream"}:
        suffix = "/stream" if unit.endswith("/stream") else ""
        if abs(value) >= 1024 * 1024:
            return f"{value / (1024 * 1024):.2f} MiB{suffix}"
        if abs(value) >= 1024:
            return f"{value / 1024:.2f} KiB{suffix}"
        return f"{value:.2f} bytes{suffix}"
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


def list_all_plots(output_dir: str) -> list[tuple[str, str]]:
    plots_dir = os.path.join(output_dir, "plots")
    if not os.path.isdir(plots_dir):
        return []
    names = [name for name in os.listdir(plots_dir) if name.endswith(".png")]
    names.sort(key=lambda name: (PLOT_ORDER.index(name) if name in PLOT_ORDER else len(PLOT_ORDER), name))
    return [(PLOT_TO_METRIC.get(name, os.path.splitext(name)[0]), name) for name in names]


def plot_url(args: argparse.Namespace, plot_name: str, directory: str = "plots") -> str:
    if args.plots_base_url:
        return args.plots_base_url.rstrip("/") + f"/{plot_name}"
    return f"benchmark-output/{directory}/{plot_name}"


def status_sentence(status: str) -> str:
    if status == "fail":
        return "At least one benchmark metric crossed a failure threshold. This job should block the PR until the regression is understood or the threshold is intentionally updated."
    if status == "warn":
        return "One or more benchmark metrics crossed a warning threshold. The job stays green, but the metrics below need attention."
    return "All benchmark thresholds passed."


def format_change(value: float) -> str:
    return f"{value:+.1f}%"


def average_change_title(summaries: list[dict[str, Any]]) -> str:
    for summary in summaries:
        if summary.get("baseline") == "PR base":
            return f" - avg vs base {format_change(float(summary.get('avg_change_pct', 0)))}"
    return ""


def build_comment(
    result: dict[str, Any],
    thresholds: list[dict[str, Any]],
    copied_plots: list[tuple[str, str]],
    all_plots: list[tuple[str, str]],
    args: argparse.Namespace,
) -> str:
    status = status_for(thresholds)
    interesting = [t for t in thresholds if t["status"] != "pass"]
    threshold_by_metric = {t["metric"]: t for t in thresholds}
    summaries_by_metric: dict[str, list[dict[str, Any]]] = {}
    for summary in result.get("baseline_change_summaries", []):
        metric = summary.get("metric")
        if isinstance(metric, str):
            summaries_by_metric.setdefault(metric, []).append(summary)
    scenarios = result.get("scenarios", [])
    streams = [str(s["streams"]) for s in scenarios]
    parameters = result.get("parameters", [])
    system = result.get("system", {})
    baseline = result.get("baseline", {})
    long_term_baseline = result.get("long_term_baseline", {})
    system_arch = "unknown"
    if system:
        os_name = system.get("os", "unknown")
        arch = system.get("arch", "unknown")
        cpus = system.get("cpus", "unknown")
        cpu_model = system.get("cpu_model") or "unknown CPU"
        cpu_frequency = system.get("cpu_frequency_mhz")
        go_version = system.get("go_version", "unknown")
        frequency_text = f"{float(cpu_frequency) / 1000:.2f} GHz" if isinstance(cpu_frequency, (int, float)) and cpu_frequency > 0 else "unknown frequency"
        system_arch = f"{os_name}/{arch}, {cpus} CPU(s), {cpu_model}, {frequency_text}, {go_version}"

    lines = [
        COMMENT_MARKER,
        "## Performance Benchmark",
        "",
        f"**Status:** {status_label(status)}",
        "",
        "**Scenario:** Yamcs quickstart simulator at 1 Hz with Grafana streams reading buffers on 1s tickers.",
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
        f"| Simulator / stream ticker | `{result.get('simulator_rate', 'unknown')} Hz` / `{result.get('read_interval_ms', 0)}ms` |",
        f"| Instance / processor | `{result.get('instance', 'unknown')}` / `{result.get('processor', 'unknown')}` |",
        f"| System architecture | `{system_arch}` |",
        f"| PR base baseline | `{baseline.get('message', 'not requested')}` |",
        f"| Long-term baseline | `{long_term_baseline.get('message', 'not available')}` |",
    ]
    if args.run_url:
        lines.append(f"| Workflow run | [open run]({args.run_url}) |")
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

    if all_plots:
        relevant_plot_names = {plot_name for _, plot_name in copied_plots}
        lines.extend(
            [
                "",
                "### Benchmark Plots",
                "",
                "Blue is the PR result. Green is the base commit before the PR changes. Orange is the checked-in long-term baseline when available.",
                "",
            ]
        )
        for metric, plot_name in all_plots:
            threshold = threshold_by_metric.get(metric)
            baseline_summaries = summaries_by_metric.get(metric, [])
            image_url = plot_url(args, plot_name)
            average_change = average_change_title(baseline_summaries)
            marker = " - needs attention" if plot_name in relevant_plot_names else ""
            lines.extend(["<details>", f"<summary>{metric_name(metric)}{average_change}{marker}</summary>", ""])
            if threshold:
                lines.extend(
                    [
                        "| Status | Observed | Warning threshold | Failure threshold |",
                        "|---|---:|---:|---:|",
                        "| {status} | {observed} | {warn} | {fail} |".format(
                            status=str(threshold.get("status", "")).upper(),
                            observed=format_value(float(threshold.get("observed", 0)), threshold.get("unit", "")),
                            warn=format_value(float(threshold.get("warn", 0)), threshold.get("unit", "")),
                            fail=format_value(float(threshold.get("fail", 0)), threshold.get("unit", "")),
                        ),
                        "",
                    ]
                )
            if baseline_summaries:
                lines.extend(
                    [
                        "| Reference | Samples | Average change | Min change | Max change |",
                        "|---|---:|---:|---:|---:|",
                    ]
                )
                for summary in baseline_summaries:
                    lines.append(
                        "| {baseline} | {samples} | {avg} | {min} | {max} |".format(
                            baseline=summary.get("baseline", "baseline"),
                            samples=int(summary.get("samples", 0)),
                            avg=format_change(float(summary.get("avg_change_pct", 0))),
                            min=format_change(float(summary.get("min_change_pct", 0))),
                            max=format_change(float(summary.get("max_change_pct", 0))),
                        )
                    )
                lines.extend(
                    [
                        "",
                    ]
                )
            lines.extend([f"![{metric_name(metric)}]({image_url})", "", "</details>", ""])
    elif not interesting:
        lines.extend(["", "All benchmark thresholds passed."])
    else:
        lines.extend(["", "No benchmark plots were selected for this warning."])

    lines.extend(["", "_This comment is updated with the latest benchmark result._"])
    return "\n".join(lines).rstrip() + "\n"


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate CI status/comment files for benchmark results.")
    parser.add_argument("--output-dir", default="benchmark-output")
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
    all_plots = list_all_plots(args.output_dir)
    comment = build_comment(result, thresholds, copied_plots, all_plots, args)

    with open(os.path.join(args.output_dir, "benchmark-comment.md"), "w", encoding="utf-8") as fp:
        fp.write(comment)

    status_payload = {
        "status": status,
        "should_comment": True,
        "should_fail": status == "fail",
        "regression_plots": [plot for _, plot in copied_plots],
        "plots": [plot for _, plot in all_plots],
    }
    with open(os.path.join(args.output_dir, "benchmark-status.json"), "w", encoding="utf-8") as fp:
        json.dump(status_payload, fp, indent=2)

    print(json.dumps(status_payload))


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
import argparse
import json
import os
import shutil
import sys
from datetime import datetime
from typing import Any

from common import yamcs as bench
from common import grafana as grafana_report
from common.metrics import METRIC_NAMES, PLOT_ORDER, PLOT_TO_METRIC, THRESHOLD_TO_PLOT
from common.plotting import format_baseline_change_rows, format_change, format_value, status_emoji, status_for, status_label

COMMENT_MARKER = "<!-- jaops-yamcs-benchmark-report -->"
METRIC_DETAILS = {
    "setup_per_stream": "Time spent creating stream demand state and Yamcs subscriptions.",
    "live_memory_growth_bytes_per_stream": "Additional live memory used per stream during the run.",
    "values_read_per_sec_per_stream": "Per-stream buffer throughput against the 1 Hz simulator cadence.",
    "values_read_fresh_pct": "Share of values read before the next 1 second simulator update.",
    "median_tick_runstream_busy": "Median total read/frame/send work done by RunStream goroutines during each 1 second tick.",
}


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
    names = [name for name in os.listdir(plots_dir) if name in PLOT_TO_METRIC]
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


def summary_change_value(summary: dict[str, Any]) -> float:
    return float(summary.get("median_change_pct", 0))


def median_change_title(summaries: list[dict[str, Any]]) -> str:
    for summary in summaries:
        if summary.get("baseline") == "PR base":
            return f" - median vs base {format_change(summary_change_value(summary))}"
    return ""


def commit_link(value: Any) -> str:
    if not isinstance(value, str) or not value:
        return ""
    server_url = os.environ.get("GITHUB_SERVER_URL", "https://github.com").rstrip("/")
    repository = os.environ.get("GITHUB_REPOSITORY", "")
    if not repository:
        return f"`{value}`"
    return f"[`{value}`]({server_url}/{repository}/commit/{value})"


def format_datetime(value: Any) -> str:
    if not isinstance(value, str) or not value:
        return ""
    normalized = value.replace("Z", "+00:00")
    try:
        parsed = datetime.fromisoformat(normalized)
    except ValueError:
        return value
    return parsed.strftime("%Y-%m-%d %H:%M UTC")


def format_frequency(value: Any) -> str:
    return f"{float(value) / 1000:.2f} GHz" if isinstance(value, (int, float)) and value > 0 else "unknown frequency"


def format_system_arch(system: Any) -> str:
    if not isinstance(system, dict) or not system:
        return "unknown"
    os_name = system.get("os", "unknown")
    arch = system.get("arch", "unknown")
    cpus = system.get("available_logical_cpus", system.get("cpus", "unknown"))
    cpu_model = system.get("cpu_model") or "unknown CPU"
    go_version = system.get("go_version", "unknown")
    parts = [f"{os_name}/{arch}", f"{cpus} available logical CPU(s)", str(cpu_model)]
    if "ghz" not in str(cpu_model).lower():
        parts.append(format_frequency(system.get("cpu_frequency_mhz")))
    parts.append(str(go_version))
    return ", ".join(parts)


def format_long_term_baseline(value: dict[str, Any]) -> str:
    metadata = value.get("metadata", {})
    if not value.get("compatible") or not isinstance(metadata, dict) or not metadata:
        return str(value.get("message", "not available"))

    parts = []
    created_at = format_datetime(metadata.get("created_at"))
    if created_at:
        parts.append(created_at)
    quickstart = metadata.get("yamcs_quickstart")
    if isinstance(quickstart, str) and quickstart:
        parts.append(quickstart)
    simulator_rate = metadata.get("simulator_rate_hz")
    stream_interval = metadata.get("stream_read_interval")
    if simulator_rate or stream_interval:
        parts.append(f"{simulator_rate or 'unknown'} Hz simulator / {stream_interval or 'unknown'} stream ticker")
    parameter_count = metadata.get("parameter_count")
    if isinstance(parameter_count, int):
        parts.append(f"{parameter_count} parameters")
    system = format_system_arch(value.get("system", {}))
    if system != "unknown":
        parts.append(system)
    return "; ".join(parts) if parts else str(value.get("message", "not available"))


def parameter_count_for(simulator: dict[str, Any] | None, grafana: dict[str, Any] | None) -> int:
    if isinstance(simulator, dict) and isinstance(simulator.get("parameters"), list):
        return len(simulator["parameters"])
    if isinstance(grafana, dict) and isinstance(grafana.get("parameters"), int):
        return int(grafana["parameters"])
    return 0


def config_value(value: Any, fallback: str = "—") -> str:
    if value is None or value == "":
        return fallback
    return str(value)


def benchmark_config(
    simulator: dict[str, Any] | None,
    grafana: dict[str, Any] | None,
    reference: str = "",
) -> dict[str, str]:
    simulator = simulator or {}
    return {
        "Reference": reference or "current",
        "Parameters": str(parameter_count_for(simulator, grafana)),
        "Simulator scenario duration": f"{float(simulator.get('duration_seconds', 0)):.2f}s",
        "Simulator scenario warmup": f"{float(simulator.get('warmup_seconds', 0)):.2f}s",
        "Simulator / stream ticker": (
            f"{config_value(simulator.get('simulator_rate'), 'unknown')} Hz / "
            f"{config_value(simulator.get('read_interval_ms'), '0')}ms"
        ),
        "Instance / processor": (
            f"{config_value(simulator.get('instance'), 'unknown')} / "
            f"{config_value(simulator.get('processor'), 'unknown')}"
        ),
        "System architecture": format_system_arch(simulator.get("system", {})),
    }


def append_config_table(
    lines: list[str],
    head_config: dict[str, str],
    baseline_config: dict[str, str] | None,
    long_term_config: dict[str, str] | None,
    run_url: str,
) -> None:
    def cell(value: str) -> str:
        if value == "—":
            return value
        if value.startswith("[") or value.startswith("`"):
            return value
        return f"`{value}`"

    baseline_columns: list[tuple[str, dict[str, str]]] = []
    if baseline_config:
        baseline_columns.append(("PR base", baseline_config))
    if long_term_config:
        baseline_columns.append(("Long-term baseline", long_term_config))

    if not baseline_columns:
        lines.extend(["| Setting | Value |", "|---|---:|"])
        for key, value in head_config.items():
            lines.append(f"| {key} | {cell(value)} |")
        if run_url:
            lines.append(f"| Workflow run | [open run]({run_url}) |")
        return

    headers = ["Setting", "HEAD"] + [label for label, _config in baseline_columns]
    lines.append("| " + " | ".join(headers) + " |")
    lines.append("|" + "|".join(["---"] + ["---:" for _ in headers[1:]]) + "|")
    for key, value in head_config.items():
        row = [key, cell(value)]
        for _label, config in baseline_columns:
            row.append(cell(config.get(key, "—")))
        lines.append("| " + " | ".join(row) + " |")
    if run_url:
        row = ["Workflow run", f"[open run]({run_url})"] + ["—" for _label, _config in baseline_columns]
        lines.append("| " + " | ".join(row) + " |")


def build_comment(
    result: dict[str, Any],
    grafana: dict[str, Any] | None,
    baseline_simulator: dict[str, Any] | None,
    baseline_grafana: dict[str, Any] | None,
    long_term_simulator: dict[str, Any] | None,
    long_term_grafana: dict[str, Any] | None,
    thresholds: list[dict[str, Any]],
    copied_plots: list[tuple[str, str]],
    all_plots: list[tuple[str, str]],
    args: argparse.Namespace,
    extra_plot_sections: str = "",
    status_thresholds: list[dict[str, Any]] | None = None,
) -> str:
    status_thresholds = status_thresholds if status_thresholds is not None else thresholds
    status = status_for(status_thresholds)
    interesting = [t for t in status_thresholds if t["status"] != "pass"]
    thresholds_by_plot: dict[str, list[dict[str, Any]]] = {}
    for threshold in thresholds:
        plot_name = THRESHOLD_TO_PLOT.get(str(threshold.get("metric")))
        if plot_name:
            thresholds_by_plot.setdefault(plot_name, []).append(threshold)
    summaries_by_metric: dict[str, list[dict[str, Any]]] = {}
    for summary in result.get("baseline_change_summaries", []):
        metric = summary.get("metric")
        if isinstance(metric, str):
            summaries_by_metric.setdefault(metric, []).append(summary)
    baseline = result.get("baseline", {})
    long_term_baseline = result.get("long_term_baseline", {})
    head_config = benchmark_config(result, grafana)
    baseline_config = None
    if baseline.get("compatible") or baseline_grafana:
        reference = commit_link(baseline.get("commit")) if baseline.get("commit") else str(baseline.get("message", "loaded"))
        baseline_config = benchmark_config(baseline_simulator, baseline_grafana, reference)
    long_term_config = None
    if long_term_baseline.get("compatible") or long_term_grafana:
        long_term_config = benchmark_config(long_term_simulator, long_term_grafana, format_long_term_baseline(long_term_baseline))

    lines = [
        COMMENT_MARKER,
        "## Performance Benchmark",
        "",
        f"**Status:** {status_emoji(status)} `{status_label(status)}`",
        "",
        status_sentence(status),
        "",
        "<details>",
        "<summary>Benchmark configuration</summary>",
        "",
    ]
    append_config_table(lines, head_config, baseline_config, long_term_config, args.run_url)
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
                    status=f"{status_emoji(threshold['status'])} `{threshold['status'].upper()}`",
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
            ]
        )
        for metric, plot_name in all_plots:
            plot_thresholds = thresholds_by_plot.get(plot_name, [])
            baseline_summaries = summaries_by_metric.get(metric, [])
            image_url = plot_url(args, plot_name)
            median_change = median_change_title(baseline_summaries)
            metric_status = status_for(plot_thresholds) if plot_thresholds else "pass"
            marker = " - needs attention" if plot_name in relevant_plot_names else ""
            lines.extend(["<details>", f"<summary>{status_emoji(metric_status)} {metric_name(metric)}{median_change}{marker}</summary>", ""])
            if plot_thresholds:
                lines.extend(
                    [
                        "| Status | Observed | Warning threshold | Failure threshold |",
                        "|---|---:|---:|---:|",
                    ]
                )
                for threshold in plot_thresholds:
                    lines.append(
                        "| {status} | {observed} | {warn} | {fail} |".format(
                            status=f"{status_emoji(str(threshold.get('status', '')))} `{str(threshold.get('status', '')).upper()}`",
                            observed=format_value(float(threshold.get("observed", 0)), threshold.get("unit", "")),
                            warn=format_value(float(threshold.get("warn", 0)), threshold.get("unit", "")),
                            fail=format_value(float(threshold.get("fail", 0)), threshold.get("unit", "")),
                        )
                    )
                lines.append("")
            if baseline_summaries:
                lines.extend(format_baseline_change_rows(baseline_summaries))
            lines.extend([f"![{metric_name(metric)}]({image_url})", "", "</details>", ""])
    elif not interesting:
        lines.extend(["", "All benchmark thresholds passed."])
    else:
        lines.extend(["", "No benchmark plots were selected for this warning."])

    if extra_plot_sections.strip():
        lines.extend(["", extra_plot_sections.strip()])

    lines.extend(["", "_This comment is updated with the latest benchmark result._"])
    return "\n".join(lines).rstrip() + "\n"


def load_result(path: str) -> dict[str, Any] | None:
    if not path or not os.path.exists(path):
        return None
    with open(path, encoding="utf-8") as fp:
        value = json.load(fp)
    return value if isinstance(value, dict) else None


def compatible_rows(result: dict[str, Any] | None, key: str) -> list[dict[str, Any]] | None:
    if not result or not bench.metric_semantics_compatible(result):
        return None
    rows = result.get(key, [])
    return rows if isinstance(rows, list) else None


def baseline_metadata(result: dict[str, Any] | None, path: str, commit: str = "") -> dict[str, Any]:
    compatible = bench.metric_semantics_compatible(result)
    message = "loaded" if result else "not provided"
    if result and not compatible:
        message = "baseline loaded but metric semantics version is incompatible"
    return {
        "compatible": compatible,
        "path": path,
        "commit": commit,
        "message": message,
        "metric_semantics_version": result.get("metric_semantics_version") if result else None,
    }


def long_term_metadata(simulator_result: dict[str, Any] | None) -> dict[str, Any]:
    compatible = bench.metric_semantics_compatible(simulator_result)
    message = "loaded" if simulator_result else "not provided"
    if simulator_result and not compatible:
        message = "long-term baseline loaded but metric semantics version is incompatible"
    return {
        "compatible": compatible,
        "path": os.path.relpath(bench.LONG_TERM_SIMULATOR_RESULTS, os.getcwd()),
        "metadata": bench.load_json_file(os.path.join(bench.LONG_TERM_BASELINE_DIR, "metadata.json")) or {},
        "system": simulator_result.get("system", {}) if simulator_result else {},
        "environment": {
            "yamcs_address": simulator_result.get("yamcs_address", "") if simulator_result else "",
            "instance": simulator_result.get("instance", "") if simulator_result else "",
            "processor": simulator_result.get("processor", "") if simulator_result else "",
        },
        "message": message,
        "metric_semantics_version": simulator_result.get("metric_semantics_version") if simulator_result else None,
    }


def combine_results(
    simulator: dict[str, Any] | None,
    micro: dict[str, Any] | None,
    baseline_simulator: dict[str, Any] | None,
    baseline_micro: dict[str, Any] | None,
    long_term_simulator: dict[str, Any] | None,
    long_term_micro: dict[str, Any] | None,
    baseline_simulator_path: str,
    baseline_commit: str,
) -> dict[str, Any]:
    result: dict[str, Any] = {}
    if simulator:
        result.update(simulator)
    elif micro:
        result.update(micro)
    result["scenarios"] = simulator.get("scenarios", []) if simulator else []
    result["microbenchmarks"] = micro.get("microbenchmarks", []) if micro else []
    if micro and not result.get("parameters"):
        result["parameters"] = micro.get("parameters", [])
    result["thresholds"] = simulator.get("thresholds", []) if simulator else []
    result["baseline"] = baseline_metadata(baseline_simulator, baseline_simulator_path, baseline_commit)
    result["long_term_baseline"] = long_term_metadata(long_term_simulator)
    result["metric_semantics_version"] = bench.METRIC_SEMANTICS_VERSION
    result["baseline_change_summaries"] = []
    result["baseline_change_summaries"] += bench.summarize_baseline_changes(
        result["scenarios"], compatible_rows(baseline_simulator, "scenarios"), "PR base"
    )
    result["baseline_change_summaries"] += bench.summarize_baseline_changes(
        result["scenarios"], compatible_rows(long_term_simulator, "scenarios"), "Long-term baseline"
    )
    result["baseline_change_summaries"] += bench.summarize_micro_baseline_changes(
        result["microbenchmarks"], compatible_rows(baseline_micro, "microbenchmarks"), "PR base"
    )
    result["baseline_change_summaries"] += bench.summarize_micro_baseline_changes(
        result["microbenchmarks"], compatible_rows(long_term_micro, "microbenchmarks"), "Long-term baseline"
    )
    return result


def copy_result(source: str, destination: str) -> None:
    if not source or not os.path.exists(source):
        return
    os.makedirs(os.path.dirname(destination), exist_ok=True)
    if os.path.abspath(source) != os.path.abspath(destination):
        shutil.copy2(source, destination)


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate CI status/comment files for benchmark results.")
    parser.add_argument("--output-dir", default="benchmark-output")
    parser.add_argument("--micro-results", default="")
    parser.add_argument("--simulator-results", default="")
    parser.add_argument("--grafana-results", default="")
    parser.add_argument("--baseline-micro-results", default="")
    parser.add_argument("--baseline-simulator-results", default="")
    parser.add_argument("--baseline-grafana-results", default="")
    parser.add_argument("--baseline-commit", default="")
    parser.add_argument("--long-term-micro-results", default=bench.LONG_TERM_MICRO_RESULTS)
    parser.add_argument("--long-term-simulator-results", default=bench.LONG_TERM_SIMULATOR_RESULTS)
    parser.add_argument("--long-term-grafana-results", default="scripts/benchmarks/baselines/long-term/grafana.json")
    parser.add_argument("--plots-base-url", default="")
    parser.add_argument("--run-url", default="")
    argv = sys.argv[1:]
    if argv and argv[0] == "--":
        argv = argv[1:]
    args = parser.parse_args(argv)

    simulator_path = args.simulator_results or os.path.join(args.output_dir, "simulator", "simulator.json")
    micro_path = args.micro_results or os.path.join(args.output_dir, "microbenchmarks", "microbenchmarks.json")
    grafana_path = args.grafana_results or os.path.join(args.output_dir, "grafana", "grafana.json")
    baseline_simulator_path = args.baseline_simulator_results or os.path.join(
        args.output_dir, "baselines", "pr", "simulator.json"
    )
    baseline_micro_path = args.baseline_micro_results or os.path.join(args.output_dir, "baselines", "pr", "microbenchmarks.json")
    baseline_grafana_path = args.baseline_grafana_results or os.path.join(args.output_dir, "baselines", "pr", "grafana.json")

    simulator = load_result(simulator_path)
    micro = load_result(micro_path)
    grafana = load_result(grafana_path)
    baseline_simulator = load_result(baseline_simulator_path)
    baseline_micro = load_result(baseline_micro_path)
    baseline_grafana = load_result(baseline_grafana_path)
    long_term_simulator = load_result(args.long_term_simulator_results)
    long_term_micro = load_result(args.long_term_micro_results)
    long_term_grafana = load_result(args.long_term_grafana_results)

    os.makedirs(args.output_dir, exist_ok=True)
    copy_result(simulator_path, os.path.join(args.output_dir, "simulator", "simulator.json"))
    copy_result(simulator_path.replace(".json", ".csv"), os.path.join(args.output_dir, "simulator", "simulator.csv"))
    copy_result(micro_path, os.path.join(args.output_dir, "microbenchmarks", "microbenchmarks.json"))
    copy_result(grafana_path, os.path.join(args.output_dir, "grafana", "grafana.json"))
    copy_result(baseline_simulator_path, os.path.join(args.output_dir, "baselines", "pr", "simulator.json"))
    copy_result(baseline_micro_path, os.path.join(args.output_dir, "baselines", "pr", "microbenchmarks.json"))
    copy_result(baseline_grafana_path, os.path.join(args.output_dir, "baselines", "pr", "grafana.json"))

    result = combine_results(
        simulator,
        micro,
        baseline_simulator,
        baseline_micro,
        long_term_simulator,
        long_term_micro,
        baseline_simulator_path,
        args.baseline_commit,
    )
    plots_dir = os.path.join(args.output_dir, "plots")
    os.makedirs(plots_dir, exist_ok=True)
    for name in os.listdir(plots_dir):
        if name.endswith(".png"):
            os.remove(os.path.join(plots_dir, name))

    baseline_simulator_rows = compatible_rows(baseline_simulator, "scenarios")
    baseline_micro_rows = compatible_rows(baseline_micro, "microbenchmarks")
    long_term_simulator_rows = compatible_rows(long_term_simulator, "scenarios")
    long_term_micro_rows = compatible_rows(long_term_micro, "microbenchmarks")
    if result["scenarios"]:
        bench.plot_all_metrics(args.output_dir, result["scenarios"], baseline_simulator_rows, long_term_simulator_rows)
    if result["microbenchmarks"]:
        bench.plot_all_micro_metrics(args.output_dir, result["microbenchmarks"], baseline_micro_rows, long_term_micro_rows)
    grafana_thresholds: list[dict[str, Any]] = []
    grafana_section = ""
    if grafana:
        for metric in grafana_report.METRICS:
            grafana_report.plot_metric(plots_dir, grafana, baseline_grafana, long_term_grafana, metric)
        grafana_thresholds = list(grafana_report.evaluate_thresholds(grafana).values())
        grafana_section = grafana_report.build_comment(grafana, baseline_grafana, long_term_grafana, args.plots_base_url)

    thresholds = result.get("thresholds", [])
    all_thresholds = thresholds + grafana_thresholds
    status = status_for(all_thresholds)
    copied_plots = copy_relevant_plots(args.output_dir, all_thresholds)
    all_plots = list_all_plots(args.output_dir)
    main_plots = [(metric, plot) for metric, plot in all_plots if not plot.startswith("grafana_")]
    comment = build_comment(
        result,
        grafana,
        baseline_simulator,
        baseline_grafana,
        long_term_simulator,
        long_term_grafana,
        thresholds,
        copied_plots,
        main_plots,
        args,
        grafana_section,
        all_thresholds,
    )

    with open(os.path.join(args.output_dir, "benchmark-comment.md"), "w", encoding="utf-8") as fp:
        fp.write(comment)

    plots_dir = os.path.join(args.output_dir, "plots")
    plot_files = sorted([name for name in os.listdir(plots_dir) if name.endswith(".png")]) if os.path.isdir(plots_dir) else []
    status_payload = {
        "status": status,
        "should_comment": True,
        "should_fail": status == "fail",
        "regression_plots": [plot for _, plot in copied_plots],
        "plots": plot_files,
        "thresholds": all_thresholds,
    }
    with open(os.path.join(args.output_dir, "benchmark-status.json"), "w", encoding="utf-8") as fp:
        json.dump(status_payload, fp, indent=2)

    print(json.dumps(status_payload))


if __name__ == "__main__":
    main()

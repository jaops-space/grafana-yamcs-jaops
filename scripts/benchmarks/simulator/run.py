#!/usr/bin/env python3
import argparse
import json
import os
import sys
import time
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import yamcs as shared


def main() -> None:
    parser = argparse.ArgumentParser(description="Run the Yamcs simulator benchmark scenario only.")
    parser.add_argument("--output-dir", default="benchmark-output/simulator")
    parser.add_argument("--yamcs-address", default="localhost:8090")
    parser.add_argument("--instance", default="myproject")
    parser.add_argument("--processor", default="realtime")
    parser.add_argument("--streams", type=shared.parse_streams, default=shared.parse_streams(shared.DEFAULT_STREAMS))
    parser.add_argument("--parameters", default=shared.DEFAULT_PARAMETERS)
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
    parser.add_argument("--baseline-results", default="", help="Optional previous simulator JSON to compare against")
    parser.add_argument("--baseline-commit", default="", help="Commit hash used to produce the PR base simulator benchmark")
    parser.add_argument("--allow-local-baseline", action="store_true")
    parser.add_argument("--fail-on-threshold", action="store_true")
    argv = sys.argv[1:]
    if argv and argv[0] == "--":
        argv = argv[1:]
    args = parser.parse_args(argv)

    refreshing_long_term = os.path.abspath(args.output_dir) == os.path.abspath(shared.LONG_TERM_BASELINE_DIR)
    if refreshing_long_term and os.environ.get("CI") != "true" and not args.allow_local_baseline:
        raise SystemExit(
            "Refusing to refresh the checked-in long-term baseline outside CI. "
            "Use the GitHub Actions baseline refresh workflow, or pass --allow-local-baseline for a local diagnostic run."
        )

    os.makedirs(args.output_dir, exist_ok=True)
    started_at = datetime.now(timezone.utc).isoformat()
    simulator = shared.run_simulator(args)
    try:
        if simulator is not None:
            time.sleep(2)
        result = shared.run_go_scenario(args, args.streams)
    finally:
        shared.stop_process(simulator)

    result["microbenchmarks"] = []
    result["python_started_at"] = started_at
    result["python_finished_at"] = datetime.now(timezone.utc).isoformat()
    result["simulator_rate"] = args.simulator_rate
    result["metric_semantics_version"] = shared.METRIC_SEMANTICS_VERSION
    threshold_results = shared.evaluate_thresholds(result["scenarios"])

    baseline_result, baseline_message = shared.load_baseline_results(args.baseline_results)
    baseline_compatible = shared.metric_semantics_compatible(baseline_result)
    if baseline_result and not baseline_compatible:
        baseline_message = "baseline loaded but metric semantics version is incompatible"
    baseline_rows = baseline_result.get("scenarios", []) if baseline_compatible else None

    if refreshing_long_term:
        long_term_result = None
        long_term_message = "long-term baseline skipped while refreshing it"
    else:
        long_term_result, long_term_message = shared.load_baseline_results(shared.LONG_TERM_SIMULATOR_RESULTS)
    long_term_compatible = shared.metric_semantics_compatible(long_term_result)
    if long_term_result and not long_term_compatible:
        long_term_message = "long-term baseline loaded but metric semantics version is incompatible"
    long_term_rows = long_term_result.get("scenarios", []) if long_term_compatible else None

    result["baseline"] = {
        "compatible": baseline_compatible,
        "path": args.baseline_results,
        "commit": args.baseline_commit,
        "message": baseline_message,
        "metric_semantics_version": baseline_result.get("metric_semantics_version") if baseline_result else None,
    }
    result["long_term_baseline"] = {
        "compatible": long_term_compatible,
        "path": os.path.relpath(shared.LONG_TERM_SIMULATOR_RESULTS, os.getcwd()),
        "metadata": shared.load_json_file(os.path.join(shared.LONG_TERM_BASELINE_DIR, "metadata.json")) or {},
        "system": long_term_result.get("system", {}) if long_term_result else {},
        "environment": {
            "yamcs_address": long_term_result.get("yamcs_address", "") if long_term_result else "",
            "instance": long_term_result.get("instance", "") if long_term_result else "",
            "processor": long_term_result.get("processor", "") if long_term_result else "",
        },
        "message": long_term_message,
        "metric_semantics_version": long_term_result.get("metric_semantics_version") if long_term_result else None,
    }
    result["baseline_change_summaries"] = shared.summarize_baseline_changes(result["scenarios"], baseline_rows, "PR base")
    result["baseline_change_summaries"] += shared.summarize_baseline_changes(
        result["scenarios"], long_term_rows, "Long-term baseline"
    )
    result["thresholds"] = threshold_results

    output_path = os.path.join(args.output_dir, "simulator.json")
    csv_path = os.path.join(args.output_dir, "simulator.csv")
    with open(output_path, "w", encoding="utf-8") as fp:
        json.dump(result, fp, indent=2)
        fp.write("\n")
    shared.write_csv(csv_path, result["scenarios"])
    if refreshing_long_term:
        shared.write_long_term_metadata(args, result)
    plot_paths = shared.plot_all_metrics(args.output_dir, result["scenarios"], baseline_rows, long_term_rows)

    print("=== Yamcs Simulator Benchmark ===")
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

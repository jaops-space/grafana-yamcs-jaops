#!/usr/bin/env python3
import argparse
import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from common import yamcs as shared


def main() -> None:
    parser = argparse.ArgumentParser(description="Run JAOPS microbenchmarks only.")
    parser.add_argument("--output-dir", default="benchmark-output/microbenchmarks")
    parser.add_argument("--parameters", default=shared.DEFAULT_PARAMETERS)
    parser.add_argument("--baseline-results", default="", help="Optional previous microbenchmark JSON to compare against")
    parser.add_argument("--baseline-commit", default="", help="Commit hash used to produce the PR base microbenchmark")
    argv = sys.argv[1:]
    if argv and argv[0] == "--":
        argv = argv[1:]
    args = parser.parse_args(argv)

    os.makedirs(args.output_dir, exist_ok=True)
    started_at = datetime.now(timezone.utc).isoformat()
    result = {
        "scenarios": [],
        "microbenchmarks": shared.run_go_microbenchmarks(args),
        "parameters": [part for part in args.parameters.split(",") if part],
        "instance": "",
        "processor": "",
        "duration_seconds": 0,
        "warmup_seconds": 0,
        "read_interval_ms": 0,
        "system": {},
        "python_started_at": started_at,
        "python_finished_at": datetime.now(timezone.utc).isoformat(),
        "simulator_rate": 0,
        "metric_semantics_version": shared.METRIC_SEMANTICS_VERSION,
        "thresholds": [],
    }

    baseline_result, baseline_message = shared.load_baseline_results(args.baseline_results)
    baseline_compatible = shared.metric_semantics_compatible(baseline_result)
    if baseline_result and not baseline_compatible:
        baseline_message = "baseline loaded but metric semantics version is incompatible"
    baseline_micro_rows = baseline_result.get("microbenchmarks", []) if baseline_compatible else None

    long_term_result, long_term_message = shared.load_baseline_results(shared.LONG_TERM_MICRO_RESULTS)
    long_term_compatible = shared.metric_semantics_compatible(long_term_result)
    if long_term_result and not long_term_compatible:
        long_term_message = "long-term baseline loaded but metric semantics version is incompatible"
    long_term_micro_rows = long_term_result.get("microbenchmarks", []) if long_term_compatible else None

    result["baseline"] = {
        "compatible": baseline_compatible,
        "path": args.baseline_results,
        "commit": args.baseline_commit,
        "message": baseline_message,
        "metric_semantics_version": baseline_result.get("metric_semantics_version") if baseline_result else None,
    }
    result["long_term_baseline"] = {
        "compatible": long_term_compatible,
        "path": os.path.relpath(shared.LONG_TERM_MICRO_RESULTS, os.getcwd()),
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
    result["baseline_change_summaries"] = shared.summarize_micro_baseline_changes(
        result["microbenchmarks"], baseline_micro_rows, "PR base"
    )
    result["baseline_change_summaries"] += shared.summarize_micro_baseline_changes(
        result["microbenchmarks"], long_term_micro_rows, "Long-term baseline"
    )

    output_path = os.path.join(args.output_dir, "microbenchmarks.json")
    with open(output_path, "w", encoding="utf-8") as fp:
        json.dump(result, fp, indent=2)
        fp.write("\n")
    shared.plot_all_micro_metrics(args.output_dir, result["microbenchmarks"], baseline_micro_rows, long_term_micro_rows)

    print("=== JAOPS Microbenchmarks ===")
    print(f"metrics: {len(result['microbenchmarks'])}")
    print(f"baseline: {baseline_message}")
    print(f"long-term baseline: {long_term_message}")
    print(f"Artifacts written to: {os.path.abspath(args.output_dir)}")


if __name__ == "__main__":
    main()

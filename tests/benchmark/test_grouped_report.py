import copy
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts" / "benchmarks"))

import report
from common import yamcs as bench
from common.metrics import SIMULATOR_METRICS
from common.mdb_leaves import leaves_for_xtce
from common.plotting import raw_percentile_columns


def result(scale=1):
    rows = []
    for size in (1, 4):
        stats = {
            "count": 192, "min": 10, "p1": 11, "p5": 12, "p30": 13,
            "median": 20, "mean": 21, "p70": 24, "p95": 35, "p99": 38, "max": 40,
        }
        stats = {key: value if key == "count" else value * size * scale for key, value in stats.items()}
        rows.append({
            "group_size": size, "groups": 64,
            "group_tick_busy_ns": stats["median"],
            "group_tick_busy_ns_distribution": stats,
        })
    return {
        "benchmark_semver": bench.BENCHMARK_SEMVER,
        "revision": {"plugin_commit": "a" * 40}, "fixture_commit": "b" * 40,
        "parameters": ["/p/one", "/p/two", "/p/three", "/p/four"],
        "tick_seconds": 1, "overlap_target": 0.15, "scenarios": rows,
    }


class GroupWorkReportTests(unittest.TestCase):
    def test_complete_distribution_and_standard_comparison(self):
        head, base = result(), result(2)
        report.validate_grouped_result(head)
        rows, message = report.compatible_grouped(head, base)
        self.assertEqual(message, "loaded")
        summaries = bench.summarize_baseline_changes(
            head["scenarios"], rows, "PR base", x_key="group_size", keys=[report.GROUP_METRIC]
        )
        self.assertEqual(summaries[0]["median_change_pct"], -50)
        self.assertEqual(summaries[0]["samples"], 2)
        columns = raw_percentile_columns({row["group_size"]: row for row in rows}, [1, 4], report.GROUP_METRIC)
        self.assertEqual(set(columns), {"p1", "p5", "p30", "p70", "p95", "p99"})

    def test_incompatible_pool_fixture_or_tick_cannot_be_compared(self):
        for key, value in (
            ("parameters", ["/different"]), ("fixture_commit", "c" * 40),
            ("tick_seconds", 0.2), ("overlap_target", 0.2), ("benchmark_semver", "old"),
        ):
            with self.subTest(key=key):
                baseline = result()
                baseline[key] = value
                rows, message = report.compatible_grouped(result(), baseline)
                self.assertIsNone(rows)
                self.assertEqual(message, f"incompatible {key}")

    def test_different_group_counts_are_not_compared(self):
        baseline = result()
        baseline["scenarios"][0]["groups"] = 4
        rows, _ = report.compatible_grouped(result(), baseline)
        self.assertEqual([row["group_size"] for row in rows], [4])

    def test_invalid_results_fail_instead_of_plotting_defaults(self):
        changes = (
            lambda data: data.update(scenarios=[]),
            lambda data: data["scenarios"].append(copy.deepcopy(data["scenarios"][0])),
            lambda data: data["scenarios"][0].update(group_tick_busy_ns=999),
            lambda data: data["scenarios"][0]["group_tick_busy_ns_distribution"].update(count=0),
            lambda data: data["scenarios"][0]["group_tick_busy_ns_distribution"].update(p95=float("nan")),
            lambda data: data["scenarios"][0]["group_tick_busy_ns_distribution"].pop("p5"),
        )
        for change in changes:
            with self.subTest(change=change):
                data = result()
                change(data)
                with self.assertRaises(ValueError):
                    report.validate_grouped_result(data)

    def test_plot_uses_standard_head_base_and_long_term_series(self):
        with tempfile.TemporaryDirectory() as directory:
            with patch.object(bench, "plot_series", wraps=bench.plot_series) as series:
                path = bench.plot_metric(
                    directory, result()["scenarios"], report.GROUP_METRIC,
                    result(2)["scenarios"], result(3)["scenarios"],
                    x_key="group_size", x_label="Parameters per logical client group (K)",
                )
            self.assertEqual([call.args[3] for call in series.call_args_list], ["head", "pr_base", "long_term"])
            self.assertEqual(Path(path).name, "simulator_group_tick_busy_ns.png")
            self.assertGreater(Path(path).stat().st_size, 1000)

    def test_regular_report_contains_one_metric_graph_and_comparison_table(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for name, value in (("head", result()), ("base", result(2))):
                (root / f"{name}.json").write_text(json.dumps(value))
            output = root / "report"
            subprocess.run([
                sys.executable, str(ROOT / "scripts/benchmarks/report.py"),
                "--output-dir", str(output),
                "--grouped-results", str(root / "head.json"),
                "--baseline-grouped-results", str(root / "base.json"),
                "--long-term-grouped-results", "",
                "--long-term-micro-results", "", "--long-term-simulator-results", "",
                "--long-term-grafana-results", "",
            ], check=True, capture_output=True, text=True)
            comment = (output / "benchmark-comment.md").read_text()
            title = SIMULATOR_METRICS[report.GROUP_METRIC]["name"]
            self.assertIn(title, comment)
            self.assertIn("| Reference | Samples | Median change |", comment)
            self.assertIn("| PR base | 2 | -50.0%", comment)
            self.assertEqual(comment.count(f"![{title}]"), 1)
            self.assertNotIn("split_over_grouped", comment)
            status = json.loads((output / "benchmark-status.json").read_text())
            self.assertEqual(status["plots"], ["simulator_group_tick_busy_ns.png"])
            self.assertTrue(status["grouped_baseline"]["compatible"])
            self.assertTrue((output / "simulator/grouped.csv").exists())


class MdbLeavesTests(unittest.TestCase):
    def test_nested_aggregate_array_and_multidimensional_paths(self):
        xml = """<SpaceSystem name="test"><TelemetryMetaData><ParameterTypeSet>
        <FloatParameterType name="number"/>
        <ArrayParameterType name="array" arrayTypeRef="number"><DimensionList>
        <Dimension><StartingIndex><FixedValue>0</FixedValue></StartingIndex>
        <EndingIndex><FixedValue>1</FixedValue></EndingIndex></Dimension>
        </DimensionList></ArrayParameterType>
        <AggregateParameterType name="group"><MemberList>
        <Member name="values" typeRef="array"/></MemberList></AggregateParameterType>
        <ArrayParameterType name="matrix" arrayTypeRef="group"><DimensionList>
        <Dimension><StartingIndex><FixedValue>0</FixedValue></StartingIndex>
        <EndingIndex><FixedValue>1</FixedValue></EndingIndex></Dimension>
        <Dimension><StartingIndex><FixedValue>0</FixedValue></StartingIndex>
        <EndingIndex><FixedValue>1</FixedValue></EndingIndex></Dimension>
        </DimensionList></ArrayParameterType></ParameterTypeSet><ParameterSet>
        <Parameter name="data" parameterTypeRef="matrix"/>
        </ParameterSet></TelemetryMetaData></SpaceSystem>"""
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "test.xml"
            path.write_text(xml)
            expected = [f"/test/data[{a}][{b}].values[{c}]" for a in range(2) for b in range(2) for c in range(2)]
            self.assertEqual(leaves_for_xtce(path), expected)


if __name__ == "__main__":
    unittest.main()

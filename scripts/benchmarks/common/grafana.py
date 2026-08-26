#!/usr/bin/env python3
import json
import os
from statistics import median
from typing import Any

import matplotlib.pyplot as plt
from matplotlib.axes import Axes
from matplotlib.ticker import FuncFormatter, NullFormatter

from .metrics import GRAFANA_METRICS, GRAFANA_THRESHOLDS
from .plotting import (
    PLOT_COLORS,
    add_range_band,
    format_axis_tick,
    format_baseline_change_rows,
    format_change,
    format_value,
    percent_change,
    plot_series,
    range_values,
    split_change_extremes,
    status_emoji,
)

METRICS = GRAFANA_METRICS
THRESHOLDS = GRAFANA_THRESHOLDS


def load_result(path: str) -> dict[str, Any] | None:
    if not path or not os.path.exists(path):
        return None
    with open(path, encoding="utf-8") as fp:
        value = json.load(fp)
    return value if isinstance(value, dict) else None


def result_rows(result: dict[str, Any] | None) -> list[dict[str, Any]]:
    if not result:
        return []
    rows = list(result.get("results", []))
    return sorted(rows, key=lambda row: row.get("panels", 0))


def has_metric(result: dict[str, Any], metric: str) -> bool:
    return any(isinstance(row.get(metric), (int, float)) for row in result_rows(result))


def style_axis(ax: Axes, title: str, ylabel: str, unit: str) -> None:
    ax.set_title(title, loc="left", fontsize=13, fontweight="normal", pad=18)
    ax.set_xlabel("Number of Grafana panels")
    ax.set_ylabel(ylabel)
    ax.spines[["top", "right"]].set_visible(False)
    for spine in ax.spines.values():
        spine.set_color(PLOT_COLORS["grid"])
    ax.grid(True, which="major", color=PLOT_COLORS["grid"], alpha=0.78)
    ax.grid(True, which="minor", color=PLOT_COLORS["minor_grid"], alpha=0.9)
    ax.yaxis.set_major_formatter(FuncFormatter(lambda value, _position: format_axis_tick(value, unit)))
    ax.yaxis.set_minor_formatter(NullFormatter())
    ax.yaxis.offsetText.set_visible(False)


def threshold_plot_value(row: dict[str, Any], key: str, threshold_value: float) -> float | None:
    if key == "backend_datapoints_per_second_per_panel":
        panels = row.get("panels")
        return threshold_value * float(panels) if isinstance(panels, (int, float)) else None
    if key == "backend_run_stream_runtime_ns_per_sample":
        samples = row.get("backend_run_stream_samples")
        return threshold_value * float(samples) if isinstance(samples, (int, float)) else None
    return threshold_value


def thresholds_for_plot(rows: list[dict[str, Any]], metric: str) -> list[tuple[str, str, str, list[int], list[float]]]:
    lines = []
    for key, threshold in THRESHOLDS.items():
        if threshold.get("plot_metric", key) != metric:
            continue
        operator = str(threshold.get("operator", "max"))
        xs = []
        for row in rows:
            panels = row.get("panels")
            if isinstance(panels, int):
                xs.append(panels)
        if not xs:
            continue
        for level, color in [("warn", PLOT_COLORS["warn"]), ("fail", PLOT_COLORS["fail"])]:
            values = []
            line_xs = []
            for row in rows:
                panels = row.get("panels")
                if not isinstance(panels, int):
                    continue
                value = threshold_plot_value(row, key, float(threshold[level]))
                if value is None:
                    continue
                line_xs.append(panels)
                values.append(value)
            if values:
                lines.append((level, operator, color, line_xs, values))
    return lines


def constant_threshold_level(threshold_lines: list[tuple[str, str, str, list[int], list[float]]], level: str) -> float | None:
    for threshold_level, _operator, _color, _xs, values in threshold_lines:
        if threshold_level == level and values and all(value == values[0] for value in values):
            return float(values[0])
    return None


def threshold_operator(threshold_lines: list[tuple[str, str, str, list[int], list[float]]]) -> str:
    for _level, operator, _color, _xs, _values in threshold_lines:
        return operator
    return "max"


def add_threshold_regions(ax: Axes, threshold_lines: list[tuple[str, str, str, list[int], list[float]]]) -> None:
    warn = constant_threshold_level(threshold_lines, "warn")
    fail = constant_threshold_level(threshold_lines, "fail")
    if warn is None or fail is None:
        return
    ymin, ymax = ax.get_ylim()
    if threshold_operator(threshold_lines) == "min":
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


def plot_metric(
    output_dir: str,
    head: dict[str, Any],
    baseline: dict[str, Any] | None,
    long_term: dict[str, Any] | None,
    metric: str,
) -> str:
    config = METRICS[metric]
    if not has_metric(head, metric):
        return ""
    path = os.path.join(output_dir, config["file"])
    head_rows = [row for row in result_rows(head) if isinstance(row.get(metric), (int, float))]
    threshold_lines = thresholds_for_plot(head_rows, metric)
    fig, ax = plt.subplots(figsize=(9.6, 5.4))
    fig.patch.set_facecolor("#f8fafc")
    ax.set_facecolor("white")

    all_values: list[float] = []
    series = [
        (head, "head"),
        (baseline, "pr_base"),
        (long_term, "long_term"),
    ]
    for result, prefix in series:
        if not result:
            continue
        rows = [row for row in result_rows(result) if isinstance(row.get(metric), (int, float))]
        xs = [int(row["panels"]) for row in rows]
        ys = [float(row[metric]) for row in rows]
        if not xs:
            continue
        all_values.extend(ys)
        if prefix == "head":
            ys_min = []
            ys_max = []
            for row, y in zip(rows, ys):
                min_value, max_value = range_values(row, metric)
                ys_min.append(y if min_value is None else min_value)
                ys_max.append(y if max_value is None else max_value)
            all_values.extend(ys_min)
            all_values.extend(ys_max)
            add_range_band(ax, xs, ys_min, ys_max, PLOT_COLORS["head"])
        plot_series(ax, xs, ys, prefix, zorder=3 if prefix == "head" else 2)
    for level, _operator, color, xs, ys in threshold_lines:
        ax.plot(xs, ys, color=color, linewidth=1.25, alpha=0.95, zorder=2)
        all_values.extend(ys)

    ax.set_xscale("log")
    if config.get("log") and any(value > 0 for value in all_values):
        ax.set_yscale("log")
    style_axis(ax, config["title"], config["label"], config["unit"])
    add_threshold_regions(ax, threshold_lines)
    warn = constant_threshold_level(threshold_lines, "warn")
    fail = constant_threshold_level(threshold_lines, "fail")
    if warn is not None:
        add_threshold_label(ax, "warn", warn, PLOT_COLORS["warn_text"])
    if fail is not None:
        add_threshold_label(ax, "fail", fail, PLOT_COLORS["fail_text"])
    ax.legend(frameon=False, loc="upper left", ncols=3)
    fig.tight_layout()
    fig.savefig(path, dpi=180, facecolor="white")
    plt.close()
    return path


def summarize_changes(head: dict[str, Any], baseline: dict[str, Any] | None, label: str) -> dict[str, dict[str, Any]]:
    if not baseline:
        return {}
    base_by_key = {row.get("panels"): row for row in baseline.get("results", [])}
    summaries = {}
    for metric in METRICS:
        changes = []
        for row in head.get("results", []):
            base = base_by_key.get(row.get("panels"))
            if not base:
                continue
            if not isinstance(row.get(metric), (int, float)) or not isinstance(base.get(metric), (int, float)):
                continue
            change = percent_change(float(row[metric]), float(base[metric]))
            if change is not None:
                changes.append(change)
        if changes:
            max_negative_change, max_positive_change = split_change_extremes(changes)
            summaries[metric] = {
                "baseline": label,
                "samples": len(changes),
                "median_change_pct": median(changes),
                "max_negative_change_pct": max_negative_change,
                "max_positive_change_pct": max_positive_change,
            }
    return summaries


def threshold_metric_value(row: dict[str, Any], key: str) -> float | None:
    if key == "backend_datapoints_per_second_per_panel":
        panels = float(row.get("panels") or 0)
        if panels <= 0:
            return None
        value = row.get("backend_datapoints_per_second")
        return float(value) / panels if isinstance(value, (int, float)) else None
    if key == "backend_run_stream_runtime_ns_per_sample":
        samples = float(row.get("backend_run_stream_samples") or 0)
        if samples <= 0:
            return None
        value = row.get("backend_run_stream_runtime_ns")
        return float(value) / samples if isinstance(value, (int, float)) else None
    value = row.get(key)
    return float(value) if isinstance(value, (int, float)) else None


def evaluate_thresholds(head: dict[str, Any]) -> dict[str, dict[str, Any]]:
    evaluated = {}
    rows = head.get("results", [])
    for key, threshold in THRESHOLDS.items():
        values = [value for row in rows if (value := threshold_metric_value(row, key)) is not None]
        if not values:
            continue
        operator = threshold["operator"]
        observed = max(values) if operator == "max" else min(values)
        warn = float(threshold["warn"])
        fail = float(threshold["fail"])
        if operator == "max":
            status = "fail" if observed > fail else "warn" if observed > warn else "pass"
        else:
            status = "fail" if observed < fail else "warn" if observed < warn else "pass"
        evaluated[key] = {
            "metric": key,
            "plot_metric": threshold.get("plot_metric", key),
            "operator": operator,
            "observed": observed,
            "warn": warn,
            "fail": fail,
            "unit": threshold["unit"],
            "status": status,
        }
    return evaluated


def build_comment(
    head: dict[str, Any],
    baseline: dict[str, Any] | None,
    long_term: dict[str, Any] | None,
    plots_base_url: str,
) -> str:
    pr_base_changes = summarize_changes(head, baseline, "PR base")
    long_term_changes = summarize_changes(head, long_term, "Long-term baseline")
    thresholds = evaluate_thresholds(head)
    thresholds_by_plot_metric: dict[str, list[dict[str, Any]]] = {}
    for threshold in thresholds.values():
        thresholds_by_plot_metric.setdefault(str(threshold["plot_metric"]), []).append(threshold)
    lines = []
    for metric, config in METRICS.items():
        if not has_metric(head, metric):
            continue
        change = format_change(pr_base_changes.get(metric, {}).get("median_change_pct"))
        suffix = f" - median vs base {change}" if change else ""
        change_summaries = [
            summary
            for summary in [pr_base_changes.get(metric), long_term_changes.get(metric)]
            if isinstance(summary, dict)
        ]
        metric_thresholds = thresholds_by_plot_metric.get(metric, [])
        metric_status = "pass"
        if any(threshold["status"] == "fail" for threshold in metric_thresholds):
            metric_status = "fail"
        elif any(threshold["status"] == "warn" for threshold in metric_thresholds):
            metric_status = "warn"
        image = config["file"]
        image_url = f"{plots_base_url.rstrip('/')}/{image}" if plots_base_url else f"benchmark-output/plots/{image}"
        lines.extend(
            [
                "<details>",
                f"<summary>{status_emoji(metric_status)} {config['title']}{suffix}</summary>",
                "",
            ]
        )
        if metric_thresholds:
            lines.extend(
                [
                    "| Status | Observed | Warning threshold | Failure threshold |",
                    "|---|---:|---:|---:|",
                ]
            )
            for threshold in metric_thresholds:
                lines.append(
                    "| {status} | {observed} | {warn} | {fail} |".format(
                        status=f"{status_emoji(str(threshold['status']))} `{str(threshold['status']).upper()}`",
                        observed=format_value(float(threshold["observed"]), str(threshold["unit"])),
                        warn=format_value(float(threshold["warn"]), str(threshold["unit"])),
                        fail=format_value(float(threshold["fail"]), str(threshold["unit"])),
                    )
                )
            lines.append("")
        if change_summaries:
            lines.extend(format_baseline_change_rows(change_summaries))
        lines.extend([f"![{config['title']}]({image_url})", "", "</details>", ""])
    return "\n".join(lines).rstrip() + "\n"

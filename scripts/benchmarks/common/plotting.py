from typing import Any

from matplotlib.axes import Axes

PLOT_COLORS = {
    "head": "#1d4ed8",
    "base": "#64748b",
    "pr_base": "#64748b",
    "long_term": "#16a34a",
    "warn": "#f59e0b",
    "warn_text": "#b45309",
    "fail": "#ef4444",
    "fail_text": "#b91c1c",
    "grid": "#e2e8f0",
    "minor_grid": "#f1f5f9",
}

SERIES_STYLES = {
    "head": {"color": PLOT_COLORS["head"], "linestyle": "-", "label": "HEAD", "linewidth": 3.0},
    "pr_base": {"color": PLOT_COLORS["pr_base"], "linestyle": "-", "label": "PR base", "linewidth": 2.3},
    "base": {"color": PLOT_COLORS["base"], "linestyle": "-", "label": "PR base", "linewidth": 2.3},
    "long_term": {
        "color": PLOT_COLORS["long_term"],
        "linestyle": (0, (5, 4)),
        "label": "Long-term baseline",
        "linewidth": 2.1,
    },
}


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


def format_time(value: float, unit: str = "ns") -> str:
    multipliers = {"ns": 1, "us": 1_000, "ms": 1_000_000, "s": 1_000_000_000}
    ns = value * multipliers.get(unit, 1)
    abs_ns = abs(ns)
    if abs_ns >= 1_000_000_000:
        return f"{format_number(ns / 1_000_000_000)} s"
    if abs_ns >= 1_000_000:
        return f"{format_number(ns / 1_000_000)} ms"
    if abs_ns >= 1_000:
        return f"{format_number(ns / 1_000)} us"
    return f"{format_number(ns)} ns"


def format_bytes(value: float, unit: str = "bytes") -> str:
    multipliers = {"bytes": 1, "KiB": 1024, "MiB": 1024 * 1024, "GiB": 1024 * 1024 * 1024}
    bytes_value = value * multipliers.get(unit, 1)
    abs_bytes = abs(bytes_value)
    if abs_bytes >= 1024 * 1024 * 1024:
        return f"{format_number(bytes_value / (1024 * 1024 * 1024))} GiB"
    if abs_bytes >= 1024 * 1024:
        return f"{format_number(bytes_value / (1024 * 1024))} MiB"
    if abs_bytes >= 1024:
        return f"{format_number(bytes_value / 1024)} KiB"
    return f"{format_number(bytes_value)} bytes"


def format_axis_tick(value: float, axis_unit: str) -> str:
    if axis_unit in {"ns", "us", "ms", "s"}:
        return format_time(value, axis_unit)
    if axis_unit in {"bytes", "KiB", "MiB", "GiB"}:
        return format_bytes(value, axis_unit)
    if axis_unit == "%":
        return f"{format_number(value)}%"
    if axis_unit == "values/sec":
        if abs(value) >= 1_000_000:
            return f"{format_number(value / 1_000_000)}M values/s"
        if abs(value) >= 1_000:
            return f"{format_number(value / 1_000)}k values/s"
        return f"{format_number(value)} values/s"
    if axis_unit in {"count", "count/s"}:
        return format_number(value)
    return f"{format_number(value)} {axis_unit}".rstrip()


def format_value(value: float, unit: str) -> str:
    if unit in {"ns", "ns/stream", "ns/sample"}:
        suffix = ""
        if unit.endswith("/stream"):
            suffix = "/stream"
        elif unit.endswith("/sample"):
            suffix = "/sample"
        return f"{format_time(value, 'ns')}{suffix}"
    if unit == "ms":
        return format_time(value, "ms")
    if unit in {"bytes", "bytes/stream", "bytes/panel"}:
        suffix = ""
        if unit.endswith("/stream"):
            suffix = "/stream"
        elif unit.endswith("/panel"):
            suffix = "/panel"
        return f"{format_bytes(value)}{suffix}"
    if unit == "%":
        return f"{format_number(value)}%"
    return f"{format_number(value)} {unit}".rstrip()


def format_change(value: float | None) -> str:
    if value is None:
        return ""
    return f"{value:+.1f}%"


def percent_change(current: float, baseline: float) -> float | None:
    if baseline == 0:
        return None
    return 100 * (current - baseline) / abs(baseline)


def format_baseline_change_rows(summaries: list[dict[str, Any]]) -> list[str]:
    if not summaries:
        return []
    lines = [
        "| Reference | Samples | Median change | Max negative change | Max positive change |",
        "|---|---:|---:|---:|---:|",
    ]
    for summary in summaries:
        lines.append(
            "| {baseline} | {samples} | {median} | {negative} | {positive} |".format(
                baseline=summary.get("baseline", "baseline"),
                samples=int(summary.get("samples", 0)),
                median=format_change(float(summary.get("median_change_pct", 0))),
                negative=format_change(float(summary.get("max_negative_change_pct", summary.get("min_change_pct", 0)))),
                positive=format_change(float(summary.get("max_positive_change_pct", summary.get("max_change_pct", 0)))),
            )
        )
    lines.append("")
    return lines


def status_for(thresholds: list[dict[str, Any]]) -> str:
    if any(t["status"] == "fail" for t in thresholds):
        return "fail"
    if any(t["status"] == "warn" for t in thresholds):
        return "warn"
    return "pass"


def status_label(status: str) -> str:
    return {"pass": "PASS", "warn": "WARN", "fail": "FAIL"}[status]


def status_emoji(status: str) -> str:
    return {"pass": "✅", "warn": "⚠️", "fail": "❌"}.get(status, "ℹ️")


def split_change_extremes(changes: list[float]) -> tuple[float, float]:
    negative_changes = [change for change in changes if change < 0]
    positive_changes = [change for change in changes if change > 0]
    return (
        min(negative_changes) if negative_changes else 0.0,
        max(positive_changes) if positive_changes else 0.0,
    )


def plot_series(ax: Axes, xs: list[int], ys: list[float], series: str, zorder: int = 2) -> None:
    style = SERIES_STYLES[series]
    ax.plot(
        xs,
        ys,
        color=str(style["color"]),
        linewidth=float(style["linewidth"]),
        linestyle=style["linestyle"],
        label=str(style["label"]),
        zorder=zorder,
    )
    ax.scatter(xs, ys, s=28 if series != "head" else 34, color=str(style["color"]), edgecolor="white", linewidth=1.0, zorder=zorder + 1)


def range_values(row: dict[str, Any], metric: str, value_key: str = "") -> tuple[float | None, float | None]:
    candidates = [
        (f"{metric}_min", f"{metric}_max"),
        (f"min_{metric}", f"max_{metric}"),
    ]
    if value_key:
        candidates.append((f"min_{value_key}", f"max_{value_key}"))
    for min_key, max_key in candidates:
        min_value = row.get(min_key)
        max_value = row.get(max_key)
        if isinstance(min_value, (int, float)) and isinstance(max_value, (int, float)):
            return float(min_value), float(max_value)
    return None, None


def add_range_band(ax: Axes, xs: list[int], ys_min: list[float], ys_max: list[float], color: str) -> None:
    if not xs or not ys_min or not ys_max:
        return
    if not any(min_value != max_value for min_value, max_value in zip(ys_min, ys_max)):
        return
    ax.fill_between(xs, ys_min, ys_max, color=color, alpha=0.11, linewidth=0, zorder=1)


# Ordered from closest-to-median (most points, least faded) to the extremes
# (fewest points, most faded), so the shading reads as a point-density fan
# rather than a flat min/max box.
DENSITY_BAND_STEPS: tuple[tuple[str, str, float], ...] = (
    ("median", "p70", 0.30),
    ("p70", "p95", 0.18),
    ("p95", "p99", 0.10),
    ("p99", "max", 0.05),
    ("median", "p30", 0.30),
    ("p30", "p5", 0.18),
    ("p5", "p1", 0.10),
    ("p1", "min", 0.05),
)


def add_density_band(
    ax: Axes,
    xs: list[int],
    ys: list[float],
    ys_min: list[float],
    ys_max: list[float],
    percentile_columns: dict[str, list[float]] | None,
    color: str,
) -> None:
    """Shade the spread around the head series.

    When per-point percentiles (p1/p5/p30/median/p70/p95/p99) are supplied,
    draws a fan of progressively more faded bands out from the median in
    both directions, so denser regions of the distribution look darker.
    Otherwise falls back to a single flat min/max band - today's benchmark
    harnesses only record min/max per point, so real reports use the
    fallback until percentile capture is added upstream.
    """
    if not xs:
        return

    columns: dict[str, list[float]] = dict(percentile_columns or {})
    columns.setdefault("median", ys)
    columns.setdefault("min", ys_min)
    columns.setdefault("max", ys_max)

    drew_any = False
    for inner_key, outer_key, alpha in DENSITY_BAND_STEPS:
        inner = columns.get(inner_key)
        outer = columns.get(outer_key)
        if not inner or not outer or len(inner) != len(xs) or len(outer) != len(xs):
            continue
        if all(a == b for a, b in zip(inner, outer)):
            continue
        ax.fill_between(xs, inner, outer, color=color, alpha=alpha, linewidth=0, zorder=1)
        drew_any = True

    if not drew_any:
        add_range_band(ax, xs, ys_min, ys_max, color)


def apply_y_axis_floor(ax: Axes, values: list[float]) -> None:
    if not values:
        return
    ymin = min(values)
    ymax = max(values)
    if ymin >= 0:
        upper = ymax * 1.12 if ymax > 0 else 1
        ax.set_ylim(bottom=0, top=upper)
        return

    span = ymax - ymin
    min_span = max(abs(ymax), abs(ymin), 1) * 0.2
    if span < min_span:
        midpoint = (ymax + ymin) / 2
        half = min_span / 2
        ax.set_ylim(midpoint - half, midpoint + half)
    else:
        ax.set_ylim(ymin - span * 0.06, ymax + span * 0.12)


def apply_log_y_axis(ax: Axes, values: list[float]) -> None:
    positive = [value for value in values if value > 0]
    if not positive:
        return
    ax.set_yscale("log")
    ax.set_ylim(bottom=min(positive) * 0.8, top=max(positive) * 1.2)


def apply_percentage_y_axis(ax: Axes, values: list[float]) -> None:
    if not values:
        return
    ymin = max(0, min(values) - 5)
    ymax = min(105, max(values) + 2)
    if ymax - ymin < 10:
        midpoint = (ymin + ymax) / 2
        ymin = max(0, midpoint - 5)
        ymax = min(105, midpoint + 5)
    ax.set_ylim(bottom=ymin, top=ymax)


def add_threshold_bands(ax: Axes, threshold_lines: list[tuple[str, str, str, list[int], list[float]]]) -> None:
    """Shade the warn/fail zones using the actual (possibly per-x-scaled)
    threshold line values, not just a flat horizontal span, so scaled
    thresholds (e.g. per-panel/per-sample) shade correctly too.
    """
    warn_entry = next(((xs, ys) for level, _op, _color, xs, ys in threshold_lines if level == "warn"), None)
    fail_entry = next(((xs, ys) for level, _op, _color, xs, ys in threshold_lines if level == "fail"), None)
    if warn_entry is None or fail_entry is None:
        return
    xs, warn_ys = warn_entry
    _, fail_ys = fail_entry
    if len(warn_ys) != len(xs) or len(fail_ys) != len(xs):
        return

    operator = next((operator for _level, operator, _color, _xs, _ys in threshold_lines), "max")
    ymin, ymax = ax.get_ylim()
    top = [ymax] * len(xs)
    bottom = [ymin] * len(xs)
    if operator == "min":
        ax.fill_between(xs, fail_ys, warn_ys, color=PLOT_COLORS["warn"], alpha=0.07, linewidth=0, zorder=0)
        ax.fill_between(xs, bottom, fail_ys, color=PLOT_COLORS["fail"], alpha=0.055, linewidth=0, zorder=0)
        return
    ax.fill_between(xs, warn_ys, fail_ys, color=PLOT_COLORS["warn"], alpha=0.07, linewidth=0, zorder=0)
    ax.fill_between(xs, fail_ys, top, color=PLOT_COLORS["fail"], alpha=0.055, linewidth=0, zorder=0)


def add_threshold_line_label(ax: Axes, xs: list[int], ys: list[float], label: str, color: str) -> None:
    """Label a threshold line (constant or per-x-scaled) at its right-most
    point, clipped to the current (data-driven) view so labels for
    thresholds that fall outside the visible range are simply skipped
    instead of stretching the axis to fit them.
    """
    if not xs or not ys:
        return
    ymin, ymax = ax.get_ylim()
    xmax = ax.get_xlim()[1]
    value = ys[-1]
    if not ymin <= value <= ymax:
        return
    text_y = value * 1.04 if ax.get_yscale() == "log" else value + (ymax - ymin) * 0.018
    text_y = min(max(text_y, ymin), ymax)
    ax.text(xmax, text_y, label, ha="right", va="bottom", color=color, fontsize=9, fontweight="bold", zorder=4)

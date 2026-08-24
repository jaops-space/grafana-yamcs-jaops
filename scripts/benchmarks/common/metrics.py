SIMULATOR_METRICS = {
    "live_memory_growth_bytes": {
        "name": "Simulator - live memory used during run",
        "label": "Live memory used during simulator run",
        "title": "Simulator - Live memory used while N Grafana streams run",
        "file": "simulator_live_memory_growth_bytes.png",
    },
    "total_allocated_bytes": {
        "name": "Simulator - total memory allocated during run",
        "label": "Total memory allocated during simulator run",
        "title": "Simulator - Total memory allocated while N Grafana streams run",
        "file": "simulator_total_allocated_bytes.png",
    },
    "values_read_per_sec": {
        "name": "Simulator - values read per second from buffers",
        "label": "Values read per second from simulator buffers",
        "title": "Simulator - Values read per second by N Grafana streams",
        "file": "simulator_values_read_per_sec.png",
    },
    "values_read_fresh_pct": {
        "name": "Simulator - values read within the same 1s tick",
        "label": "Values read within 1s simulator tick",
        "title": "Simulator - Values read within the same 1s simulator tick",
        "file": "simulator_values_read_fresh_pct.png",
    },
    "median_tick_runstream_busy": {
        "name": "Simulator - Median RunStream busy time per 1s tick",
        "label": "Median RunStream busy time per 1s simulator tick",
        "title": "Simulator - Median time spent doing RunStream work per 1s tick",
        "file": "simulator_median_tick_runstream_busy.png",
    },
    "setup": {
        "name": "Simulator - stream setup time",
        "label": "Time to set up simulator streams",
        "title": "Simulator - Time to set up N Grafana streams",
        "file": "simulator_setup.png",
    },
}

MICRO_METRICS = {
    "frame_numeric_full": {
        "name": "Microbenchmark - median time to convert numeric buffer to full frame",
        "label": "Median time to convert numeric buffer to full frame",
        "title": "Microbenchmark - Median time to convert a numeric buffer to a full frame",
        "file": "micro_frame_numeric_full.png",
    },
    "frame_numeric_average": {
        "name": "Microbenchmark - median time to convert numeric buffer to average frame",
        "label": "Median time to convert numeric buffer to average frame",
        "title": "Microbenchmark - Median time to convert a numeric buffer to an average frame",
        "file": "micro_frame_numeric_average.png",
    },
    "frame_numeric_average_minmax": {
        "name": "Microbenchmark - median time to convert numeric buffer to average/min/max frame",
        "label": "Median time to convert numeric buffer to average/min/max frame",
        "title": "Microbenchmark - Median time to convert a numeric buffer to an average/min/max frame",
        "file": "micro_frame_numeric_average_minmax.png",
    },
    "frame_discrete": {
        "name": "Microbenchmark - median time to convert discrete buffer to frame",
        "label": "Median time to convert discrete buffer to frame",
        "title": "Microbenchmark - Median time to convert a discrete buffer to a frame",
        "file": "micro_frame_discrete.png",
    },
    "process_stream_10_values": {
        "name": "Microbenchmark - median time to process 10 values into stream buffers",
        "label": "Median time to process 10 values into stream buffers",
        "title": "Microbenchmark - Median time to process 10 values into N stream buffers",
        "file": "micro_process_stream_10_values.png",
    },
}

GRAFANA_METRICS = {
    "time_to_panels_ready_ms": {
        "name": "Grafana - Time to panels ready",
        "title": "Grafana - Time to panels ready",
        "label": "Time to panels ready",
        "unit": "ms",
        "file": "grafana_ready.png",
        "log": True,
    },
    "browser_heap_after_gc_bytes": {
        "name": "Grafana - Browser heap after GC",
        "title": "Grafana - Browser heap after GC",
        "label": "Browser heap after GC",
        "unit": "bytes",
        "file": "grafana_heap.png",
        "log": False,
    },
    "backend_run_stream_runtime_ns": {
        "name": "Grafana - Total backend RunStream runtime",
        "title": "Grafana - Total backend RunStream runtime",
        "label": "Total backend RunStream runtime",
        "unit": "ns",
        "file": "grafana_runstream.png",
        "log": True,
    },
    "backend_median_heap_alloc_bytes": {
        "name": "Grafana - Median backend live heap while panels stream",
        "title": "Grafana - Median backend live heap while panels stream",
        "label": "Median backend live heap while panels stream",
        "unit": "bytes",
        "file": "grafana_backend_memory.png",
        "log": False,
    },
    "backend_datapoints_per_second": {
        "name": "Grafana - Datapoints received per second",
        "title": "Grafana - Datapoints received per second",
        "label": "Datapoints received per second",
        "unit": "count/s",
        "file": "grafana_datapoints.png",
        "log": False,
    },
    "live_streams_opened": {
        "name": "Grafana - Live streams opened",
        "title": "Grafana - Live streams opened",
        "label": "Live streams opened",
        "unit": "count",
        "file": "grafana_streams.png",
        "log": False,
    },
}

GRAFANA_THRESHOLDS = {
    "time_to_panels_ready_ms": {
        "operator": "max",
        "warn": 10_000,
        "fail": 20_000,
        "unit": "ms",
    },
    "backend_datapoints_per_second_per_panel": {
        "plot_metric": "backend_datapoints_per_second",
        "operator": "min",
        "warn": 0.8,
        "fail": 0.5,
        "unit": "points/sec/panel",
    },
    "backend_run_stream_runtime_ns_per_sample": {
        "plot_metric": "backend_run_stream_runtime_ns",
        "operator": "max",
        "warn": 500_000,
        "fail": 1_000_000,
        "unit": "ns/sample",
    },
    "backend_median_heap_alloc_bytes": {
        "plot_metric": "backend_median_heap_alloc_bytes",
        "operator": "max",
        "warn": 256 * 1024 * 1024,
        "fail": 512 * 1024 * 1024,
        "unit": "bytes",
    },
}

SIMULATOR_THRESHOLDS = {
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
    "median_tick_runstream_busy": {
        "warn": 1_000_000_000,
        "fail": 1_200_000_000,
        "operator": "max",
        "unit": "ns",
        "plot_key": "median_tick_runstream_busy",
        "scale": "constant",
    },
}

DERIVED_METRICS = {
    "setup_per_stream": {
        "name": "Setup time per stream",
        "unit": "ns/stream",
        "plot_metric": "setup",
    },
    "live_memory_growth_bytes_per_stream": {
        "name": "Live memory per stream",
        "unit": "bytes/stream",
        "plot_metric": "live_memory_growth_bytes",
    },
    "values_read_per_sec_per_stream": {
        "name": "Values read per second from buffers per stream",
        "unit": "values/sec/stream",
        "plot_metric": "values_read_per_sec",
    },
    "backend_datapoints_per_second_per_panel": {
        "name": "Grafana - Datapoints received per second per panel",
        "unit": "points/sec/panel",
        "plot_metric": "backend_datapoints_per_second",
    },
    "backend_run_stream_runtime_ns_per_sample": {
        "name": "Grafana - Backend RunStream runtime per sample",
        "unit": "ns/sample",
        "plot_metric": "backend_run_stream_runtime_ns",
    },
}

METRIC_UNITS = {
    "live_memory_growth_bytes": "bytes",
    "total_allocated_bytes": "bytes",
    "values_read_per_sec": "values/sec",
    "values_read_fresh_pct": "%",
    "avg_tick_runstream": "ns",
    "median_tick_runstream_busy": "ns",
    "setup": "ns",
    "frame_numeric_full": "ns",
    "frame_numeric_average": "ns",
    "frame_numeric_average_minmax": "ns",
    "frame_discrete": "ns",
    "process_stream_10_values": "ns",
    "time_to_panels_ready_ms": "ms",
    "browser_heap_after_gc_bytes": "bytes",
    "backend_run_stream_runtime_ns": "ns",
    "backend_median_heap_alloc_bytes": "bytes",
    "backend_datapoints_per_second": "count/s",
    "live_streams_opened": "count",
}

METRIC_NAMES = {key: value["name"] for key, value in DERIVED_METRICS.items()}
METRIC_NAMES.update({key: value["name"] for key, value in SIMULATOR_METRICS.items()})
METRIC_NAMES.update({key: value["name"] for key, value in MICRO_METRICS.items()})
METRIC_NAMES.update({key: value["name"] for key, value in GRAFANA_METRICS.items()})

PLOT_TO_METRIC = {
    value["file"]: key
    for source in (MICRO_METRICS, SIMULATOR_METRICS, GRAFANA_METRICS)
    for key, value in source.items()
}

PLOT_ORDER = [value["file"] for value in MICRO_METRICS.values()]
PLOT_ORDER += [value["file"] for value in SIMULATOR_METRICS.values()]
PLOT_ORDER += [value["file"] for value in GRAFANA_METRICS.values()]

THRESHOLD_TO_PLOT = {
    "setup": SIMULATOR_METRICS["setup"]["file"],
    "setup_per_stream": SIMULATOR_METRICS["setup"]["file"],
    "live_memory_growth_bytes": SIMULATOR_METRICS["live_memory_growth_bytes"]["file"],
    "live_memory_growth_bytes_per_stream": SIMULATOR_METRICS["live_memory_growth_bytes"]["file"],
    "total_allocated_bytes": SIMULATOR_METRICS["total_allocated_bytes"]["file"],
    "values_read_per_sec": SIMULATOR_METRICS["values_read_per_sec"]["file"],
    "values_read_per_sec_per_stream": SIMULATOR_METRICS["values_read_per_sec"]["file"],
    "values_read_fresh_pct": SIMULATOR_METRICS["values_read_fresh_pct"]["file"],
    "median_tick_runstream_busy": SIMULATOR_METRICS["median_tick_runstream_busy"]["file"],
    "time_to_panels_ready_ms": GRAFANA_METRICS["time_to_panels_ready_ms"]["file"],
    "backend_datapoints_per_second_per_panel": GRAFANA_METRICS["backend_datapoints_per_second"]["file"],
    "backend_run_stream_runtime_ns_per_sample": GRAFANA_METRICS["backend_run_stream_runtime_ns"]["file"],
    "backend_median_heap_alloc_bytes": GRAFANA_METRICS["backend_median_heap_alloc_bytes"]["file"],
}

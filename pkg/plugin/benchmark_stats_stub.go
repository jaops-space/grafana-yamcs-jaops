//go:build !benchmark

package plugin

import (
	"net/http"
	"time"
)

type benchmarkStatsCollector struct{}

var streamBenchmarkStats = benchmarkStatsCollector{}

func (benchmarkStatsCollector) recordRunStream(string) {}

func (benchmarkStatsCollector) recordRunStreamWork(string, time.Duration, int) {}

func (d *Datasource) handleBenchmarkReset(w http.ResponseWriter, req *http.Request) {
	http.NotFound(w, req)
}

func (d *Datasource) handleBenchmarkStats(w http.ResponseWriter, req *http.Request) {
	http.NotFound(w, req)
}

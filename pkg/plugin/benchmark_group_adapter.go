//go:build benchmark

package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/source"
)

// BenchmarkRunStream drives the real RunParameterStream for one stream path.
// packetSender receives every SendFrame packet through the SDK's StreamSender.
// Used only by the grouped-benchmark harness (benchmark build tag).
func BenchmarkRunStream(ctx context.Context, endpoint *source.YamcsEndpoint, q PluginQuery, path string, packetSender backend.StreamPacketSender) error {
	req := &backend.RunStreamRequest{Path: path}
	sender := backend.NewStreamSender(packetSender)
	return RunParameterStream(ctx, req, sender, endpoint, q)
}

// BenchmarkResetRunStreamWork clears the RunStream work spans recorded by
// recordRunStreamWork. The harness calls it between warmup and the measured window.
func BenchmarkResetRunStreamWork() {
	streamBenchmarkStats.reset(0)
	streamBenchmarkStats.mu.Lock()
	streamBenchmarkStats.collectPathSamples = true
	streamBenchmarkStats.mu.Unlock()
}

// BenchmarkRunStreamSamplesByPath returns the RunStream work spans (nanoseconds,
// one per non-empty tick that reached SendFrame) keyed by stream path.
func BenchmarkRunStreamSamplesByPath() map[string][]int64 {
	return streamBenchmarkStats.samplesByPath()
}

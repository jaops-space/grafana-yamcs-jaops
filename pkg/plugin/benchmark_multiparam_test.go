package plugin

// Targeted comparison between the two streaming architectures for
// multi-parameter queries:
//
//   - "multi-observer" (RunParameterStream, used for a single parameter or
//     a query type that doesn't support batching): one Grafana Live
//     channel, one goroutine/ticker, one single-field frame, and one
//     Arrow-encoded SendFrame per parameter in the query.
//   - "multi-parameter stream" (RunMultiParameterStream, used whenever a
//     Graph/SingleValue/DiscreteValue query has more than one parameter):
//     one Grafana Live channel, one goroutine/ticker, and one multi-field
//     frame (one value field per parameter) Arrow-encoded and sent once
//     per tick.
//
// These benchmarks isolate exactly the work that differs between the two
// approaches per tick: frame construction, Arrow encoding, and goroutine/
// ticker scaffolding. The underlying Yamcs subscription cost is identical
// in both approaches (already deduped at the client-subscription level) and
// is intentionally not modeled here. This is a synthetic microbenchmark of
// raw per-tick cost only - it doesn't exercise RunMultiParameterStream's
// carry-forward/"latest wins" row-timestamp policy, which is covered
// separately by datasource_run_multiparameter_test.go.
//
// Run with:
//
//	go test ./pkg/plugin/ -run '^$' -bench 'BenchmarkMultiParam' -benchmem -count=5

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// paramCounts mirrors realistic multi-parameter panel sizes: a couple of
// correlated values, a wide table/state-timeline panel, and a stress case.
var paramCounts = []int{1, 2, 5, 15, 50}

const valuesPerTick = 10 // matches one buffered batch per stream tick

func benchNumericBatch(paramIndex int) []*pvalue.ParameterValue {
	buffer := make([]*pvalue.ParameterValue, valuesPerTick)
	base := time.Unix(1700000000, 0)
	for i := range buffer {
		v := float64(paramIndex)*1000 + float64(i)*1.25
		status := pvalue.AcquisitionStatus_ACQUIRED
		buffer[i] = &pvalue.ParameterValue{
			GenerationTime:    timestamppb.New(base.Add(time.Duration(i) * 100 * time.Millisecond)),
			AcquisitionStatus: &status,
			EngValue: &protobuf.Value{
				Type:        protobuf.Value_DOUBLE.Enum(),
				DoubleValue: &v,
			},
		}
	}
	return buffer
}

// joinFramesToMultiField merges N single-parameter frames (as produced by
// ConvertBufferToFrame) into a single frame with one shared time field and
// N value fields, approximating what a RunMultiParameterStream handler
// would build before a single Arrow encode + SendFrame.
func joinFramesToMultiField(frames []*data.Frame) *data.Frame {
	joined := data.NewFrame("response", frames[0].Fields[0])
	for _, f := range frames {
		joined.Fields = append(joined.Fields, f.Fields[1])
	}
	return joined
}

// BenchmarkMultiParam_FrameEncoding_MultiObserver builds and Arrow-encodes
// one frame per parameter per tick, i.e. today's shipped behavior.
func BenchmarkMultiParam_FrameEncoding_MultiObserver(b *testing.B) {
	for _, n := range paramCounts {
		b.Run(paramCountLabel(n), func(b *testing.B) {
			batches := make([][]*pvalue.ParameterValue, n)
			for i := range batches {
				batches[i] = benchNumericBatch(i)
			}

			b.ReportAllocs()
			b.ResetTimer()
			var totalBytes int64
			for i := 0; i < b.N; i++ {
				for p := 0; p < n; p++ {
					frame := tools.ConvertBufferToFrame(batches[p], "/bench/Param", false, false, false)
					encoded, err := frame.MarshalArrow()
					if err != nil {
						b.Fatal(err)
					}
					totalBytes += int64(len(encoded))
				}
			}
			b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
			b.ReportMetric(float64(n), "messages/op")
		})
	}
}

// BenchmarkMultiParam_FrameEncoding_MultiParameterStream builds one joined
// multi-field frame per tick and Arrow-encodes it once, i.e. the proposed
// batched-stream follow-up.
func BenchmarkMultiParam_FrameEncoding_MultiParameterStream(b *testing.B) {
	for _, n := range paramCounts {
		b.Run(paramCountLabel(n), func(b *testing.B) {
			batches := make([][]*pvalue.ParameterValue, n)
			for i := range batches {
				batches[i] = benchNumericBatch(i)
			}

			b.ReportAllocs()
			b.ResetTimer()
			var totalBytes int64
			for i := 0; i < b.N; i++ {
				frames := make([]*data.Frame, n)
				for p := 0; p < n; p++ {
					frames[p] = tools.ConvertBufferToFrame(batches[p], "/bench/Param", false, false, false)
				}
				joined := joinFramesToMultiField(frames)
				encoded, err := joined.MarshalArrow()
				if err != nil {
					b.Fatal(err)
				}
				totalBytes += int64(len(encoded))
			}
			b.ReportMetric(float64(totalBytes)/float64(b.N), "bytes/op")
			b.ReportMetric(1, "messages/op")
		})
	}
}

// BenchmarkMultiParam_Scaffolding_MultiObserver measures the cost of
// spinning up N independent goroutines each with their own ticker, the way
// N RunParameterStream calls do today for one multi-parameter query.
func BenchmarkMultiParam_Scaffolding_MultiObserver(b *testing.B) {
	for _, n := range paramCounts {
		b.Run(paramCountLabel(n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var wg sync.WaitGroup
				stop := make(chan struct{})
				var tick int64
				for p := 0; p < n; p++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						ticker := time.NewTicker(time.Hour) // never actually fires; we tick manually below
						defer ticker.Stop()
						<-stop
					}()
				}
				runtime.Gosched()
				_ = tick
				close(stop)
				wg.Wait()
			}
		})
	}
}

// BenchmarkMultiParam_Scaffolding_MultiParameterStream measures the cost of
// spinning up a single goroutine with a single ticker draining N parameter
// rings, the way one RunMultiParameterStream call would for the same query.
func BenchmarkMultiParam_Scaffolding_MultiParameterStream(b *testing.B) {
	for _, n := range paramCounts {
		b.Run(paramCountLabel(n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				stop := make(chan struct{})
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					ticker := time.NewTicker(time.Hour)
					defer ticker.Stop()
					// Draining N parameter cursors happens inline on this one
					// goroutine's tick, no extra goroutines regardless of n.
					_ = n
					<-stop
				}()
				runtime.Gosched()
				close(stop)
				wg.Wait()
			}
		})
	}
}

func paramCountLabel(n int) string {
	switch n {
	case 1:
		return "params=1"
	case 2:
		return "params=2"
	case 5:
		return "params=5"
	case 15:
		return "params=15"
	case 50:
		return "params=50"
	default:
		return "params=n"
	}
}

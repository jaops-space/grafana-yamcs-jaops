//go:build benchmark

package source

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf"
	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/protobuf/pvalue"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/config"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/tools"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/types"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const microProcessValuesPerSample = 10

var benchmarkMicroSinkFrame any

type benchmarkMicroResult struct {
	Metrics []benchmarkMicroMetric `json:"metrics"`
}

type benchmarkMicroMetric struct {
	Metric         string                `json:"metric"`
	Group          string                `json:"group"`
	X              int                   `json:"x"`
	XLabel         string                `json:"x_label"`
	Samples        int                   `json:"samples"`
	MedianNS       float64               `json:"median_ns"`
	MinNS          float64               `json:"min_ns"`
	MaxNS          float64               `json:"max_ns"`
	NSDistribution benchmarkDistribution `json:"ns_distribution"`
	Values         int                   `json:"values,omitempty"`
	Streams        int                   `json:"streams,omitempty"`
	BatchSize      int                   `json:"batch_size,omitempty"`
}

func TestBenchmarkMicroCurves(t *testing.T) {
	output := os.Getenv("BENCHMARK_MICRO_OUTPUT")
	if output == "" {
		t.Skip("BENCHMARK_MICRO_OUTPUT not set")
	}

	result := benchmarkMicroResult{}
	for _, size := range []int{1, 2, 3, 4, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000} {
		numeric := benchmarkMicroBuffer(size, false)
		discrete := benchmarkMicroBuffer(size, true)
		result.Metrics = append(result.Metrics, benchmarkMicroMeasureFrame("frame_numeric_full", "Frame tool: numeric full frame", size, 5000, func() {
			benchmarkMicroSinkFrame = tools.ConvertBufferToFrame(numeric, "/BENCH/VALUE", false, false, false)
		}))
		if size > 3 {
			result.Metrics = append(result.Metrics, benchmarkMicroMeasureFrame("frame_numeric_average", "Frame tool: numeric average frame", size, 5000, func() {
				benchmarkMicroSinkFrame = tools.ConvertBufferToAverageFrame(numeric, "/BENCH/VALUE", false, false, false)
			}))
			result.Metrics = append(result.Metrics, benchmarkMicroMeasureFrame("frame_numeric_average_minmax", "Frame tool: numeric average frame + min/max", size, 5000, func() {
				benchmarkMicroSinkFrame = tools.ConvertBufferToAverageFrame(numeric, "/BENCH/VALUE", true, true, false)
			}))
		}
		result.Metrics = append(result.Metrics, benchmarkMicroMeasureFrame("frame_discrete", "Frame tool: discrete frame", size, 2500, func() {
			benchmarkMicroSinkFrame = tools.ConvertDiscreteBufferToFrame(discrete, "/BENCH/STATE", true, false)
		}))
	}

	for _, streams := range []int{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000} {
		result.Metrics = append(result.Metrics, benchmarkMicroMeasureProcess(streams, 5000))
	}

	file, err := os.Create(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatal(err)
	}
}

func benchmarkMicroMeasureFrame(metric string, group string, values int, samples int, fn func()) benchmarkMicroMetric {
	durations := make([]int64, samples)
	for i := 0; i < samples; i++ {
		started := time.Now()
		fn()
		durations[i] = time.Since(started).Nanoseconds()
	}
	return benchmarkMicroPoint(metric, group, values, "Number of values in buffer", samples, durations)
}

func benchmarkMicroMeasureProcess(streams int, samples int) benchmarkMicroMetric {
	endpoint := benchmarkMicroEndpoint(streams)
	listener := endpoint.getChannelParameterListener()
	values := make([]client.ParameterValue, microProcessValuesPerSample)
	for i := range values {
		values[i] = benchmarkMicroNumericValue(i)
	}
	durations := make([]int64, samples)
	for i := 0; i < samples; i++ {
		started := time.Now()
		for _, value := range values {
			if err := listener("/BENCH/VALUE", value); err != nil {
				panic(err)
			}
		}
		durations[i] = time.Since(started).Nanoseconds()
	}
	point := benchmarkMicroPoint(
		"process_stream_10_values",
		"Process stream: 10 incoming values into N buffers",
		streams,
		"Number of stream buffers receiving each value",
		samples,
		durations,
	)
	point.Streams = streams
	point.BatchSize = microProcessValuesPerSample
	return point
}

// benchmarkDistribution bundles a full statistical summary (median/min/max
// plus a percentile spread) so a metric declares a single nested field
// instead of one flat field per statistic. Mirrors the scenario simulator's
// distributionStats (scripts/benchmarks/simulator/scenario.go) and the
// Grafana benchmark spec's DistributionStats.
type benchmarkDistribution struct {
	Median float64 `json:"median"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	P1     float64 `json:"p1"`
	P5     float64 `json:"p5"`
	P30    float64 `json:"p30"`
	P70    float64 `json:"p70"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
}

// computeBenchmarkDistribution expects sorted to already be sorted ascending.
func computeBenchmarkDistribution(sorted []int64) benchmarkDistribution {
	if len(sorted) == 0 {
		return benchmarkDistribution{}
	}
	percentile := func(p float64) float64 {
		if len(sorted) == 1 {
			return float64(sorted[0])
		}
		rank := p * float64(len(sorted)-1)
		lower := int(rank)
		if lower >= len(sorted)-1 {
			return float64(sorted[len(sorted)-1])
		}
		fraction := rank - float64(lower)
		return float64(sorted[lower]) + fraction*float64(sorted[lower+1]-sorted[lower])
	}
	middle := len(sorted) / 2
	median := float64(sorted[middle])
	if len(sorted)%2 == 0 {
		median = float64(sorted[middle-1]+sorted[middle]) / 2
	}
	return benchmarkDistribution{
		Median: median,
		Min:    float64(sorted[0]),
		Max:    float64(sorted[len(sorted)-1]),
		P1:     percentile(0.01),
		P5:     percentile(0.05),
		P30:    percentile(0.30),
		P70:    percentile(0.70),
		P95:    percentile(0.95),
		P99:    percentile(0.99),
	}
}

func benchmarkMicroPoint(metric string, group string, x int, xLabel string, samples int, durations []int64) benchmarkMicroMetric {
	sorted := append([]int64(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	distribution := computeBenchmarkDistribution(sorted)
	return benchmarkMicroMetric{
		Metric:         metric,
		Group:          group,
		X:              x,
		XLabel:         xLabel,
		Samples:        samples,
		MedianNS:       distribution.Median,
		MinNS:          distribution.Min,
		MaxNS:          distribution.Max,
		NSDistribution: distribution,
		Values:         x,
	}
}

func benchmarkMicroEndpoint(streams int) *YamcsEndpoint {
	endpoint := &YamcsEndpoint{
		Configuration: &config.YamcsEndpointConfiguration{Instance: "bench", Processor: "realtime"},
		Parameters:    map[string]*ParameterDemand{},
	}
	parameter := &ParameterDemand{
		endpoint: endpoint,
		Name:     "/BENCH/VALUE",
		Streams:  map[string]*ParameterStreamDemand{},
		Ring:     types.NewRing[*pvalue.ParameterValue](ParameterRingCapacity),
	}
	endpoint.Parameters[parameter.Name] = parameter
	for i := 0; i < streams; i++ {
		path := fmt.Sprintf("bench/stream-%d", i)
		parameter.Streams[path] = &ParameterStreamDemand{
			parameter: parameter,
			Path:      path,
		}
	}
	return endpoint
}

func benchmarkMicroBuffer(size int, discrete bool) []client.ParameterValue {
	buffer := make([]client.ParameterValue, size)
	for i := range buffer {
		if discrete {
			buffer[i] = benchmarkMicroDiscreteValue(i)
		} else {
			buffer[i] = benchmarkMicroNumericValue(i)
		}
	}
	return buffer
}

func benchmarkMicroNumericValue(i int) *pvalue.ParameterValue {
	value := float64(i) * 1.25
	status := pvalue.AcquisitionStatus_ACQUIRED
	return &pvalue.ParameterValue{
		GenerationTime:    timestamppb.New(time.Unix(1700000000+int64(i), int64(i%1000))),
		AcquisitionStatus: &status,
		EngValue: &protobuf.Value{
			Type:        protobuf.Value_DOUBLE.Enum(),
			DoubleValue: &value,
		},
	}
}

func benchmarkMicroDiscreteValue(i int) *pvalue.ParameterValue {
	value := fmt.Sprintf("state-%d", i%8)
	status := pvalue.AcquisitionStatus_ACQUIRED
	return &pvalue.ParameterValue{
		GenerationTime:    timestamppb.New(time.Unix(1700000000+int64(i), int64(i%1000))),
		AcquisitionStatus: &status,
		EngValue: &protobuf.Value{
			Type:        protobuf.Value_STRING.Enum(),
			StringValue: &value,
		},
	}
}

func BenchmarkMicroCurves(b *testing.B) {
	runtime.KeepAlive(benchmarkMicroSinkFrame)
}

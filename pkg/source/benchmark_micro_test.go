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
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/yamcs/client"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const microProcessValuesPerSample = 10

var benchmarkMicroSinkFrame any

type benchmarkMicroResult struct {
	Metrics []benchmarkMicroMetric `json:"metrics"`
}

type benchmarkMicroMetric struct {
	Metric    string  `json:"metric"`
	Group     string  `json:"group"`
	X         int     `json:"x"`
	XLabel    string  `json:"x_label"`
	Samples   int     `json:"samples"`
	MedianNS  float64 `json:"median_ns"`
	Values    int     `json:"values,omitempty"`
	Streams   int     `json:"streams,omitempty"`
	BatchSize int     `json:"batch_size,omitempty"`
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
		if i%128 == 0 {
			for _, stream := range endpoint.Parameters["/BENCH/VALUE"].Streams {
				stream.Buffer = stream.Buffer[:0]
			}
		}
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

func benchmarkMicroPoint(metric string, group string, x int, xLabel string, samples int, durations []int64) benchmarkMicroMetric {
	sorted := append([]int64(nil), durations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	median := float64(0)
	if len(sorted) > 0 {
		middle := len(sorted) / 2
		if len(sorted)%2 == 0 {
			median = float64(sorted[middle-1]+sorted[middle]) / 2
		} else {
			median = float64(sorted[middle])
		}
	}
	return benchmarkMicroMetric{
		Metric:   metric,
		Group:    group,
		X:        x,
		XLabel:   xLabel,
		Samples:  samples,
		MedianNS: median,
		Values:   x,
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
	}
	endpoint.Parameters[parameter.Name] = parameter
	for i := 0; i < streams; i++ {
		path := fmt.Sprintf("bench/stream-%d", i)
		parameter.Streams[path] = &ParameterStreamDemand{
			parameter: parameter,
			Path:      path,
			Buffer:    make([]client.ParameterValue, 0, microProcessValuesPerSample),
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

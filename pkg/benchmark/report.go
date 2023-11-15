package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type LoaderReport struct {
	QPS         float64              `json:"QPS"`
	TotalCount  int                  `json:"TotalCount"`
	FailedCount int                  `json:"FailedCount"`
	PercLatency *latencyInPercentile `json:"PercLatency"`
	Unit        string               `json:"Unit"`
}

func (lr *LoaderReport) Report(title string) {
	fmt.Printf("=== %s ===\n", title)
	fmt.Printf("%s", prettyJSON(lr))
	fmt.Println("=== END ===")
}

func prettyJSON(metrics interface{}) string {
	output := &bytes.Buffer{}
	if err := json.NewEncoder(output).Encode(metrics); err != nil {
		klog.Errorf("Error building encoder: %v", err)
		return ""
	}
	formatted := &bytes.Buffer{}
	if err := json.Indent(formatted, output.Bytes(), "", "  "); err != nil {
		klog.Errorf("Error indenting: %v", err)
		return ""
	}
	return string(formatted.Bytes())
}

type latencyInPercentile struct {
	Perc50  float64 `json:"Perc50"`
	Perc75  float64 `json:"Perc75"`
	Perc90  float64 `json:"Perc90"`
	Perc99  float64 `json:"Perc99"`
	Perc100 float64 `json:"Perc100"`
}

func extractLatencyMetrics(latencies []metav1.Duration) *latencyInPercentile {
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i].Duration < latencies[j].Duration
	})
	length := len(latencies)
	if length <= 0 {
		return &latencyInPercentile{}
	}
	perc50 := latencies[int(math.Ceil(float64(length*50)/100))-1].Duration.Seconds()
	perc75 := latencies[int(math.Ceil(float64(length*75)/100))-1].Duration.Seconds()
	perc90 := latencies[int(math.Ceil(float64(length*90)/100))-1].Duration.Seconds()
	perc99 := latencies[int(math.Ceil(float64(length*99)/100))-1].Duration.Seconds()
	perc100 := latencies[length-1].Duration.Seconds()
	return &latencyInPercentile{
		Perc50:  perc50,
		Perc75:  perc75,
		Perc90:  perc90,
		Perc99:  perc99,
		Perc100: perc100,
	}
}

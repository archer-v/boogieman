package scheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"testing"
)

func Test_probeDataMetric(t *testing.T) {

	type expectedResult struct {
		valueType  prometheus.ValueType
		value      float64
		labels     []string
		labelNames []string
	}

	type testCase struct {
		data           any
		expectedResult []expectedResult
	}

	var testString = "test"
	type structData struct {
		Timings  map[string]int    `json:"timings"`
		Regex    map[string]bool   `json:"regex,omitempty"`
		Captures map[string]string `json:"captures,omitempty"`
		Capture  string            `json:"capture,omitempty"`
		Empty    string            `json:"empty,omitempty"`
		Ignored  string            `json:"-"`
	}
	cases := []testCase{
		{
			data: 10,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(10),
					[]string{},
					nil,
				},
			},
		},
		{
			data: true,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{},
					nil,
				},
			},
		},
		{
			data: testString,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{testString},
					[]string{"item"},
				},
			},
		},
		{
			data: 10.5,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					10.5,
					[]string{},
					nil,
				},
			},
		},
		{
			data: &testString,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{testString},
					[]string{"item"},
				},
			},
		},
		{
			data: map[string]int{
				"val1": 20,
				"val2": 30,
			},
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(20),
					[]string{"val1"},
					[]string{"item"},
				},
				{
					prometheus.GaugeValue,
					float64(30),
					[]string{"val2"},
					[]string{"item"},
				},
			},
		},
		{
			data: map[string]string{
				"version": "1.2.3",
				"status":  "ok",
			},
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{"status", "ok"},
					[]string{"item", "value"},
				},
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{"version", "1.2.3"},
					[]string{"item", "value"},
				},
			},
		},
		{
			data: structData{
				Timings: map[string]int{
					"https://example.com/": 42,
				},
				Regex: map[string]bool{
					"https://example.com/":   true,
					"https://empty.example/": false,
				},
				Captures: map[string]string{
					"https://example.com/":   "1.2.3",
					"https://empty.example/": "",
				},
				Capture: "1.2.3",
				Empty:   "",
				Ignored: "hidden",
			},
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(42),
					[]string{"timings", "https://example.com/"},
					[]string{"field", "item"},
				},
				{
					prometheus.GaugeValue,
					float64(0),
					[]string{"regex", "https://empty.example/"},
					[]string{"field", "item"},
				},
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{"regex", "https://example.com/"},
					[]string{"field", "item"},
				},
				{
					prometheus.GaugeValue,
					float64(0),
					[]string{"captures", "https://empty.example/", ""},
					[]string{"field", "item", "value"},
				},
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{"captures", "https://example.com/", "1.2.3"},
					[]string{"field", "item", "value"},
				},
				{
					prometheus.GaugeValue,
					float64(1),
					[]string{"capture", "1.2.3"},
					[]string{"field", "value"},
				},
				{
					prometheus.GaugeValue,
					float64(0),
					[]string{"empty", ""},
					[]string{"field", "value"},
				},
			},
		},
	}

	for i, c := range cases {
		metrics := probeMetrics(c.data)
		if len(metrics) != len(c.expectedResult) {
			t.Errorf("Case %v should return %v metrics, but got %v", i, len(c.expectedResult), len(metrics))
			continue
		}
		for j, m := range metrics {
			exp := c.expectedResult[j]
			if m.valueType != exp.valueType {
				t.Errorf("Case %v should return valueType %v, but got %v", i, exp.valueType, m.valueType)
				continue
			}
			if m.value != exp.value {
				t.Errorf("Case %v should return value %v, but got %v", i, exp.value, m.value)
				continue
			}
			if len(m.labels) != len(exp.labels) {
				t.Errorf("Case %v should return %v labels, but got %v", i, len(exp.labels), len(m.labels))
				continue
			}
			for idx, l := range m.labels {
				if l != exp.labels[idx] {
					t.Errorf("Case %v should return labels %v at index %v, but got %v", i, exp.labels[idx], idx, l)
					continue
				}
			}
			if len(m.labelNames) != len(exp.labelNames) {
				t.Errorf("Case %v should return %v label names, but got %v", i, len(exp.labelNames), len(m.labelNames))
				continue
			}
			for idx, l := range m.labelNames {
				if l != exp.labelNames[idx] {
					t.Errorf("Case %v should return label name %v at index %v, but got %v", i, exp.labelNames[idx], idx, l)
					continue
				}
			}
		}
	}
}

func Test_sendMetricProbeDataItemWithDynamicLabels(t *testing.T) {
	pDescriptors = map[string]*prometheus.Desc{}
	s := &Scheduler{}
	ch := make(chan prometheus.Metric, 1)

	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("sendMetric should not panic on dynamic probe data labels: %v", err)
		}
	}()

	s.sendMetric(
		ch,
		[]string{pNameDataItem, "field,item"},
		metricData{
			valueType:  prometheus.GaugeValue,
			value:      42,
			labels:     []string{"job", "script.yml", "task", "web", "timings", "https://example.com/"},
			labelNames: []string{"field", "item"},
		},
	)
}

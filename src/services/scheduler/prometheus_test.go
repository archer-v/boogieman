package scheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"testing"
)

func Test_probeDataMetric(t *testing.T) {

	type expectedResult struct {
		valueType prometheus.ValueType
		value     float64
		labels    []string
	}

	type testCase struct {
		data           any
		expectedResult []expectedResult
	}

	addLabel := func(arr []string, s string) []string {
		r := make([]string, len(arr)+1)
		copy(r, arr)
		r[len(arr)] = s
		return r
	}
	var testString = "test"
	labels := []string{"one", "two"}
	cases := []testCase{
		{
			data: 10,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(10),
					labels,
				},
			},
		},
		{
			data: true,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(1),
					labels,
				},
			},
		},
		{
			data: testString,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(1),
					addLabel(labels, testString),
				},
			},
		},
		{
			data: 10.5,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					10.5,
					labels,
				},
			},
		},
		{
			data: &testString,
			expectedResult: []expectedResult{
				{
					prometheus.GaugeValue,
					float64(1),
					addLabel(labels, testString),
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
					addLabel(labels, "val1"),
				},
				{
					prometheus.GaugeValue,
					float64(30),
					addLabel(labels, "val2"),
				},
			},
		},
	}

	for i, c := range cases {
		metrics := probeMetrics(c.data, labels)
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
					t.Errorf("Case %v should return label %v at index %v, but got %v", i, exp.labels[idx], idx, l)
					continue
				}
			}
		}
	}
}

package model

import (
	"encoding/json"
)

type TaskMetric struct {
	Labels   MetricLabels
	ValueMap MetricLabelsValueMap
}

type MetricLabels struct {
	data map[string]string
	key  string
}

type MetricLabelsValueMap map[string]string

func (s *MetricLabels) UnmarshalJSON(b []byte) (err error) {
	data := MetricLabels{data: map[string]string{}}

	err = json.Unmarshal(b, &data.data)
	if err != nil {
		return
	}

	*s = data

	return
}

func (s *MetricLabels) Data() map[string]string {
	return s.data
}

func (s *MetricLabels) IsEmpty() bool {
	return !(len(s.data) > 0)
}

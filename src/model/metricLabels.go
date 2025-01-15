package model

import (
	"encoding/json"
	"sort"
	"strings"
)

type MetricLabels struct {
	data map[string]string
	key  string
}

func (s *MetricLabels) UnmarshalJSON(b []byte) (err error) {
	data := MetricLabels{data: map[string]string{}}

	err = json.Unmarshal(b, &data.data)
	if err != nil {
		return
	}

	*s = data

	return
}

func (s *MetricLabels) CombinedKey() string {
	if s.key == "" && len(s.data) > 0 {
		keys := make([]string, 0, len(s.data))
		for k := range s.data {
			keys = append(keys, k)
			sort.Strings(keys)
		}
		s.key = strings.Join(keys, "|")
	}
	return s.key
}

func (s *MetricLabels) Data() map[string]string {
	return s.data
}

func (s *MetricLabels) IsEmpty() bool {
	return !(len(s.data) > 0)
}

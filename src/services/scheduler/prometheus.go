package scheduler

import (
	"boogieman/src/model"
	"github.com/prometheus/client_golang/prometheus"
	"reflect"
	"strconv"
	"strings"
)

var (
	probeDataGeneralLabels     = []string{"job", "script", "task", "probe"}
	probeDateItemGeneralLabels = []string{"job", "script", "task", "probe", "item"}
	taskGeneralLabels          = []string{"job", "script", "task"}
)

const (
	probeDataHelpDescr         = "probe execution data result"
	probeDataName              = "boogieman_probe_data"
	probeDataItemName          = "boogieman_probe_data_item"
	descriptorKeyProbeData     = "probe_data"
	descriptorKeyProbeDataItem = "probe_data_item"
)

// prometheus metrics descriptors
var pDescriptors = map[string]*prometheus.Desc{
	"script_result":            prometheus.NewDesc("boogieman_script_result", "script execution result", []string{"job", "script"}, nil),
	"task_result":              prometheus.NewDesc("boogieman_task_result", "task execution result", taskGeneralLabels, nil),
	"task_runtime":             prometheus.NewDesc("boogieman_task_runtime", "task runtime", taskGeneralLabels, nil),
	"task_runs":                prometheus.NewDesc("boogieman_task_runs", "task run counter", taskGeneralLabels, nil),
	descriptorKeyProbeData:     prometheus.NewDesc(probeDataName, probeDataHelpDescr, probeDataGeneralLabels, nil),
	descriptorKeyProbeDataItem: prometheus.NewDesc(probeDataItemName, probeDataHelpDescr, probeDateItemGeneralLabels, nil),
}

// probe metrics struct
type probeDataMetric struct {
	valueType prometheus.ValueType
	value     float64
	labels    []string
}

// Describe - implementation of prometheus.Collector interface
func (s *Scheduler) Describe(chan<- *prometheus.Desc) {

}

// Collect - implementation of prometheus.Collector interface
// invokes metrics from each job and tasks, prepares and send data to prometheus module
func (s *Scheduler) Collect(ch chan<- prometheus.Metric) {
	s.Lock()
	defer s.Unlock()
	for _, j := range s.jobs {
		scriptResult := j.Script.ResultFinished()
		ch <- prometheus.MustNewConstMetric(
			pDescriptors["script_result"],
			prometheus.GaugeValue, gbValue(scriptResult.Success),
			j.Name, j.ScriptFile,
		)
		for _, t := range scriptResult.Tasks {
			// task general metrics
			taskMetricLabelValues := []string{j.Name, j.ScriptFile, t.Name}
			ch <- prometheus.MustNewConstMetric(
				pDescriptors["task_result"],
				prometheus.GaugeValue, gbValue(t.Success),
				taskMetricLabelValues...,
			)
			ch <- prometheus.MustNewConstMetric(
				pDescriptors["task_runtime"],
				prometheus.GaugeValue, float64(t.RuntimeMs),
				taskMetricLabelValues...,
			)
			ch <- prometheus.MustNewConstMetric(
				pDescriptors["task_runs"],
				prometheus.CounterValue, float64(t.RunCounter),
				taskMetricLabelValues...,
			)
			// task data metrics
			if t.Probe.Data != nil {
				// check if there are additional metric labels for this task
				var taskLabels model.MetricLabels
				for _, task := range j.Script.Tasks {
					if task.Name == t.Name {
						taskLabels = task.MetricLabels
						break
					}
				}
				if !taskLabels.IsEmpty() {

				}
				dataMetricLabelValues := taskMetricLabelValues[:]
				dataMetricLabelValues = append(dataMetricLabelValues, t.Probe.Name)
				ms := probeMetrics(t.Probe.Data)
				for _, m := range ms {
					var (
						labelValues    []string
						pDescriptorKey string
					)
					if len(m.labels) == 0 {
						labelValues = dataMetricLabelValues
						pDescriptorKey = descriptorKeyProbeData
					} else {
						labelValues = dataMetricLabelValues[:]
						labelValues = append(labelValues, m.labels...)
						pDescriptorKey = descriptorKeyProbeDataItem
					}
					// has dynamic labels
					if !taskLabels.IsEmpty() {
						descriptorKey := strings.Join(taskMetricLabelValues, "|")
						if _, exists := pDescriptors[descriptorKey]; !exists {
							var (
								labels []string
								name   string
							)

							switch pDescriptorKey {
							case descriptorKeyProbeData:
								labels = probeDataGeneralLabels
								name = probeDataName
							case descriptorKeyProbeDataItem:
								labels = probeDateItemGeneralLabels
								name = probeDataItemName
							}

							pDescriptors[descriptorKey] =
								prometheus.NewDesc(name, probeDataHelpDescr, labels, taskLabels.Data())
						}
						pDescriptorKey = descriptorKey
					}
					ch <- prometheus.MustNewConstMetric(pDescriptors[pDescriptorKey], m.valueType, m.value, labelValues...)
				}
			}
		}
	}
}

// gbValue returns a gauge value for boolean data (1 for true, 0 - false)
func gbValue(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func addToArray(arr []string, s string) []string {
	r := make([]string, len(arr)+1)
	copy(r, arr)
	r[len(arr)] = s
	return r
}

func reflectIsInt(kind reflect.Kind) bool {
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64
}

func reflectIsFloat(kind reflect.Kind) bool {
	return kind == reflect.Float32 || kind == reflect.Float64
}

// probeMetrics returns a slice of probe metrics
// data can be a simple value or hash of names and values
//
//nolint:funlen
func probeMetrics(data any) (metrics []probeDataMetric) {
	var (
		valueType = prometheus.GaugeValue
		labels    []string
		value     float64
	)

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	kind := v.Kind()

	// if data is a map
	if kind == reflect.Map {
		iter := v.MapRange()
		stop := false
		for iter.Next() && !stop {
			k := iter.Key()
			v := iter.Value()
			switch {
			case reflectIsInt(k.Kind()):
				labels = []string{strconv.Itoa(int(k.Int()))}
			case k.Kind() == reflect.String:
				labels = []string{k.String()}
			default:
				// unsupported type of map key
				stop = true
				continue
			}

			switch {
			case reflectIsInt(v.Kind()):
				value = float64(v.Int())
			case reflectIsFloat(v.Kind()):
				value = v.Float()
			default:
				// unsupported type of map value
				stop = true
				continue
			}

			metrics = append(metrics, probeDataMetric{
				valueType: valueType,
				value:     value,
				labels:    labels,
			})
		}
		return metrics
	}

	// data contains simple value
	switch {
	case kind == reflect.Bool:
		value = gbValue(v.Bool())
	case reflectIsInt(kind):
		value = float64(v.Int())
	case reflectIsFloat(kind):
		value = v.Float()
	case kind == reflect.String:
		value = 1
		labels = []string{v.String()}
	}

	return []probeDataMetric{
		{
			valueType: valueType,
			value:     value,
			labels:    labels,
		},
	}
}

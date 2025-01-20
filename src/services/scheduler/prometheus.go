package scheduler

import (
	"boogieman/src/model"
	"github.com/prometheus/client_golang/prometheus"
	"reflect"
	"strconv"
	"strings"
)

var (
	LabelsScriptGeneral        = []string{"job", "script"}
	LabelsTaskGeneral          = []string{"job", "script", "task"}
	LabelsProbeDataGeneral     = []string{"job", "script", "task", "probe"}
	LabelsProbeDateItemGeneral = []string{"job", "script", "task", "probe", "item"}
)

const (
	probeDataHelpDescr = "probe execution data result"
	pNameData          = "boogieman_probe_data"
	pNameDataItem      = "boogieman_probe_data_item"
	pNameScriptResult  = "boogieman_script_result"
	pNameTaskResult    = "boogieman_task_result"
	pNameTaskRuntime   = "boogieman_task_runtime"
	pNameTaskRuns      = "boogieman_task_runs"
)

// probe metrics struct
type metricData struct {
	valueType   prometheus.ValueType
	value       float64
	labels      []string
	constLabels prometheus.Labels
}

type metricDescriptorInfo struct {
	name   string
	help   string
	labels []string
}

var metricBaseDescriptors = map[string]metricDescriptorInfo{
	pNameScriptResult: {
		pNameScriptResult, "script execution result", LabelsScriptGeneral,
	},
	pNameTaskResult: {
		pNameTaskResult, "task execution result", LabelsTaskGeneral,
	},
	pNameTaskRuntime: {
		pNameTaskRuntime, "task runtime", LabelsTaskGeneral,
	},
	pNameTaskRuns: {
		pNameTaskRuns, "task run counter", LabelsTaskGeneral,
	},
	pNameData: {
		pNameData, probeDataHelpDescr, LabelsProbeDataGeneral,
	},
	pNameDataItem: {
		pNameDataItem, probeDataHelpDescr, LabelsProbeDateItemGeneral,
	},
}

// prometheus metrics descriptors
var pDescriptors = map[string]*prometheus.Desc{}

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
		s.sendMetric(ch, []string{pNameScriptResult},
			metricData{prometheus.GaugeValue, gbValue(scriptResult.Success), []string{j.Name, j.ScriptFile}, nil},
		)
		for _, t := range scriptResult.Tasks {
			// task general metrics
			taskMetricLabelValues := []string{j.Name, j.ScriptFile, t.Name}

			// check if there are additional metric labels for this task
			var taskMetric model.TaskMetric
			for _, task := range j.Script.Tasks {
				if task.Name == t.Name {
					taskMetric = task.Metric
					break
				}
			}
			s.sendMetric(
				ch, []string{pNameTaskResult},
				metricData{prometheus.GaugeValue, gbValue(t.Success), taskMetricLabelValues, taskMetric.Labels.Data()})
			s.sendMetric(
				ch, []string{pNameTaskRuntime},
				metricData{prometheus.GaugeValue, float64(t.RuntimeMs), taskMetricLabelValues, taskMetric.Labels.Data()})
			s.sendMetric(
				ch, []string{pNameTaskRuns},
				metricData{prometheus.CounterValue, float64(t.RunCounter), taskMetricLabelValues, taskMetric.Labels.Data()})

			// task data metrics
			if t.Probe.Data != nil {
				// add probe name to metric labels
				dataMetricLabelValues := taskMetricLabelValues[:]
				dataMetricLabelValues = append(dataMetricLabelValues, t.Probe.Name)
				ms := probeMetrics(t.Probe.Data)
				for _, m := range ms {
					var (
						labelValues    []string
						pDescriptorKey []string
					)
					if len(m.labels) == 0 {
						labelValues = dataMetricLabelValues
						pDescriptorKey = []string{pNameData}
					} else {
						labelValues = dataMetricLabelValues[:]
						if len(taskMetric.ValueMap) > 0 {
							val, ok := taskMetric.ValueMap[m.labels[0]]
							if ok {
								m.labels[0] = val
							}
						}
						labelValues = append(labelValues, m.labels...)
						pDescriptorKey = []string{pNameDataItem}
					}
					// if task has custom labels
					if !taskMetric.Labels.IsEmpty() {
						pDescriptorKey = append(pDescriptorKey, taskMetricLabelValues...)
					}
					s.sendMetric(
						ch, pDescriptorKey,
						metricData{
							valueType: m.valueType, value: m.value, labels: labelValues, constLabels: taskMetric.Labels.Data(),
						})
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

func (s *Scheduler) sendMetric(ch chan<- prometheus.Metric, descrCompositeKey []string, metricData metricData) {

	var (
		descrKey string
		pDescr   *prometheus.Desc
		ok       bool
	)

	// metric descriptor key is a concatenation of base metric descriptor key and task labels
	if len(descrCompositeKey) == 1 && len(metricData.constLabels) > 0 {
		descrCompositeKey = append(descrCompositeKey, metricData.labels...)
	}

	if len(descrCompositeKey) == 1 {
		descrKey = descrCompositeKey[0]
	} else {
		descrKey = strings.Join(descrCompositeKey, "|")
	}

	// check if metric descriptor already exists
	pDescr, ok = pDescriptors[descrKey]
	if !ok { // metric descriptor not found, create it
		descrInfo, ok := metricBaseDescriptors[descrKey]
		if ok {
			pDescr = prometheus.NewDesc(descrKey, descrInfo.help, descrInfo.labels, metricData.constLabels)
			pDescriptors[descrKey] = pDescr
		} else {
			descrInfo, ok = metricBaseDescriptors[descrCompositeKey[0]]
			if !ok {
				s.logger.Printf("unknown base metric descriptor: %s", descrCompositeKey[0])
				return
			}
			pDescr =
				prometheus.NewDesc(descrCompositeKey[0], descrInfo.help, descrInfo.labels, metricData.constLabels)
			pDescriptors[descrKey] = pDescr
		}
	}
	ch <- prometheus.MustNewConstMetric(
		pDescr,
		metricData.valueType, metricData.value,
		metricData.labels...,
	)
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
func probeMetrics(data any) (metrics []metricData) {
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

			metrics = append(metrics, metricData{
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

	return []metricData{
		{
			valueType: valueType,
			value:     value,
			labels:    labels,
		},
	}
}

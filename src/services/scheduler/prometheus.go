package scheduler

import (
	"boogieman/src/model"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
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
	labelNames  []string
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
//
//nolint:funlen
func (s *Scheduler) Collect(ch chan<- prometheus.Metric) {
	s.Lock()
	defer s.Unlock()
	for _, j := range s.jobs {
		scriptResult := j.Script.ResultFinished()
		s.sendMetric(ch, []string{pNameScriptResult},
			metricData{prometheus.GaugeValue, gbValue(scriptResult.Success), []string{j.Name, j.ScriptFile}, nil, nil},
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
				metricData{prometheus.GaugeValue, gbValue(t.Success), taskMetricLabelValues, nil, taskMetric.Labels.Data()})
			s.sendMetric(
				ch, []string{pNameTaskRuntime},
				metricData{prometheus.GaugeValue, float64(t.RuntimeMs), taskMetricLabelValues, nil, taskMetric.Labels.Data()})
			s.sendMetric(
				ch, []string{pNameTaskRuns},
				metricData{prometheus.CounterValue, float64(t.RunCounter), taskMetricLabelValues, nil, taskMetric.Labels.Data()})

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
							for i, labelName := range m.labelNames {
								if labelName != "item" {
									continue
								}
								val, ok := taskMetric.ValueMap[m.labels[i]]
								if ok {
									m.labels[i] = val
								}
								break
							}
						}
						labelValues = append(labelValues, m.labels...)
						pDescriptorKey = []string{pNameDataItem, strings.Join(m.labelNames, ",")}
					}
					// if task has custom labels
					if !taskMetric.Labels.IsEmpty() {
						pDescriptorKey = append(pDescriptorKey, taskMetricLabelValues...)
					}
					s.sendMetric(
						ch, pDescriptorKey,
						metricData{
							valueType: m.valueType, value: m.value, labels: labelValues,
							labelNames: m.labelNames, constLabels: taskMetric.Labels.Data(),
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
			labels := descrInfo.labels
			if descrCompositeKey[0] == pNameDataItem && len(metricData.labelNames) > 0 {
				labels = append(append([]string{}, LabelsProbeDataGeneral...), metricData.labelNames...)
			}
			pDescr =
				prometheus.NewDesc(descrCompositeKey[0], descrInfo.help, labels, metricData.constLabels)
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

func reflectIsString(kind reflect.Kind) bool {
	return kind == reflect.String
}

// probeMetrics returns a slice of probe metrics
// data can be a simple value or hash of names and values
//
//nolint:funlen
func probeMetrics(data any) (metrics []metricData) {
	var (
		valueType  = prometheus.GaugeValue
		labels     []string
		labelNames []string
		value      float64
	)

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	kind := v.Kind()

	if kind == reflect.Struct {
		return probeStructMetrics(v)
	}

	// if data is a map
	if kind == reflect.Map {
		stop := false
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool {
			return mapKeyString(keys[i]) < mapKeyString(keys[j])
		})
		for _, k := range keys {
			if stop {
				break
			}
			v := v.MapIndex(k)
			switch {
			case reflectIsInt(k.Kind()):
				labels = []string{strconv.Itoa(int(k.Int()))}
				labelNames = []string{"item"}
			case k.Kind() == reflect.String:
				labels = []string{k.String()}
				labelNames = []string{"item"}
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
			case reflectIsString(v.Kind()):
				value = 1
				labels = append(labels, v.String())
				labelNames = append(labelNames, "value")
			default:
				// unsupported type of map value
				stop = true
				continue
			}

			metrics = append(metrics, metricData{
				valueType:  valueType,
				value:      value,
				labels:     labels,
				labelNames: labelNames,
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
		labelNames = []string{"item"}
	}

	return []metricData{
		{
			valueType:  valueType,
			value:      value,
			labels:     labels,
			labelNames: labelNames,
		},
	}
}

//nolint:funlen
func probeStructMetrics(v reflect.Value) (metrics []metricData) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		fieldName := exportedFieldName(field)
		if fieldName == "" {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}

		if fv.Kind() == reflect.Map {
			metrics = append(metrics, probeStructMapMetrics(fieldName, fv)...)
			continue
		}

		m, ok := probeStructSimpleMetric(fieldName, fv)
		if !ok {
			continue
		}
		metrics = append(metrics, m)
	}
	return metrics
}

func probeStructMapMetrics(fieldName string, v reflect.Value) (metrics []metricData) {
	if v.Type().Key().Kind() != reflect.String {
		return nil
	}

	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return mapKeyString(keys[i]) < mapKeyString(keys[j])
	})
	for _, k := range keys {
		value := v.MapIndex(k)
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				continue
			}
			value = value.Elem()
		}

		m := metricData{
			valueType:  prometheus.GaugeValue,
			labels:     []string{fieldName, k.String()},
			labelNames: []string{"field", "item"},
		}
		switch {
		case reflectIsInt(value.Kind()):
			m.value = float64(value.Int())
		case reflectIsFloat(value.Kind()):
			m.value = value.Float()
		case value.Kind() == reflect.Bool:
			m.value = gbValue(value.Bool())
		case value.Kind() == reflect.String:
			m.value = 1
			m.labels = append(m.labels, value.String())
			m.labelNames = append(m.labelNames, "value")
		default:
			continue
		}
		metrics = append(metrics, m)
	}
	return metrics
}

func probeStructSimpleMetric(fieldName string, v reflect.Value) (metricData, bool) {
	m := metricData{
		valueType:  prometheus.GaugeValue,
		labels:     []string{fieldName},
		labelNames: []string{"field"},
	}
	switch {
	case v.Kind() == reflect.Bool:
		m.value = gbValue(v.Bool())
	case reflectIsInt(v.Kind()):
		m.value = float64(v.Int())
	case reflectIsFloat(v.Kind()):
		m.value = v.Float()
	case v.Kind() == reflect.String:
		m.value = 1
		m.labels = append(m.labels, v.String())
		m.labelNames = append(m.labelNames, "value")
	default:
		return metricData{}, false
	}
	return m, true
}

func mapKeyString(v reflect.Value) string {
	switch {
	case reflectIsInt(v.Kind()):
		return strconv.Itoa(int(v.Int()))
	case v.Kind() == reflect.String:
		return v.String()
	default:
		return ""
	}
}

func exportedFieldName(field reflect.StructField) string {
	jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
	switch jsonName {
	case "-":
		return ""
	case "":
		return field.Name
	default:
		return jsonName
	}
}

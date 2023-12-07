package scheduler

import (
	"github.com/prometheus/client_golang/prometheus"
	"reflect"
	"strconv"
)

var pDescriptors = map[string]*prometheus.Desc{
	"script_result":   prometheus.NewDesc("boogieman_script_result", "script execution result", []string{"job", "script"}, nil),
	"task_result":     prometheus.NewDesc("boogieman_task_result", "task execution result", []string{"job", "script", "task"}, nil),
	"task_runtime":    prometheus.NewDesc("boogieman_task_runtime", "task runtime", []string{"job", "script", "task"}, nil),
	"task_runs":       prometheus.NewDesc("boogieman_task_runs", "task run counter", []string{"job", "script", "task"}, nil),
	"probe_data":      prometheus.NewDesc("boogieman_probe_data", "probe execution data result", []string{"job", "script", "task", "probe"}, nil),
	"probe_data_item": prometheus.NewDesc("boogieman_probe_data_item", "probe execution data result", []string{"job", "script", "task", "probe", "item"}, nil),
}

type probeDataMetric struct {
	descr     *prometheus.Desc
	valueType prometheus.ValueType
	value     float64
	labels    []string
}

// Describe - implementation of prometheus.Collector interface
func (s *Scheduler) Describe(chan<- *prometheus.Desc) {

}

// Collect - implementation of prometheus.Collector interface
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
			labels := []string{j.Name, j.ScriptFile, t.Name}
			ch <- prometheus.MustNewConstMetric(
				pDescriptors["task_result"],
				prometheus.GaugeValue, gbValue(t.Success),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				pDescriptors["task_runtime"],
				prometheus.GaugeValue, float64(t.RuntimeMs),
				labels...,
			)
			ch <- prometheus.MustNewConstMetric(
				pDescriptors["task_runs"],
				prometheus.CounterValue, float64(t.RunCounter),
				labels...,
			)
			if t.Probe.Data != nil {
				ms := probeMetrics(t.Probe.Data, addToArray(labels, t.Probe.Name))
				for _, m := range ms {
					ch <- prometheus.MustNewConstMetric(m.descr, m.valueType, m.value, m.labels...)
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

//nolint:funlen
func probeMetrics(data any, labels []string) (metrics []probeDataMetric) {
	var (
		descr     *prometheus.Desc
		value     float64
		valueType = prometheus.GaugeValue
		l         = labels
	)

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	kind := v.Kind()

	switch {
	case kind == reflect.Bool:
		descr = pDescriptors["probe_data"]
		value = gbValue(v.Bool())
	case reflectIsInt(kind):
		descr = pDescriptors["probe_data"]
		value = float64(v.Int())
	case reflectIsFloat(kind):
		descr = pDescriptors["probe_data"]
		value = v.Float()
	case kind == reflect.String:
		descr = pDescriptors["probe_data_item"]
		value = 1
		l = addToArray(labels, v.String())
	}

	if descr != nil {
		return []probeDataMetric{
			{
				descr:     descr,
				valueType: valueType,
				value:     value,
				labels:    l,
			},
		}
	}

	if kind == reflect.Map {
		descr = pDescriptors["probe_data_item"]
		iter := v.MapRange()
		stop := false
		for iter.Next() && !stop {
			k := iter.Key()
			v := iter.Value()
			switch {
			case reflectIsInt(k.Kind()):
				l = addToArray(labels, strconv.Itoa(int(k.Int())))
			case k.Kind() == reflect.String:
				l = addToArray(labels, k.String())
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
				descr:     descr,
				valueType: valueType,
				value:     value,
				labels:    l,
			})
		}
		return metrics
	}

	return
}

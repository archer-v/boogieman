package configuration

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"encoding/json"
	"fmt"
	"github.com/creasty/defaults"
	"sigs.k8s.io/yaml" // uses it instead gopkg.in/yaml.v3 as it also supports json attributes in struct
)

type script struct {
	// Timeout time.Duration `default:"60s"`
	Script []task
}

type task struct {
	Name         string
	Probe        probe
	CGroup       string
	MetricLabels model.MetricLabels `json:"metricLabels"`
}

type probe struct {
	Name             string
	RawConfiguration *json.RawMessage `json:"configuration"`
	Configuration    any              `json:"-"`
	Options          model.ProbeOptions
	Expect           bool
}

func ScriptYMLConfiguration(data []byte, overrideConfigOptions ...map[string]map[string]string) (s *model.Script, err error) {
	s = &model.Script{}
	parsed := script{}
	if err = defaults.Set(&parsed); err != nil {
		return
	}
	if err = yaml.Unmarshal(data, &parsed); err != nil {
		return
	}

	var configOptions map[string]map[string]string
	if len(overrideConfigOptions) > 0 {
		configOptions = overrideConfigOptions[0]
	} else {
		configOptions = make(map[string]map[string]string)
	}
	var p model.Prober
	for _, t := range parsed.Script {
		var config any
		// get the probe configuration struct
		config, err = probefactory.NewProbeConfiguration(t.Probe.Name)
		if err != nil {
			err = fmt.Errorf("[%v] %w", t.Name, err)
			return
		}
		// try to fill probe config from RawConfiguration data
		if t.Probe.RawConfiguration != nil {
			if e := json.Unmarshal(*t.Probe.RawConfiguration, config); e == nil {
				// override config properties if there are options in configOptions for this task (task name)
				if o, ok := configOptions[t.Name]; ok {
					for fName, fValue := range o {
						if e = setStructField(config, fName, fValue); e != nil {
							err = fmt.Errorf("[%v] %w", t.Name, e)
							return
						}
					}
				}
			} else {
				// if unmarshal error, set probe config to raw data in order the probe try to parse raw data by itself
				config = []byte(*t.Probe.RawConfiguration)
			}
		}

		p, err = probefactory.NewProbe(t.Probe.Name, t.Probe.Options, config)
		if err != nil {
			err = fmt.Errorf("[%v] %w", t.Name, err)
			return
		}
		s.AddTask(model.NewTask(t.Name, t.CGroup, t.MetricLabels, p))
	}
	return
}

package configuration

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"encoding/json"
	"fmt"
	"github.com/creasty/defaults"
	"sigs.k8s.io/yaml" // uses it instead gopkg.in/yaml.v3 as it also supports json attributes in struct
)

type script struct {
	// Timeout time.Duration `default:"60s"`
	Script []record
}

type record struct {
	Name   string
	Probe  probe
	CGroup string
	// DependsOn      string
}

type probe struct {
	Name             string
	RawConfiguration *json.RawMessage `json:"configuration"`
	Configuration    any              `json:"-"`
	Options          model.ProbeOptions
	Expect           bool
}

func ScriptYMLConfiguration(data []byte) (s *model.Script, err error) {
	s = &model.Script{}
	parsed := script{}
	if err = defaults.Set(&parsed); err != nil {
		return
	}
	if err = yaml.Unmarshal(data, &parsed); err != nil {
		return
	}

	// s.Tasks = make([]*model.Task, len(parsed.script))
	var p model.Prober
	for _, v := range parsed.Script {
		var config any
		// get the probe configuration struct
		config, err = probeFactory.NewProbeConfiguration(v.Probe.Name)
		if err != nil {
			err = fmt.Errorf("[%v] %w", v.Name, err)
			return
		}
		// try to fill DaemonConfig from RawConfiguration data
		if v.Probe.RawConfiguration != nil {
			e := json.Unmarshal(*v.Probe.RawConfiguration, config)
			// if unmarshal error, set DaemonConfig to raw data in order the probe try to parse DaemonConfig by itself
			if e != nil {
				config = []byte(*v.Probe.RawConfiguration)
			}
		}

		p, err = probeFactory.NewProbe(v.Probe.Name, v.Probe.Options, config)
		if err != nil {
			err = fmt.Errorf("[%v] %w", v.Name, err)
			return
		}
		s.AddTask(model.NewTask(v.Name, v.CGroup, p))
	}
	return
}

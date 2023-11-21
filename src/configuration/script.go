package configuration

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"encoding/json"
	"fmt"
	"github.com/creasty/defaults"
	"sigs.k8s.io/yaml" // uses it instead gopkg.in/yaml.v3 as it also supports json attributes in struct
	"time"
)

type Script struct {
	Daemon  bool
	Timeout time.Duration `default:"60s"`
	Script  []Record
}

type Record struct {
	Name   string
	Probe  Probe
	CGroup string
	//DependsOn      string
}

type Probe struct {
	Name             string
	RawConfiguration *json.RawMessage `json:"configuration"`
	Configuration    any              `json:"-"`
	Options          model.ProbeOptions
	Expect           bool
}

func ymlConfiguration(data []byte) (s model.Script, err error) {
	s = model.NewScript()
	parsed := Script{}
	if err = defaults.Set(&parsed); err != nil {
		return
	}
	if err = yaml.Unmarshal(data, &parsed); err != nil {
		return
	}

	//s.Tasks = make([]*model.Task, len(parsed.Script))
	var p model.Prober
	for _, v := range parsed.Script {
		var config any
		// get the probe configuration struct
		config, err = probeFactory.NewProbeConfiguration(v.Probe.Name)
		if err != nil {
			err = fmt.Errorf("[%v] %w", v.Name, err)
			return
		}
		// try to fill config from RawConfiguration data
		if v.Probe.RawConfiguration != nil {
			e := json.Unmarshal(*v.Probe.RawConfiguration, config)
			// if unmarshal error, set config to raw data in order the Probe try to parse config by itself
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

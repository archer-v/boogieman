package probeFactory

import (
	"github.com/creasty/defaults"
	"liberator-check/src/model"
)

type BaseConstructor struct {
	Name string
}

func (c BaseConstructor) ProbeName() string {
	return c.Name
}

func (c BaseConstructor) SetConfigDefaults(conf any) any {
	_ = defaults.Set(conf)
	return conf
}

type Constructor interface {
	NewProbe(options model.ProbeOptions, configuration any) (model.Prober, error)
	NewProbeConfiguration() any
	ProbeName() string
}

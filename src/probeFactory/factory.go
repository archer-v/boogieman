package probeFactory

import (
	"boogieman/src/model"
	"errors"
	"fmt"
)

var probes map[string]Constructor

func NewProbe(name string, options model.ProbeOptions, configuration any) (p model.Prober, err error) {
	c, err := probeConstructorByName(name)
	if err != nil {
		return
	}
	p, err = c.NewProbe(options, configuration)
	if err != nil {
		err = fmt.Errorf("can't create probe '%v': %v", name, err)
	}
	return
}

// NewProbeConfiguration returns a new instance of probe configuration struct
func NewProbeConfiguration(name string) (conf any, err error) {
	c, err := probeConstructorByName(name)
	if err != nil {
		return
	}
	return c.NewProbeConfiguration(), nil
}

func RegisterProbe(c Constructor) {
	if probes == nil {
		probes = map[string]Constructor{}
	}
	probes[c.ProbeName()] = c
}

func probeConstructorByName(name string) (Constructor, error) {
	c, ok := probes[name]
	if !ok {
		return nil, errors.New("Unknown probe with name '" + name + "'")
	}
	return c, nil
}

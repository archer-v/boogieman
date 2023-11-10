package probeFactory

import (
	"errors"
	"fmt"
	"liberator-check/src/model"
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

/*
func ConfigurationFromString(name string, configuration string) (any, error) {
	c, ok := probes[name]
	if !ok {
		return nil, errors.New("Unknown probeFactory with name '" + name + "'")
	}
	c.ConfigurationFromString()
}

*/

// ProbeConfiguration returns a new instance of probe configuration struct
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

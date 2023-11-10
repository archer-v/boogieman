package ping

import (
	"liberator-check/src/model"
	"liberator-check/src/probeFactory"
	"regexp"
)

type constructor struct {
	probeFactory.BaseConstructor
}

func (c constructor) NewProbe(options model.ProbeOptions, configuration any) (p model.Prober, err error) {

	var config Config
	if config, err = c.configuration(configuration); err != nil {
		return
	}

	if len(config.Hosts) == 0 || config.Hosts[0] == "" {
		return nil, model.ErrorConfig
	}

	return New(options, config), nil
}

func (c constructor) NewProbeConfiguration() any {
	return &Config{}
}

// configuration casts configuration of any type to Config struct
func (c constructor) configuration(conf any) (configuration Config, err error) {

	if conf == nil {
		err = model.ErrorConfig
		return
	}

	if c, ok := conf.(*Config); ok {
		return *c, nil
	}

	if str, ok := conf.(string); ok {
		hosts := regexp.MustCompile("\\s*,\\s*").Split(str, -1)
		return Config{hosts}, nil
	}

	err = model.ErrorConfig
	return
}

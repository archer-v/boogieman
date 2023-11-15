package traceroute

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
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

	if config.Host == "" {
		return nil, model.ErrorConfig
	}
	if len(config.ExpectedHop) == 0 || config.ExpectedHop[0] == "" {
		return nil, model.ErrorConfig
	}

	return New(options, config), nil
}

func (c constructor) NewProbeConfiguration() any {
	return c.SetConfigDefaults(&Config{})
}

// configuration casts configuration of any type to Config struct
func (c constructor) configuration(conf any) (config Config, err error) {

	if conf == nil {
		err = model.ErrorConfig
		return
	}

	if cc, ok := conf.(*Config); ok {
		config = *cc
	} else if cc, ok := conf.(Config); ok {
		config = cc
	} else if str, ok := conf.(string); ok {
		s := regexp.MustCompile("\\s*,\\s*").Split(str, -1)
		if len(s) < 2 {
			err = model.ErrorConfig
			return
		}
		config = *(c.NewProbeConfiguration().(*Config))
		config.Host = s[0]
		config.ExpectedHop = s[1:]
	} else {
		err = model.ErrorConfig
		return
	}
	/*
		for _, r := range config.ExpectedHop {
			//s := regexp.MustCompile("\\s*,\\s*").Split(str, -1)
		}
	*/
	return
}

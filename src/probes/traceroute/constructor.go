package traceroute

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"regexp"
)

type constructor struct {
	probefactory.BaseConstructor
}

func (c constructor) NewProbe(options model.ProbeOptions, configuration any) (p model.Prober, err error) {
	var config Config
	if config, err = c.configuration(configuration); err != nil {
		return
	}

	if config.Host == "" {
		return nil, model.ErrorConfig
	}
	if len(config.ExpectedHops) == 0 || config.ExpectedHops[0] == "" {
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
		config.ExpectedHops = s[1:]
	} else {
		err = model.ErrorConfig
		return
	}
	return
}

package web

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
	if len(config.Urls) == 0 || config.Urls[0] == "" {
		return nil, model.ErrorConfig
	}
	return New(options, config), nil
}

func (c constructor) NewProbeConfiguration() any {
	return c.SetConfigDefaults(&Config{})
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
		urls := regexp.MustCompile("\\s*,\\s*").Split(str, -1)
		newConfig := c.NewProbeConfiguration().(*Config)
		newConfig.Urls = urls
		return *newConfig, nil
	}

	err = model.ErrorConfig
	return
}

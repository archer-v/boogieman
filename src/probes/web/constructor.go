package web

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"fmt"
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

	switch v := conf.(type) {
	case *Config:
		configuration = *v
	case Config:
		configuration = v
	case string:
		urls := regexp.MustCompile("\\s*,\\s*").Split(v, -1)
		newConfig := c.NewProbeConfiguration().(*Config)
		newConfig.Urls = urls
		configuration = *newConfig
	default:
		err = model.ErrorConfig
		return
	}

	err = configuration.compileBodyRegex()
	return
}

func (c *Config) compileBodyRegex() error {
	if c.BodyRegex == "" {
		if c.BodyRegexCaptureGroup > 0 {
			return fmt.Errorf("bodyRegexCaptureGroup requires bodyRegex")
		}
		return nil
	}
	if c.BodyRegexCaptureGroup < 0 {
		return fmt.Errorf("bodyRegexCaptureGroup should be greater than or equal to 0")
	}
	if c.BodyRegexInvert && c.BodyRegexCaptureGroup > 0 {
		return fmt.Errorf("bodyRegexCaptureGroup cannot be used with bodyRegexInvert")
	}

	r, err := regexp.Compile(c.BodyRegex)
	if err != nil {
		return fmt.Errorf("wrong bodyRegex: %w", err)
	}
	if c.BodyRegexCaptureGroup > r.NumSubexp() {
		return fmt.Errorf(
			"bodyRegexCaptureGroup %d is out of range, regex has %d capture groups",
			c.BodyRegexCaptureGroup, r.NumSubexp(),
		)
	}
	c.bodyRegexp = r
	return nil
}

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

	err = configuration.compileRegex()
	return
}

func (c *Config) compileRegex() error {
	if c.Regex == "" {
		if c.RegexCaptureGroup > 0 {
			return fmt.Errorf("regexCaptureGroup requires regex")
		}
		if c.CaptureRegex != "" {
			return fmt.Errorf("captureRegex requires regex")
		}
		return nil
	}
	if c.RegexCaptureGroup < 0 {
		return fmt.Errorf("regexCaptureGroup should be greater than or equal to 0")
	}
	if c.RegexInvert && c.RegexCaptureGroup > 0 {
		return fmt.Errorf("regexCaptureGroup cannot be used with regexInvert")
	}
	if c.CaptureRegex != "" && c.RegexCaptureGroup <= 0 {
		return fmt.Errorf("captureRegex requires regexCaptureGroup")
	}

	r, err := regexp.Compile(c.Regex)
	if err != nil {
		return fmt.Errorf("wrong regex: %w", err)
	}
	if c.RegexCaptureGroup > r.NumSubexp() {
		return fmt.Errorf(
			"regexCaptureGroup %d is out of range, regex has %d capture groups",
			c.RegexCaptureGroup, r.NumSubexp(),
		)
	}
	c.regexp = r
	if c.CaptureRegex != "" {
		cr, err := regexp.Compile(c.CaptureRegex)
		if err != nil {
			return fmt.Errorf("wrong captureRegex: %w", err)
		}
		c.captureRegexp = cr
	}
	return nil
}

package openvpn

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"fmt"
	"os"
)

type constructor struct {
	probefactory.BaseConstructor
}

func (c constructor) NewProbe(options model.ProbeOptions, configuration any) (p model.Prober, err error) {

	var config Config
	if config, err = c.configuration(configuration); err != nil {
		return
	}

	return New(options, config), nil
}

func (c constructor) NewProbeConfiguration() any {
	return c.SetConfigDefaults(&Config{})
}

// configuration casts configuration of any type to Config struct
func (c constructor) configuration(data any) (config Config, err error) {

	if data == nil {
		err = model.ErrorConfig
		return
	}

	cc, ok := data.(*Config)

	if ok {
		config = *cc
	} else if str, ok := data.(string); ok {
		cc = c.NewProbeConfiguration().(*Config)
		config = *cc
		config.ConfigFile = str
	}

	if config.ConfigFile == "" && config.ConfigData == "" {
		err = model.ErrorConfig
		return
	}

	if cc.ConfigFile != "" {
		config.ConfigData, err = c.readConfig(cc.ConfigFile)
		if err != nil {
			err = fmt.Errorf("can't load openvpn configuration from file %v: %w", cc.ConfigFile, err)
		}
	}

	return
}

func (c constructor) readConfig(path string) (data string, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("can't read file %v: %v", path, err)
		return
	}
	return string(b), nil
}

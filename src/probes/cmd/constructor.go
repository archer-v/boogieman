package cmd

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
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
func (c constructor) configuration(conf any) (config Config, err error) {
	if conf == nil {
		err = model.ErrorConfig
		return
	}

	if str, ok := conf.(string); ok {
		config = Config{}
		if err = config.initWithString(str); err != nil {
			return
		}
	} else if c, ok := conf.(*Config); ok {
		config = *c
	} else if c, ok := conf.(Config); ok {
		config = c
	} else {
		err = model.ErrorConfig
		return
	}
	// if no Args, config.Cmd can contain 'shell-like' command string, parse it
	if len(config.Args) == 0 {
		if err = config.initWithString(config.Cmd); err != nil {
			return
		}
	}

	path, err := exec.LookPath(config.Cmd)
	if err != nil {
		err = fmt.Errorf("can't lookup a cmd %v: %w", config.Cmd, err)
	}
	if i, e := os.Stat(path); e == nil {
		if i.IsDir() {
			err = fmt.Errorf("cmd %v is a directory", config.Cmd)
		} else if unix.Access(path, unix.X_OK) != nil {
			err = fmt.Errorf("cmd %v cannot be executed by this user", config.Cmd)
		}
	} else if errors.Is(e, os.ErrNotExist) {
		err = fmt.Errorf("cmd %v doesn't exists", config.Cmd)
	} else {
		err = fmt.Errorf("cmd %v couldn't be run: %w", config.Cmd, e)
	}

	return
}

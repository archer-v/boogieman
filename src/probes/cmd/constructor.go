package cmd

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"errors"
	"fmt"
	"github.com/kgadams/go-shellquote"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
)

type constructor struct {
	probeFactory.BaseConstructor
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
		args, e := shellquote.Split(str)
		if len(args) == 0 || args[0] == "" {
			err = model.ErrorConfig
			return
		}
		if e != nil {
			err = fmt.Errorf("can't parse cmd: %w", err)
			return
		}
		config = Config{Cmd: args[0], Args: args[1:]}
	} else if c, ok := conf.(*Config); ok {
		config = *c
	} else if c, ok := conf.(Config); ok {
		config = c
	} else {
		err = model.ErrorConfig
		return
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

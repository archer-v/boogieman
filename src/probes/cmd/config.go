package cmd

import (
	"boogieman/src/model"
	"fmt"
	"github.com/kgadams/go-shellquote"
)

type Config struct {
	Cmd      string // path to a cmd binary file
	Args     []string
	ExitCode int
	LogDump  bool
}

func (c *Config) initWithString(str string) (err error) {
	args, e := shellquote.Split(str)
	if len(args) == 0 || args[0] == "" {
		err = model.ErrorConfig
		return
	}
	if e != nil {
		err = fmt.Errorf("can't parse cmd: %w", err)
		return
	}
	c.Cmd = args[0]
	c.Args = args[1:]
	return
}

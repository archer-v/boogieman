package cmd

import (
	"boogieman/src/model"
	"fmt"
	"github.com/kgadams/go-shellquote"
	"regexp"
)

type Config struct {
	Cmd                     string // path to a cmd binary file
	Args                    []string
	ExitCode                int
	LogDump                 bool
	StdoutRegex             string `json:"stdoutRegex,omitempty"`
	StdoutRegexInvert       bool   `json:"stdoutRegexInvert,omitempty"`
	StdoutRegexCaptureGroup int    `json:"stdoutRegexCaptureGroup,omitempty"`
	stdoutRegexp            *regexp.Regexp
}

type ResultData struct {
	ExitCode int    `json:"exitCode"`
	Capture  string `json:"capture,omitempty"`
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

func (c *Config) compileStdoutRegex() error {
	if c.StdoutRegex == "" {
		if c.StdoutRegexCaptureGroup > 0 {
			return fmt.Errorf("stdoutRegexCaptureGroup requires stdoutRegex")
		}
		return nil
	}
	if c.StdoutRegexCaptureGroup < 0 {
		return fmt.Errorf("stdoutRegexCaptureGroup should be greater than or equal to 0")
	}
	if c.StdoutRegexInvert && c.StdoutRegexCaptureGroup > 0 {
		return fmt.Errorf("stdoutRegexCaptureGroup cannot be used with stdoutRegexInvert")
	}

	r, err := regexp.Compile(c.StdoutRegex)
	if err != nil {
		return fmt.Errorf("wrong stdoutRegex: %w", err)
	}
	if c.StdoutRegexCaptureGroup > r.NumSubexp() {
		return fmt.Errorf(
			"stdoutRegexCaptureGroup %d is out of range, regex has %d capture groups",
			c.StdoutRegexCaptureGroup, r.NumSubexp(),
		)
	}
	c.stdoutRegexp = r
	return nil
}

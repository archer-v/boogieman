package cmd

import (
	"boogieman/src/model"
	"fmt"
	"github.com/kgadams/go-shellquote"
	"regexp"
)

type Config struct {
	Cmd               string // path to a cmd binary file
	Args              []string
	ExitCode          int
	LogDump           bool
	Regex             string `json:"regex,omitempty"`
	RegexInvert       bool   `json:"regexInvert,omitempty"`
	RegexRequired     *bool  `json:"regexRequired,omitempty"`
	RegexCaptureGroup int    `json:"regexCaptureGroup,omitempty"`
	regexp            *regexp.Regexp
}

type ResultData struct {
	ExitCode int     `json:"exitCode"`
	Regex    *bool   `json:"regex,omitempty"`
	Capture  *string `json:"capture,omitempty"`
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

func (c *Config) compileRegex() error {
	if c.Regex == "" {
		if c.RegexCaptureGroup > 0 {
			return fmt.Errorf("regexCaptureGroup requires regex")
		}
		return nil
	}
	if c.RegexCaptureGroup < 0 {
		return fmt.Errorf("regexCaptureGroup should be greater than or equal to 0")
	}
	if c.RegexInvert && c.RegexCaptureGroup > 0 {
		return fmt.Errorf("regexCaptureGroup cannot be used with regexInvert")
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
	return nil
}

func (c Config) regexRequired() bool {
	return c.RegexRequired == nil || *c.RegexRequired
}

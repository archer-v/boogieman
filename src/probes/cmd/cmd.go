package cmd

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"context"
	"errors"
	"fmt"
	"github.com/go-cmd/cmd"
	"log"
	"time"
)

type Probe struct {
	model.ProbeHandler
	Config `json:"config"`
	cmd    *cmd.Cmd
}

type Config struct {
	Cmd      string // path to openvpn client configuration file
	Args     []string
	ExitCode int
	LogDump  bool
}

var name = "cmd"
var ErrTimeout = errors.New("timeout")

func init() {
	probeFactory.RegisterProbe(constructor{probeFactory.BaseConstructor{Name: name}})
}

func New(options model.ProbeOptions, config Config) *Probe {
	p := Probe{}
	p.ProbeOptions = options
	p.Name = name
	p.Config = config
	p.CanStayBackground = true
	p.SetRunner(p.Runner).SetFinisher(p.Finisher)
	return &p
}

func (c *Probe) Runner(ctx context.Context) (succ bool, resultObject any) {
	var (
		err      error
		finished cmd.Status
	)

	defer func() {
		if err != nil {
			c.Log("[%v] %v, %vms", c.Cmd, err, c.Duration().Milliseconds())
			c.SetError(err)
		} else {
			c.Log("[%v] OK, %vms", c.Cmd, c.Duration().Milliseconds())
		}
	}()

	if c.cmd != nil {
		err = fmt.Errorf("another cmd is still running")
		return
	}

	c.cmd = cmd.NewCmdOptions(cmd.Options{Buffered: true, Streaming: true}, c.Cmd, c.Args...)

	status := c.cmd.Start()

	timer := time.After(c.Timeout)
	var timeout error
	for finished.Runtime == 0 && timeout == nil {
		select {
		case finished = <-status:
			break
		case <-timer:
			timeout = ErrTimeout
			break
		case line := <-c.cmd.Stdout:
			if c.LogDump {
				log.Print(line)
			}
		case line := <-c.cmd.Stderr:
			if c.LogDump {
				log.Print(line)
			}
		}
	}

	if c.StayBackground {
		// process waiting timeout is happened, process is still alive
		if timeout != nil {
			succ = true
			// continue to read stdout/stderr of running process until channel closing
			go func(cmd *cmd.Cmd) {
				var line string
				ok := true
				for ok {
					select {
					case line, ok = <-c.cmd.Stdout:
					case line, ok = <-c.cmd.Stderr:
					}
					if ok && c.LogDump && line != "" {
						log.Print(line)
					}
				}
				// channel is closed
				c.Finish(ctx)
			}(c.cmd)
			return
		}
		succ = false
		return
	}

	if timeout != nil {
		err = timeout
		c.Finish(ctx)
		return false, nil
	}

	if finished.Exit != c.ExitCode {
		err = fmt.Errorf("wrong exit code %v", finished.Exit)
	}
	succ = finished.Exit == c.ExitCode == c.Expect
	return
}

func (c *Probe) Finisher(ctx context.Context) {
	if c.cmd != nil {
		err := c.cmd.Stop()
		if err != nil {
			c.Log("unexpected error on stopping cmd process")
		}
		c.cmd = nil
	}
}

func (c *Probe) IsAlive() bool {
	return c.cmd != nil
}

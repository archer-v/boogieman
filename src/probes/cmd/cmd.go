package cmd

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
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

var name = "cmd"
var ErrTimeout = errors.New("timeout")
var ErrUnexpectedExit = errors.New("cmd exited unexpectedly")

func init() {
	probefactory.RegisterProbe(constructor{probefactory.BaseConstructor{Name: name}})
}

func New(options model.ProbeOptions, config Config) *Probe {
	p := Probe{}
	p.ProbeOptions = options
	p.Name = name
	p.Config = config
	p.ProbeHandler.Config = config
	p.CanStayBackground = true
	p.SetRunner(p.Runner).SetFinisher(p.Finisher)
	return &p
}

//nolint:funlen
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
	var interrupted error
	for finished.Runtime == 0 && interrupted == nil {
		select {
		// cmd has been finished
		case finished = <-status:
			break
		// context cancel is happened
		case <-ctx.Done():
			interrupted = ctx.Err()
		// interrupted is happened
		case <-timer:
			interrupted = ErrTimeout
			break
		// new lines in stdout / stderr
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
	// if the process should stay background
	if c.StayBackground {
		// a waiting timeout isn't happened, it means the process exited unexpectedly
		if interrupted == nil {
			c.Log("Command should stay background but exited unexpectedly")
			succ = false
			err = ErrUnexpectedExit
			return
		}

		if interrupted != ErrTimeout {
			err = interrupted
			succ = false
			return
		}

		// process waiting timeout is happened, process is still alive
		succ = true
		// continue to read stdout/stderr of running process until channel closing
		go func(cmd *cmd.Cmd) {
			var line string
			ok := true
			for ok {
				select {
				case line, ok = <-cmd.Stdout:
				case line, ok = <-cmd.Stderr:
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

	if interrupted != nil {
		err = interrupted
		c.Finish(ctx)
		return false, nil
	}

	if finished.Exit != c.ExitCode {
		err = fmt.Errorf("wrong exit code %v", finished.Exit)
	}
	succ = finished.Exit == c.ExitCode == c.Expect
	resultObject = finished.Exit
	return
}

func (c *Probe) Finisher(context.Context) {
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

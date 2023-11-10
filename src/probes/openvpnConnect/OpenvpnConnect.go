package openvpnConnect

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-cmd/cmd"
	"liberator-check/src/model"
	"liberator-check/src/probeFactory"
	"liberator-check/src/util"
	"log"
	"strings"
	"syscall"
	"time"
)

var OpenvpnBinaryPath = "openvpn"

type Probe struct {
	model.ProbeHandler
	Config
	cmd *cmd.Cmd
}

type Config struct {
	ConfigFile string // path to openvpn client configuration file
	LogDump    bool
	ConfigData string // openvpn client configuration
}

var name = "openvpnConnect"

var (
	ErrTimeout     = errors.New("timeout")
	ErrConnRefused = errors.New("connection refused")
)

func init() {
	probeFactory.RegisterProbe(constructor{probeFactory.BaseConstructor{Name: name}})
}

func New(options model.ProbeOptions, config Config) *Probe {
	p := Probe{}
	p.ProbeOptions = options
	p.Name = name
	p.Config = config
	p.SetRunner(p.Runner)
	return &p
}

func (c *Probe) Runner(ctx context.Context) (succ bool) {
	if c.cmd != nil {
		c.SetError(fmt.Errorf("another openvpn is still running by this probe"))
		return
	}

	var configFileName string
	var err error
	if c.ConfigFile != "" {
		configFileName = c.ConfigFile
	} else {
		configFileName, err = util.StringToFile(c.ConfigData)
		if err != nil {
			err = fmt.Errorf("can't create temporary file with openvpn config: %v", err)
			c.SetError(err)
			return
		}
		defer func() {
			err = syscall.Unlink(configFileName)
			if err != nil {
				c.Log("can't remove tmpFile %v: %v", configFileName, err)
			}
		}()
	}

	var host string
	for _, l := range strings.Split(c.ConfigData, "\n") {
		if strings.Contains(l, "remote") {
			args := strings.Split(l, " ")
			if len(args) == 3 {
				host = args[1] + ":" + args[2]
			}
		}
	}

	if host == "" {
		err = errors.New("can't get remote addr from openvpn configuration")
		c.SetError(err)
		return
	}

	c.SetLogContext(host)
	c.cmd, err = openvpnStart(ctx, configFileName, c.Timeout, c.LogDump)

	succ = err == nil

	c.SetError(err).Finished(succ)

	if err != nil {
		c.Log("[%v] %v, %vms", host, err, c.Duration().Milliseconds())
	} else {
		c.Log("[%v] OK, %vms", host, c.Duration().Milliseconds())
	}

	if !c.StayAlive {
		c.Finish()
	}
	return succ
}

func (c *Probe) Finish() {
	if c.cmd != nil {
		err := c.cmd.Stop()
		if err != nil {
			c.Log("unexpected error on stopping openvpn process")
		}
		c.cmd = nil
	}
}

// openvpnStart starts openvpn process and wait until connection established or error | timeout happened,
// returns cmd.Cmd describing running openvpn instance or error
func openvpnStart(ctx context.Context, configPath string, initTimeout time.Duration, logout bool) (runner *cmd.Cmd, err error) {
	runner = cmd.NewCmdOptions(cmd.Options{Buffered: true, Streaming: true}, OpenvpnBinaryPath, "--config", configPath)

	status := runner.Start()

	var finished cmd.Status
	timer := time.After(initTimeout)
	var succ bool
	for finished.Cmd == "" && err == nil && !succ {
		select {
		case finished = <-status:
			// finished (possible with error)
			break
		case <-timer:
			err = ErrTimeout
			if e := runner.Stop(); e != nil {
				log.Printf("unexpected error on stopping openvpn process: %v", e)
			}
			// exit with timeout
		case line := <-runner.Stdout:
			if logout {
				log.Printf(line)
			}
			if strings.Contains(line, "Initialization Sequence Completed") {
				succ = true
				break
			}
			if strings.Contains(line, "Connection refused") {
				err = ErrConnRefused
				break
			}
		case line := <-runner.Stderr:
			if logout {
				log.Printf(line)
			}
		}
	}

	if succ || err != nil {
		return
	}

	// exit with error on startup
	if finished.Cmd != "" {
		for _, l := range finished.Stdout {
			if strings.Contains(l, "ERROR") || strings.Contains(l, "error") {
				err = errors.New(l)
				break
			}
		}
		if err == nil {
			err = fmt.Errorf("error with code %v on startup", finished.Exit)
			// todo possible double logdump
			if logout {
				for _, l := range finished.Stdout {
					log.Printf(l)
				}
			}
		}
	}
	return
}

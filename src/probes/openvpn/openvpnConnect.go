package openvpn

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"boogieman/src/util"
	"context"
	"errors"
	"fmt"
	"github.com/go-cmd/cmd"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

var BinaryPath = "openvpn"

type Probe struct {
	model.ProbeHandler
	Config `json:"config"`
	cmd    *cmd.Cmd
}

type Config struct {
	ConfigFile string // path to openvpn client configuration file
	LogDump    bool
	ConfigData string // openvpn client configuration
}

var name = "openvpn"

var (
	ErrTimeout     = errors.New("timeout")
	ErrConnRefused = errors.New("connection refused")
)

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
		host           string
		configFileName string
		err            error
	)

	defer func() {
		if err != nil {
			c.Log("[%v] %v, %vms", host, err, c.Duration().Milliseconds())
			c.SetError(err)
		} else {
			c.Log("[%v] OK, %vms", host, c.Duration().Milliseconds())
		}
	}()

	if c.cmd != nil {
		err = fmt.Errorf("another openvpn is still running by this probe")
		return
	}

	if c.ConfigFile != "" {
		configFileName = c.ConfigFile
		configData, e := os.ReadFile(configFileName)
		if e != nil {
			err = fmt.Errorf("can't read openvpn config: %w", e)
			return
		}
		host = hostFromConfigData(string(configData))
	} else {
		configFileName, err = util.StringToFile(c.ConfigData)
		if err != nil {
			err = fmt.Errorf("can't create temporary file with openvpn config: %w", err)
			return
		}
		defer func() {
			e := syscall.Unlink(configFileName)
			if e != nil {
				c.Log("can't remove tmpFile %v: %v", configFileName, e)
			}
		}()
		host = hostFromConfigData(c.ConfigData)
	}

	if host == "" {
		err = errors.New("can't get remote addr from openvpn configuration")
		return
	}

	c.cmd, err = openvpnStart(ctx, configFileName, c.Timeout, c.LogDump)

	succ = err == nil

	if !c.StayBackground {
		c.Finish(ctx)
	} else if succ {
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
	}

	return
}

func (c *Probe) Finisher(context.Context) {
	if c.cmd != nil {
		err := c.cmd.Stop()
		if err != nil {
			c.Log("unexpected error on stopping openvpn process")
		}
		c.cmd = nil
	}
}

func (c *Probe) IsAlive() bool {
	return c.cmd != nil
}

// openvpnStart starts openvpn process and wait until connection established or error | timeout happened,
// returns cmd.Cmd describing running openvpn instance or error
func openvpnStart(ctx context.Context, configPath string, initTimeout time.Duration, logout bool) (cmdRunner *cmd.Cmd, err error) {
	cmdRunner = cmd.NewCmdOptions(cmd.Options{Buffered: true, Streaming: true}, BinaryPath, "--config", configPath)

	status := cmdRunner.Start()

	var finished cmd.Status
	timer := time.After(initTimeout)
	succ := false
	for finished.Runtime == 0 && err == nil && !succ {
		select {
		case finished = <-status:
			break
		case <-ctx.Done():
			err = ctx.Err()
		case <-timer:
			err = ErrTimeout
			break
		case line := <-cmdRunner.Stdout:
			if logout {
				log.Print(line)
			}
			if strings.Contains(line, "Initialization Sequence Completed") {
				succ = true
				break
			}
			if strings.Contains(line, "Connection refused") { // is happened only on openvpn client startup
				err = ErrConnRefused
				break
			}
		case line := <-cmdRunner.Stderr:
			if logout {
				log.Print(line)
			}
		}
	}

	// stop process on errors
	if err != nil {
		if e := cmdRunner.Stop(); e != nil {
			log.Printf("unexpected error on stopping openvpn process: %v", e)
		}
	}

	// parse stdout for inapp errors if a process has stopped
	if err == nil && finished.Runtime != 0 {
		for _, l := range finished.Stdout {
			if strings.Contains(l, "ERROR") || strings.Contains(l, "error") {
				err = errors.New(l)
				break
			}
		}
		if err == nil {
			err = fmt.Errorf("error with code %v on startup", finished.Exit)
		}
	}

	return
}

func hostFromConfigData(configData string) (host string) {
	for _, l := range strings.Split(configData, "\n") {
		if strings.Contains(l, "remote") {
			args := strings.Split(l, " ")
			if len(args) == 3 {
				host = args[1] + ":" + args[2]
			}
		}
	}
	return
}

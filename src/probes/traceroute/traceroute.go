package traceroute

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"context"
	"errors"
	"github.com/archer-v/gotraceroute"
	"log"
	"strings"
	"time"
)

type Probe struct {
	model.ProbeHandler
	Config
}

type Config struct {
	Host         string
	Port         int
	MaxHops      int
	HopTimeout   time.Duration `default:"200ms"`
	ExpectedHop  []string
	Retries      int `default:"2"`
	LogDump      bool
	traceOptions gotraceroute.Options
}

var name = "traceroute"

var ErrTimeout = errors.New("timeout")

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

	var (
		err error
	)
	tOptions := gotraceroute.Options{
		Port:    c.Port,
		Timeout: c.HopTimeout,
		MaxHops: c.MaxHops,
		Retries: c.Retries,
	}

	defer func() {
		if err != nil {
			c.Log("[%v] %v, %vms", c.Host, err, c.Duration().Milliseconds())
			c.SetError(err)
		} else {
			c.Log("[%v] OK, %vms", c.Host, c.Duration().Milliseconds())
		}
	}()

	timer := time.After(c.Timeout)
	ctxWithCancel, cancel := context.WithCancel(ctx)

	defer cancel()

	hopChan, err := gotraceroute.Run(ctxWithCancel, c.Host, tOptions)
	if err != nil {
		return
	}

	var (
		hop gotraceroute.Hop
		ok  = true
	)
	for !succ && ok && err == nil {
		select {
		case <-timer:
			err = ErrTimeout
			break
		case hop, ok = <-hopChan:
			if !ok {
				break
			}
			if c.LogDump {
				log.Printf(hop.StringHuman())
			}
			for _, exp := range c.ExpectedHop {
				if strings.Contains(hop.Node.String(), exp) {
					succ = true
					break
				}
			}
		}
	}

	succ = succ && err == nil
	return
}

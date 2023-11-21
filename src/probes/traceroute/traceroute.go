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
	Host          string
	Port          int
	MaxHops       int
	HopTimeout    time.Duration `default:"200ms"`
	ExpectedHops  []string
	ExpectedMatch string `default:"any"` // any | all | none
	Retries       int    `default:"2"`
	LogDump       bool
	traceOptions  gotraceroute.Options
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
		hop             gotraceroute.Hop
		traceInProgress = true
		finished        = false
		matches         = 0
	)
	if c.ExpectedMatch == "none" {
		succ = true
	}
	for !finished && traceInProgress && err == nil {
		select {
		case <-timer:
			err = ErrTimeout
			break
		case hop, traceInProgress = <-hopChan:
			if !traceInProgress {
				break
			}
			if c.LogDump {
				log.Printf(hop.StringHuman())
			}
			for _, exp := range c.ExpectedHops {
				if strings.Contains(hop.Node.String(), exp) {
					if c.ExpectedMatch == "any" {
						succ = true
						finished = true
					} else if c.ExpectedMatch == "none" {
						succ = false
						finished = true
					} else if c.ExpectedMatch == "all" {
						matches++
						if matches == len(c.ExpectedHops) {
							succ = true
							finished = true
						}
					}
					break
				}
			}
		}
	}

	succ = succ && err == nil
	return
}

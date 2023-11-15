package traceroute

// github.com/aeden/traceroute module is pretty buggy, so it should be replaced to something else
// know bugs:
//    - doesn't work on windows
//    - catch someone else's icmp replies
//    - no ivp6

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"context"
	"errors"
	"fmt"
	"github.com/aeden/traceroute"
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
	traceOptions traceroute.TracerouteOptions
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
		err      error
		tOptions traceroute.TracerouteOptions
		hop      traceroute.TracerouteHop
	)
	if c.Port > 0 {
		tOptions.SetPort(c.Port)
	}
	if c.HopTimeout != 0 {
		tOptions.SetTimeoutMs(int(c.HopTimeout.Milliseconds()))
	}
	if c.MaxHops > 0 {
		tOptions.SetMaxHops(c.MaxHops)
	}
	if c.Retries > 0 {
		tOptions.SetRetries(c.Retries)
	}

	defer func() {
		if err != nil {
			c.Log("[%v] %v, %vms", c.Host, err, c.Duration().Milliseconds())
			c.SetError(err)
		} else {
			c.Log("[%v] OK, %vms", c.Host, c.Duration().Milliseconds())
		}
	}()

	tHop := make(chan traceroute.TracerouteHop, tOptions.MaxHops())
	errChan := make(chan error)
	go func(host string, options traceroute.TracerouteOptions, hops chan traceroute.TracerouteHop, err chan error) {
		out, e := traceroute.Traceroute(host, &options, hops)
		if e != nil {
			err <- fmt.Errorf("failed: %w", e)
		} else if len(out.Hops) == 0 {
			err <- fmt.Errorf("failed. expected at least one hop")
		}
	}(c.Host, tOptions, tHop, errChan)

	timer := time.After(c.Timeout)
	ok := true
	for ok && err == nil && !succ {
		select {
		case hop, ok = <-tHop:
			//parse hop
			for _, exp := range c.ExpectedHop {
				if hop.Host != "" && strings.Contains(hop.Host, exp) {
					succ = true
					break
				} else if strings.Contains(hop.AddressString(), exp) {
					succ = true
					break
				}
			}
			if c.LogDump {
				log.Printf("%-3d %v (%v)  %v\n", hop.TTL, hop.HostOrAddressString(), hop.AddressString(), hop.ElapsedTime)
			}
			break
		case <-timer:
			err = ErrTimeout
			break
		case err = <-errChan:

		}
	}

	succ = succ && err == nil
	return
}

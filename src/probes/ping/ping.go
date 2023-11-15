package ping

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"context"
	"errors"
	"fmt"
	"github.com/prometheus-community/pro-bing"
	"strings"
	"sync"
	"time"
)

type Probe struct {
	model.ProbeHandler
	Config
}

type Config struct {
	Hosts []string
}

var name = "ping"

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
	var wg sync.WaitGroup
	done := 0
	for _, host := range c.Hosts {
		wg.Add(1)
		go func(s string) {
			t := time.Now()
			var dur time.Duration
			var err error
			defer func() {
				dur = time.Since(t)
				if err != nil {
					c.Log("[%v] %v, %vms", s, err, dur.Milliseconds())
					if errors.Is(ErrTimeout, err) && !c.Expect {
						done++
					}
				} else {
					c.SetTimeStat(s, dur)
					if c.Expect {
						done++
					}
					c.Log("[%v] OK, %vms", s, dur.Milliseconds())
				}
				wg.Done()
			}()

			p, err := probing.NewPinger(s)
			if err != nil {
				//c.Log(err.Error())
				//if strings.Contains(err.Error(), "no such host") {
				//	err = errors.New("no such host")
				//}
				return
			}
			p.Count = 3
			p.Timeout = c.Timeout
			p.Interval = 500 * time.Millisecond
			p.SetPrivileged(true)

			p.OnRecv = func(pkt *probing.Packet) {
				// stop on a first received packet
				p.Stop()
			}

			err = p.RunWithContext(ctx)

			if err != nil {
				if strings.Contains(err.Error(), "not permitted") {
					err = fmt.Errorf("error %v, see https://github.com/prometheus-community/pro-bing#supported-operating-systems", err)
				}
				return
			}

			stats := p.Statistics()
			if stats.PacketsRecv == 0 {
				err = ErrTimeout
			}
		}(host)
	}
	wg.Wait()
	succ = done == len(c.Hosts)
	return
}

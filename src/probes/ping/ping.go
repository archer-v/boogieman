package ping

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"context"
	"errors"
	"fmt"
	"github.com/creasty/defaults"
	"github.com/prometheus-community/pro-bing"
	"strings"
	"sync"
	"time"
)

type Probe struct {
	model.ProbeHandler
	Config `json:"config"`
}

type Config struct {
	Interval int `default:"500"`
	Hosts    []string
}

var name = "ping"

var ErrTimeout = errors.New("timeout")

func init() {
	probefactory.RegisterProbe(constructor{probefactory.BaseConstructor{Name: name}})
}

func New(options model.ProbeOptions, config Config) *Probe {
	p := Probe{}
	p.ProbeOptions = options
	p.Name = name
	_ = defaults.Set(&config)
	p.Config = config
	p.ProbeHandler.Config = config
	p.SetRunner(p.Runner)
	return &p
}

func (c *Probe) Runner(ctx context.Context) (succ bool, resultObject any) {
	var timings model.Timings
	var wg sync.WaitGroup
	var mutex sync.Mutex
	done := 0
	for _, host := range c.Hosts {
		wg.Add(1)
		go func(s string) {
			t := time.Now()
			var dur time.Duration
			var err error
			defer func() {
				dur = time.Since(t)
				if e := recover(); e != nil {
					err = fmt.Errorf("panic occurred: %v", e)
				}

				if err != nil {
					c.Log("[%v] %v, %vms", s, err, dur.Milliseconds())
					if errors.Is(ErrTimeout, err) && !c.Expect {
						mutex.Lock()
						done++
						mutex.Unlock()
					}
				} else {
					timings.Set(s, dur)
					if c.Expect {
						mutex.Lock()
						done++
						mutex.Unlock()
					}
					c.Log("[%v] OK, %vms", s, dur.Milliseconds())
				}
				wg.Done()
			}()

			p, err := probing.NewPinger(s)
			if err != nil {
				return
			}
			p.Count = 3
			p.Timeout = c.Timeout
			p.Interval = time.Duration(c.Interval) * time.Millisecond
			p.SetPrivileged(true)

			p.OnRecv = func(pkt *probing.Packet) {
				// stop on a first received packet
				p.Stop()
			}

			err = p.RunWithContext(ctx)

			if err != nil {
				if strings.Contains(err.Error(), "not permitted") {
					// see https://github.com/prometheus-community/pro-bing#supported-operating-systems
					err = fmt.Errorf("error %w, root privileges is required or SET_CAP_RAW flag", err)
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
	resultObject = timings.TimingsMs()
	return
}

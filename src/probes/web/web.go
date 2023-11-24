package web

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Probe struct {
	model.ProbeHandler
	Config `json:"config"`
}

type Config struct {
	HttpStatus int `default:"200"`
	Urls       []string
}

var name = "web"
var ErrTimeout = errors.New("timeout")

func init() {
	probeFactory.RegisterProbe(constructor{probeFactory.BaseConstructor{Name: name}})
}

func New(options model.ProbeOptions, config Config) *Probe {
	p := Probe{}
	p.ProbeOptions = options
	p.ProbeHandler.Name = name
	p.Config = config
	p.SetRunner(p.Runner)
	return &p
}

func (c *Probe) Runner(ctx context.Context) (succ bool, resultObject any) {
	var timings model.Timings
	var wg sync.WaitGroup
	done := 0
	for _, s := range c.Urls {
		if _, e := url.Parse(s); e != nil {
			c.Log("wrong url %v", s)
			return false, nil
		}

		wg.Add(1)
		go func(s string) {
			t := time.Now()
			var dur time.Duration
			var err error
			defer func() {
				dur = time.Since(t)
				if err != nil {
					c.Log("[%v] %v, %vms", s, err, dur.Milliseconds())
					if !c.Expect {
						done++
					}
				} else {
					timings.Set(s, dur)
					if c.Expect {
						done++
					}
					c.Log("[%v] OK, %vms", s, dur.Milliseconds())
				}
				wg.Done()
			}()

			client := http.Client{Timeout: c.Timeout}
			r, err := client.Get(s)
			if err != nil {
				if strings.Contains(err.Error(), "context deadline exceeded") {
					err = ErrTimeout
				} else {
					err = fmt.Errorf("http error %v", err)
				}
				return
			}
			if r.StatusCode != c.HttpStatus {
				err = fmt.Errorf("wrong response %v", r.StatusCode)
				return
			}
		}(s)
	}
	wg.Wait()
	succ = done == len(c.Urls)
	resultObject = &timings
	return
}

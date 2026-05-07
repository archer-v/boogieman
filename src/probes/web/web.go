package web

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Probe struct {
	model.ProbeHandler
	Config `json:"config"`
}

type Config struct {
	HTTPStatus      int `default:"200"`
	Urls            []string
	FWMark          int    `json:"fwMark,omitempty"`
	BodyRegex       string `json:"bodyRegex,omitempty"`
	BodyRegexInvert bool   `json:"bodyRegexInvert,omitempty"`
	bodyRegexp      *regexp.Regexp
}

var name = "web"
var ErrTimeout = errors.New("timeout")
var DefaultHttpScheme = "https"

func init() {
	probefactory.RegisterProbe(constructor{probefactory.BaseConstructor{Name: name}})
}

func New(options model.ProbeOptions, config Config) *Probe {
	p := Probe{}
	p.ProbeOptions = options
	p.ProbeHandler.Name = name
	p.Config = config
	p.ProbeHandler.Config = config
	p.SetRunner(p.Runner)
	return &p
}

//nolint:funlen
func (c *Probe) Runner(ctx context.Context) (succ bool, resultObject any) {
	var timings model.Timings
	var wg sync.WaitGroup
	var mutex sync.Mutex
	done := 0
	for _, s := range c.Urls {
		if u, e := url.Parse(s); e != nil {
			c.Log("wrong url %v", s)
			return false, nil
		} else if u.Scheme == "" {
			s = DefaultHttpScheme + "://" + s
		}

		wg.Add(1)
		go func(s string) {
			t := time.Now()
			var (
				dur time.Duration
				err error
				r   *http.Response
			)
			defer func() {
				dur = time.Since(t)
				if err != nil {
					c.Log("[%v] %v, %vms", s, err, dur.Milliseconds())
					if !c.Expect {
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
				if r != nil {
					_ = r.Body.Close()
				}
				wg.Done()
			}()

			client := c.httpClient()
			r, err = client.Get(s)
			if err != nil {
				if strings.Contains(err.Error(), "context deadline exceeded") {
					err = ErrTimeout
				} else {
					err = fmt.Errorf("http error %w", err)
				}
				return
			}
			if r.StatusCode != c.HTTPStatus {
				err = fmt.Errorf("wrong response %v", r.StatusCode)
				return
			}
			err = c.checkBody(r)
		}(s)
	}
	wg.Wait()
	succ = done == len(c.Urls)
	resultObject = timings.TimingsMs()
	return
}

func (c *Probe) httpClient() http.Client {
	return newHTTPClient(c.Timeout, c.FWMark)
}

func (c *Probe) checkBody(r *http.Response) error {
	if c.bodyRegexp == nil {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("can't read response body: %w", err)
	}

	matched := c.bodyRegexp.Match(body)
	if matched == c.BodyRegexInvert {
		if c.BodyRegexInvert {
			return fmt.Errorf("body matches forbidden regex")
		}
		return fmt.Errorf("body doesn't match regex")
	}
	return nil
}

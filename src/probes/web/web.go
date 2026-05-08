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
	HTTPStatus        int
	Urls              []string
	FWMark            int    `json:"fwMark,omitempty"`
	Regex             string `json:"regex,omitempty"`
	RegexInvert       bool   `json:"regexInvert,omitempty"`
	RegexRequired     *bool  `json:"regexRequired,omitempty"`
	RegexCaptureGroup int    `json:"regexCaptureGroup,omitempty"`
	regexp            *regexp.Regexp
}

type ResultData struct {
	Timings    map[string]int    `json:"timings"`
	HTTPStatus map[string]int    `json:"httpStatus"`
	Regex      map[string]bool   `json:"regex,omitempty"`
	Captures   map[string]string `json:"captures,omitempty"`
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
	captures := make(map[string]string)
	httpStatus := make(map[string]int)
	regex := make(map[string]bool)
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
			timings.Set(s, time.Since(t))
			mutex.Lock()
			httpStatus[s] = r.StatusCode
			mutex.Unlock()

			matched, capture, bodyErr := c.checkBody(r)
			if c.regexp != nil {
				mutex.Lock()
				regex[s] = matched
				mutex.Unlock()
			}
			if c.RegexCaptureGroup > 0 {
				mutex.Lock()
				captures[s] = capture
				mutex.Unlock()
			}
			if c.HTTPStatus != 0 && r.StatusCode != c.HTTPStatus {
				err = fmt.Errorf("wrong response %v", r.StatusCode)
				return
			}
			err = bodyErr
		}(s)
	}
	wg.Wait()
	succ = done == len(c.Urls)

	rd := ResultData{
		Timings:    timings.TimingsMs(),
		HTTPStatus: httpStatus,
	}
	if c.regexp != nil {
		rd.Regex = regex
	}
	if c.RegexCaptureGroup > 0 {
		rd.Captures = captures
	}
	resultObject = rd

	return
}

func (c *Probe) httpClient() http.Client {
	return newHTTPClient(c.Timeout, c.FWMark)
}

func (c *Probe) checkBody(r *http.Response) (matched bool, capture string, err error) {
	if c.regexp == nil {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("can't read response body: %w", err)
		return
	}

	matches := c.regexp.FindSubmatch(body)
	matched = len(matches) > 0
	if matched && c.RegexCaptureGroup > 0 {
		capture = string(matches[c.RegexCaptureGroup])
	}
	if !c.regexRequired() {
		return
	}
	if matched == c.RegexInvert {
		if c.RegexInvert {
			err = fmt.Errorf("body matches forbidden regex")
			return
		}
		err = fmt.Errorf("body doesn't match regex")
		return
	}
	return
}

func (c *Probe) regexRequired() bool {
	return c.Config.regexRequired()
}

func (c Config) regexRequired() bool {
	return c.RegexRequired == nil || *c.RegexRequired
}

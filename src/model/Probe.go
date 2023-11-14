package model

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/creasty/defaults"
	"log"
	"strings"
	"sync"
	"time"
)

func init() {
	defaults.MustSet(&DefaultProbeOptions)
}

var ErrorConfig = errors.New("wrong configuration")

var DefaultProbeOptions ProbeOptions

type ProbeOptions struct {
	Timeout   time.Duration `json:"timeout,omitempty" default:"5000ms"`
	StayAlive bool          `json:"stayAlive,omitempty"` // probe process stays live after runner is exit, and should be finished by calling Finish method
	Expect    bool          `json:"expect,omitempty" default:"true"`
}

func (s *ProbeOptions) UnmarshalJSON(b []byte) (err error) {
	// init default options than parse json
	o := DefaultProbeOptions

	type tmp ProbeOptions
	t := tmp(o)

	err = json.Unmarshal(b, &t)
	if err != nil {
		return
	}
	t.Timeout = t.Timeout * time.Millisecond
	*s = ProbeOptions(t)

	return
}

type ProbeRunner func(ctx context.Context) (succ bool)
type ProbeFinisher func(ctx context.Context)

type ProbeHandler struct {
	ProbeOptions
	Runner
	Result
	Name              string // probe name
	CanStayBackground bool
	runner            ProbeRunner              // probe runner
	finisher          ProbeFinisher            // probe finisher, only for probe that stays alive in background
	timings           map[string]time.Duration // timings data
	logContext        string                   // log prefix string
	error             error                    // last startup error
	sync.Mutex
}

// Start starts the probing and returns a probing result
func (c *ProbeHandler) Start(ctx context.Context) (succ bool) {
	if c.runner == nil {
		c.Log("runner isn't defined")
		return
	}
	if err := c.EStatusRun(); err != nil {
		c.Log("wrong runner status: %v", err.Error())
	}
	c.Result.PrepareToStart()
	if ctx != nil && ctx.Value("id") != nil {
		c.SetLogContext(ctx.Value("id").(string))
	}
	succ = c.runner(ctx)

	c.Result.End(succ)

	// todo do not clear && messy, need refactor
	if !c.CanStayBackground || !succ || c.Error() != nil || !c.StayAlive {
		_ = c.EStatusFinish(succ)
	}

	if succ {
		c.Log("SUCCESS")
	} else {
		c.Log("FAIL")
	}
	c.SetLogContext("")

	return
}

// SetError sets probing error, should be called internally from the probe
func (c *ProbeHandler) SetError(err error) *ProbeHandler {
	c.error = err
	return c
}

// SetRunner sets the probe runner, should be set from the probe on init
func (c *ProbeHandler) SetRunner(r ProbeRunner) *ProbeHandler {
	c.runner = r
	return c
}

// SetFinisher sets the probe finishing method, should be set from the probe on init
func (c *ProbeHandler) SetFinisher(f ProbeFinisher) *ProbeHandler {
	c.finisher = f
	return c
}

// SetTimeStat saves probing statistics, should be called internally from the probe
func (c *ProbeHandler) SetTimeStat(name string, dur time.Duration) {
	c.Lock()
	if c.timings == nil {
		c.timings = make(map[string]time.Duration)
	}
	c.timings[name] = dur
	c.Unlock()
}

// TimeStat returns probing stat data
func (c *ProbeHandler) TimeStat() map[string]time.Duration {
	if c.timings == nil {
		return make(map[string]time.Duration)
	}
	return c.timings
}

// Log logouts the message, should be called internally from the probe
func (c *ProbeHandler) Log(format string, args ...any) {
	if c.logContext == "" {
		c.SetLogContext("")
	}
	var delim string
	if !strings.HasPrefix(format, "[") {
		delim = " "
	}
	log.Printf(c.logContext+delim+format+"\n", args...)
}

// SetLogContext sets log context, should be called internally from the probe
func (c *ProbeHandler) SetLogContext(ctx string) {
	if ctx != "" {
		c.logContext = "[" + ctx + "][" + c.Name + "]"
	} else {
		c.logContext = "[" + c.Name + "]"
	}
}

// Finish is just a stub for probe that should fit the Prober interface but doesn't have a Finish method itself
func (c *ProbeHandler) Finish(ctx context.Context) {
	if c.finisher != nil {
		c.finisher(ctx)
	}
	_ = c.EStatusFinish(true)
}

func (c *ProbeHandler) Error() error {
	return c.error
}

func (c *ProbeHandler) IsAlive() bool {
	return c.Runner.EStatus == EStatusRunning
}

type Prober interface {
	Start(ctx context.Context) (succ bool)
	Finish(ctx context.Context)
	Error() error
	IsAlive() bool
	TimeStat() map[string]time.Duration
}

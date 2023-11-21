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
	StayAlive bool          `json:"stayAlive,omitempty"` // probe process should stay alive after check is finished
	Expect    bool          `json:"expect,omitempty" default:"true"`
	Debug     bool          `json:"debug,omitempty" default:"false"`
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

type ProbeTimings struct {
	Timings map[string]time.Duration
	sync.Mutex
}

func (c *ProbeTimings) Set(name string, dur time.Duration) {
	c.Lock()
	if c.Timings == nil {
		c.Timings = make(map[string]time.Duration)
	}
	c.Timings[name] = dur
	c.Unlock()
}

// ProbeRunner is the probe runner that performs check
type ProbeRunner func(ctx context.Context) (succ bool)

// ProbeFinisher is the finish method for long-lived probe,
// is called cleanup code at the script end
type ProbeFinisher func(ctx context.Context)

type ProbeHandler struct {
	ProbeOptions `json:"-"`
	Runner
	Result
	ProbeResult       any
	Name              string                   // probe name
	CanStayBackground bool                     `json:"-"` // flag that means the probing process can stay in background
	runner            ProbeRunner              // probe runner
	finisher          ProbeFinisher            // probe finisher, only for probe that stays alive in background
	timings           map[string]time.Duration // timings data
	logContext        string                   // log prefix string
	error             error                    // last startup error
	//	sync.Mutex
}

// Start starts the probing and returns a probing result, don't call directly
func (c *ProbeHandler) Start(ctx context.Context) (succ bool) {
	if c.runner == nil {
		c.Log("runner isn't defined")
		return
	}
	if err := c.EStatusRun(); err != nil {
		c.Log("wrong runner status: %v", err.Error())
	}
	c.Result.PrepareToStart()

	var ctxID string
	if ctx != nil && ctx.Value("id") != nil {
		ctxID = ctx.Value("id").(string)
	}
	c.SetLogContext(ctxID)
	c.logDebug("Starting the probe runner")
	succ = c.runner(ctx)
	c.logDebug("The probe runner has been finished with success status: %v", succ)
	c.Result.End(succ)

	// todo do not clear && messy, need refactor
	if !c.CanStayBackground || !succ || c.Error() != nil || !c.StayAlive {
		_ = c.EStatusFinish(succ)
	} else {
		c.logDebug("The probe process stays alive")
	}

	if succ {
		c.Log("SUCCESS")
	} else {
		c.Log("FAIL")
	}
	//	c.SetLogContext("")

	return
}

// Finish finishes a long-lived background probe process, don't call directly
func (c *ProbeHandler) Finish(ctx context.Context) {
	if c.finisher != nil {
		c.logDebug("Finishing a background probe process")
		c.finisher(ctx)
	}
	_ = c.EStatusFinish(true)
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

func (c *ProbeHandler) logDebug(format string, args ...any) {
	if !c.Debug {
		return
	}
	c.Log("[debug] "+format, args...)
}

// SetLogContext sets log context, should be called internally from the probe
func (c *ProbeHandler) SetLogContext(ctx string) {
	if ctx != "" {
		c.logContext = "[" + ctx + "][" + c.Name + "]"
	} else {
		c.logContext = "[" + c.Name + "]"
	}
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
}

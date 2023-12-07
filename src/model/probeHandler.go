package model

import (
	"context"
	"strings"
)

type ProbeHandler struct {
	ProbeOptions      `json:"options"`
	Worker                   // describes status o probe working process
	Name              string // probe name
	Config            any
	CanStayBackground bool          `json:"-"` // flag that means the probing process can stay in background
	lastResult        Result        // represents the last probe running Result
	curResult         Result        // represents the current probe running Result
	probingData       any           // a specific probe implementation saves a lastResult object here
	runner            ProbeRunner   // probe runner func
	finisher          ProbeFinisher // probe finisher func, only for probe that stays alive in background
	logContext        string        // log prefix string
	error             error         // last startup error
	logger            Logger
}

// Start starts the probing and returns a probing curResult, don't call directly
func (c *ProbeHandler) Start(ctx context.Context) (succ bool) {
	if c.runner == nil {
		c.Log("runner isn't defined")
		return
	}
	if err := c.EStatusRun(); err != nil {
		c.Log("wrong runner status: %v", err.Error())
	}

	c.curResult.PrepareToStart()

	//var ctxID string
	//if ctx != nil && ctx.Value("id") != nil {
	//		ctxID = ctx.Value("id").(string)
	//}

	c.logger = GetLogger(ctx)
	//c.SetLogContext(ctxID)
	c.logDebug("Starting the probe runner")

	var probingData any
	defer func() {
		if err := recover(); err != nil {
			c.logger.Print("panic occurred:", err, "\n")
			succ = false
			_ = c.EStatusFinish(succ)
		}

		c.Lock()
		c.curResult.End(succ)
		c.lastResult = c.curResult
		c.probingData = probingData
		c.Unlock()

		if succ {
			c.Log("SUCCESS")
		} else {
			c.Log("FAIL")
		}
	}()
	succ, probingData = c.runner(ctx)

	c.logDebug("The probe runner has been finished with success status: %v", succ)

	// todo do not clear && messy, need refactor
	if !c.CanStayBackground || !succ || c.Error() != nil || !c.StayBackground {
		_ = c.EStatusFinish(succ)
	} else {
		c.logDebug("The probe process stays alive")
	}
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
	c.logger.Printf(c.logContext+delim+format+"\n", args...)
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
	return c.Worker.EStatus == EStatusRunning
}

func (c *ProbeHandler) Result() (r ProbeResult) {
	c.Lock()
	r.Name = c.Name
	r.Options = c.ProbeOptions
	if c.curResult.Completed() {
		r.Result = c.lastResult
		r.Data = c.probingData
	} else {
		r.Result = c.curResult
	}
	r.Configuration = c.Config
	c.Unlock()
	return
}

func (c *ProbeHandler) ResultFinished() (r ProbeResult) {
	c.Lock()
	r.Name = c.Name
	r.Options = c.ProbeOptions
	r.Result = c.lastResult
	r.Data = c.probingData
	r.Configuration = c.Config
	c.Unlock()
	return
}

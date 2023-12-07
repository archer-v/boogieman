package model

import (
	"context"
	"errors"
)

var ErrorConfig = errors.New("wrong configuration")

// ProbeRunner is the probe runner that performs a probe work
type ProbeRunner func(ctx context.Context) (succ bool, resultObject any)

// ProbeFinisher is the finish method for long-lived probe,
// is called by cleanup code at the script end
type ProbeFinisher func(ctx context.Context)

type ProbeResult struct {
	Name          string       `json:"name"`
	Options       ProbeOptions `json:"options"`
	Configuration any          `json:"configuration"`
	Result
	Data any `json:"data"`
}

type Prober interface {
	Start(ctx context.Context) (succ bool)
	Finish(ctx context.Context)
	Error() error
	IsAlive() bool
	Result() (r ProbeResult)
	ResultFinished() (r ProbeResult)
}

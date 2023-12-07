package model

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"time"
)

func init() {
	defaults.MustSet(&DefaultProbeOptions)
}

var DefaultProbeOptions ProbeOptions

type ProbeOptions struct {
	Timeout        time.Duration `json:"timeout,omitempty" default:"5000ms"`
	StayBackground bool          `json:"stayBackground,omitempty"` // a probe runner should stay alive after check is finished
	Expect         bool          `json:"expect" default:"true"`
	Debug          bool          `json:"debug,omitempty" default:"false"`
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
	t.Timeout *= time.Millisecond
	*s = ProbeOptions(t)

	return
}

func (s *ProbeOptions) MarshalJSON() ([]byte, error) {
	// Time should be represented in milliseconds
	type Options ProbeOptions
	o := Options(*s)
	o.Timeout = time.Duration(o.Timeout.Milliseconds())
	return json.Marshal(&o)
}

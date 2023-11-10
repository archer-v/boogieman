package model

import "time"

type Worker struct {
	PollInterval time.Duration
	CheckTimeout time.Duration
	Machine      Machine
	//Checks       CheckPlan
}

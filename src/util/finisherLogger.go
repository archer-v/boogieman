package util

import (
	"boogieman/src/model"
	"github.com/pseidemann/finish"
)

type logger struct {
	l model.Logger
}

func (l *logger) Infof(format string, v ...interface{}) {
	l.l.Printf(format, v...)
}

func (l *logger) Errorf(format string, v ...interface{}) {
	l.l.Printf(format, v...)
}

func FinisherLogger() finish.Logger {
	l := &logger{}
	l.l = model.NewChainLogger(model.DefaultLogger, "finisher")
	return l
}

package cmd

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

func Test_Runner(t *testing.T) {
	defOptions := model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true}

	type testCase struct {
		name           string
		config         any
		options        model.ProbeOptions
		expectedResult bool
		expectedError  error
		ctxTimeout     time.Duration
	}

	var testId string

	defer func() {
		if err := recover(); err != nil {
			log.Println(testId, " panic occurred:", err)
		}
	}()
	cases := []testCase{
		{
			"command with args is started and returns true",
			Config{Cmd: "ls", Args: []string{"-la"}, LogDump: false},
			defOptions,
			true,
			nil,
			0,
		},
		{
			"command with args is started and returns false",
			Config{Cmd: "ls", Args: []string{"-unknown_flag"}, LogDump: false},
			defOptions,
			false,
			nil,
			0,
		},
		{
			"command with described as a string is started and returns true",
			"test/cmd_sleep.sh 0.5 0",
			defOptions,
			true,
			nil,
			0,
		},
		{
			"command is started and aborted with timeout and returns false",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"2", "0"}},
			defOptions,
			false,
			nil,
			0,
		},
		{
			"command is started and stays alive and probe returns true",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"3", "0"}},
			model.ProbeOptions{Timeout: time.Millisecond * 200, Expect: true, StayBackground: true},
			true,
			nil,
			0,
		},
		{
			"command is started and have to stays alive but finished and probe returns false",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"0.5", "0"}},
			model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true, StayBackground: true, Debug: true},
			false,
			nil,
			0,
		},
		{
			"command is started and will be interrupted with context.deadline and should returns false",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"1", "0"}},
			model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true, StayBackground: false},
			false,
			nil,
			time.Millisecond * 500,
		},
		{
			"command is started and have to stays alive but will be interrupted with context.deadline and should returns false",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"1", "0"}},
			model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true, StayBackground: true},
			false,
			nil,
			time.Millisecond * 500,
		},
	}

	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}
	for i, c := range cases {
		testId = fmt.Sprintf("test %v", i+1)
		log.Printf("[%v] started", testId)
		p, err := constructor.NewProbe(c.options, c.config)
		if c.expectedError == nil && err != nil {
			t.Errorf("[%v] probe constructor returned error %v", testId, err)
			continue
		} else if err != c.expectedError {
			t.Errorf("[%v] probe constructor should return error %v", testId, c.expectedError)
			continue
		}
		var (
			ctx        context.Context
			cancelFunc context.CancelFunc
		)
		if c.ctxTimeout == 0 {
			ctx = context.Background()
		} else {
			ctx, cancelFunc = context.WithTimeout(context.Background(), c.ctxTimeout)
		}
		ctx = model.ContextWithLogger(ctx, model.NewChainLogger(model.DefaultLogger, testId))
		if p.Start(ctx) != c.expectedResult {
			t.Errorf("[%v] probe should return %v", testId, c.expectedResult)
			if cancelFunc != nil {
				cancelFunc()
			}
			continue
		}
		if cancelFunc != nil {
			cancelFunc()
		}
		p.Finish(ctx)
		log.Printf("[%v] OK", testId)
	}
}

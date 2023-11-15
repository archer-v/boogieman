package cmd

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCmd_Runner(t *testing.T) {
	ctx := context.Background()
	defOptions := model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true}

	type testCase struct {
		name           string
		config         any
		options        model.ProbeOptions
		expectedResult bool
		expectedError  error
	}

	cases := []testCase{
		{
			"command with args is started and returns true",
			Config{Cmd: "ls", Args: []string{"-la"}, LogDump: false},
			defOptions,
			true,
			nil,
		},
		{
			"command with args is started and returns false",
			Config{Cmd: "ls", Args: []string{"-unknown_flag"}, LogDump: false},
			defOptions,
			false,
			nil,
		},
		{
			"command with described as a string is started and returns true",
			"test/cmd_sleep.sh 0.5 0",
			defOptions,
			true,
			nil,
		},
		{
			"command is started and aborted with timeout and returns false",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"2", "0"}},
			defOptions,
			false,
			nil,
		},
		{
			"command is started and stays alive and probe returns true",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"3", "0"}},
			model.ProbeOptions{Timeout: time.Millisecond * 200, Expect: true, StayAlive: true},
			true,
			nil,
		},
		{
			"command is started and have to stays alive but finished and probe returns false",
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"0.5", "0"}},
			model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true, StayAlive: true},
			false,
			nil,
		},
	}

	contructor := constructor{
		probeFactory.BaseConstructor{
			Name: name,
		},
	}
	for i, c := range cases {
		p, err := contructor.NewProbe(c.options, c.config)
		if c.expectedError == nil && err != nil {
			t.Errorf("Probe %v constructor returned error %v", i, err)
			continue
		} else if err != c.expectedError {
			t.Errorf("Probe %v constructor should return error %v", i, c.expectedError)
			continue
		}
		if p.Start(context.WithValue(ctx, "id", fmt.Sprintf("test %v", i+1))) != c.expectedResult {
			t.Errorf("Probe runner %v should return %v", i, c.expectedResult)
		} else {
			p.Finish(ctx)
		}
	}
}

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
	options := model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true}

	type testCase struct {
		config         any
		expectedResult bool
		expectedError  error
	}

	cases := []testCase{
		{
			Config{Cmd: "ls", Args: []string{"-la"}, LogDump: false},
			true,
			nil,
		},
		{
			Config{Cmd: "ls", Args: []string{"-unknown_flag"}, LogDump: false},
			false,
			nil,
		},
		{
			"ls .",
			true,
			nil,
		},
		{
			"test/cmd_sleep.sh 0.5 0",
			true,
			nil,
		},
		{
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"0.5", "1"}, ExitCode: 1, LogDump: false},
			true,
			nil,
		},
		{
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"0.5", "1"}, ExitCode: 0, LogDump: false},
			false,
			nil,
		},
		{
			Config{Cmd: "test/cmd_sleep.sh", Args: []string{"2", "0"}, ExitCode: 0, LogDump: false},
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
		p, err := contructor.NewProbe(options, c.config)
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

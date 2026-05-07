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

func Test_RunnerStdoutRegex(t *testing.T) {
	defOptions := model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true}
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	type testCase struct {
		name           string
		config         Config
		expectedResult bool
	}

	cases := []testCase{
		{
			name: "stdout regex matches",
			config: Config{
				Cmd:         "sh",
				Args:        []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				StdoutRegex: `version:\s+\d+\.\d+\.\d+`,
			},
			expectedResult: true,
		},
		{
			name: "stdout regex does not match",
			config: Config{
				Cmd:         "sh",
				Args:        []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				StdoutRegex: `maintenance`,
			},
			expectedResult: false,
		},
		{
			name: "inverted stdout regex fails on match",
			config: Config{
				Cmd:               "sh",
				Args:              []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				StdoutRegex:       `status:\s+ok`,
				StdoutRegexInvert: true,
			},
			expectedResult: false,
		},
		{
			name: "inverted stdout regex succeeds without match",
			config: Config{
				Cmd:               "sh",
				Args:              []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				StdoutRegex:       `maintenance`,
				StdoutRegexInvert: true,
			},
			expectedResult: true,
		},
		{
			name: "exit code mismatch still fails",
			config: Config{
				Cmd:         "sh",
				Args:        []string{"-c", "printf 'service version: 1.2.3\\n'; exit 7"},
				ExitCode:    0,
				StdoutRegex: `version`,
			},
			expectedResult: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := constructor.NewProbe(defOptions, c.config)
			if err != nil {
				t.Fatalf("constructor returned error: %v", err)
			}
			ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, c.name))
			if p.Start(ctx) != c.expectedResult {
				t.Fatalf("probe should return %v", c.expectedResult)
			}
			p.Finish(ctx)
		})
	}
}

func Test_ConstructorWrongStdoutRegex(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	_, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:         "sh",
			Args:        []string{"-c", "printf test"},
			StdoutRegex: "[",
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid stdoutRegex")
	}
}

func Test_RunnerStdoutRegexCapture(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	p, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:                     "sh",
			Args:                    []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
			StdoutRegex:             `version:\s+(\d+\.\d+\.\d+)`,
			StdoutRegexCaptureGroup: 1,
		},
	)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "stdout capture"))
	if !p.Start(ctx) {
		t.Fatal("probe should return true")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.ExitCode != 0 {
		t.Fatalf("exit code should be 0, got %d", data.ExitCode)
	}
	if data.Capture != "1.2.3" {
		t.Fatalf("capture should be %q, got %q", "1.2.3", data.Capture)
	}
	p.Finish(ctx)
}

func Test_ConstructorWrongStdoutRegexCaptureGroup(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	_, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:                     "sh",
			Args:                    []string{"-c", "printf test"},
			StdoutRegex:             `version:\s+(\d+)`,
			StdoutRegexCaptureGroup: 2,
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid stdoutRegexCaptureGroup")
	}
}

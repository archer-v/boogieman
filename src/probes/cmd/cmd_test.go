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

func boolPtr(v bool) *bool {
	return &v
}

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

func Test_RunnerRegex(t *testing.T) {
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
				Cmd:   "sh",
				Args:  []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex: `version:\s+\d+\.\d+\.\d+`,
			},
			expectedResult: true,
		},
		{
			name: "stdout regex does not match",
			config: Config{
				Cmd:   "sh",
				Args:  []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex: `maintenance`,
			},
			expectedResult: false,
		},
		{
			name: "inverted stdout regex fails on match",
			config: Config{
				Cmd:         "sh",
				Args:        []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex:       `status:\s+ok`,
				RegexInvert: true,
			},
			expectedResult: false,
		},
		{
			name: "inverted stdout regex succeeds without match",
			config: Config{
				Cmd:         "sh",
				Args:        []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex:       `maintenance`,
				RegexInvert: true,
			},
			expectedResult: true,
		},
		{
			name: "exit code mismatch still fails",
			config: Config{
				Cmd:      "sh",
				Args:     []string{"-c", "printf 'service version: 1.2.3\\n'; exit 7"},
				ExitCode: 0,
				Regex:    `version`,
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

func Test_ConstructorWrongRegex(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	_, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:   "sh",
			Args:  []string{"-c", "printf test"},
			Regex: "[",
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid regex")
	}
}

func Test_RunnerRegexCapture(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	p, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:               "sh",
			Args:              []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
			Regex:             `version:\s+(\d+\.\d+\.\d+)`,
			RegexCaptureGroup: 1,
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
	if data.Regex == nil || !*data.Regex {
		t.Fatal("regex result should be true")
	}
	if data.Capture == nil || *data.Capture != "1.2.3" {
		t.Fatalf("capture should be %q, got %v", "1.2.3", data.Capture)
	}
	p.Finish(ctx)
}

func Test_RunnerReturnsDataOnFailedCondition(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	p, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:               "sh",
			Args:              []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'; exit 7"},
			ExitCode:          0,
			Regex:             `version:\s+(\d+\.\d+\.\d+)`,
			RegexCaptureGroup: 1,
		},
	)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "failed condition data"))
	if p.Start(ctx) {
		t.Fatal("probe should return false because exit code does not match")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.ExitCode != 7 {
		t.Fatalf("exit code should be 7, got %d", data.ExitCode)
	}
	if data.Regex == nil || !*data.Regex {
		t.Fatal("regex result should be true")
	}
	if data.Capture == nil || *data.Capture != "1.2.3" {
		t.Fatalf("capture should be %q, got %v", "1.2.3", data.Capture)
	}
	p.Finish(ctx)
}

func Test_RunnerRegexDoesNotValidateWhenDisabled(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	p, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:           "sh",
			Args:          []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
			Regex:         `maintenance`,
			RegexRequired: boolPtr(false),
		},
	)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "regex does not validate"))
	if !p.Start(ctx) {
		t.Fatal("probe should return true when regexRequired is false")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.Regex == nil {
		t.Fatal("regex result should be exported")
	}
	if *data.Regex {
		t.Fatal("regex result should be false")
	}
	p.Finish(ctx)
}

func Test_RunnerRegexCaptureAddsEmptyValueWhenNotMatched(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	p, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:               "sh",
			Args:              []string{"-c", "printf 'service version unavailable\\nstatus: ok\\n'"},
			Regex:             `version:\s+(\d+\.\d+\.\d+)`,
			RegexRequired:     boolPtr(false),
			RegexCaptureGroup: 1,
		},
	)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "empty capture"))
	if !p.Start(ctx) {
		t.Fatal("probe should return true when regexRequired is false")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.Regex == nil {
		t.Fatal("regex result should be exported")
	}
	if *data.Regex {
		t.Fatal("regex result should be false")
	}
	if data.Capture == nil {
		t.Fatal("capture should be exported even when regex does not match")
	}
	if *data.Capture != "" {
		t.Fatalf("capture should be empty, got %q", *data.Capture)
	}
	p.Finish(ctx)
}

func Test_RunnerCaptureRegex(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	type testCase struct {
		name           string
		config         Config
		expectedResult bool
		expectedMatch  bool
	}

	cases := []testCase{
		{
			name: "capture regex matches",
			config: Config{
				Cmd:               "sh",
				Args:              []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex:             `version:\s+(\d+\.\d+\.\d+)`,
				RegexCaptureGroup: 1,
				CaptureRegex:      `^1\.2\.`,
			},
			expectedResult: true,
			expectedMatch:  true,
		},
		{
			name: "capture regex does not match",
			config: Config{
				Cmd:               "sh",
				Args:              []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex:             `version:\s+(\d+\.\d+\.\d+)`,
				RegexCaptureGroup: 1,
				CaptureRegex:      `^2\.`,
			},
			expectedResult: false,
			expectedMatch:  false,
		},
		{
			name: "inverted capture regex succeeds without match",
			config: Config{
				Cmd:                "sh",
				Args:               []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex:              `version:\s+(\d+\.\d+\.\d+)`,
				RegexCaptureGroup:  1,
				CaptureRegex:       `^2\.`,
				CaptureRegexInvert: true,
			},
			expectedResult: true,
			expectedMatch:  false,
		},
		{
			name: "inverted capture regex fails on match",
			config: Config{
				Cmd:                "sh",
				Args:               []string{"-c", "printf 'service version: 1.2.3\\nstatus: ok\\n'"},
				Regex:              `version:\s+(\d+\.\d+\.\d+)`,
				RegexCaptureGroup:  1,
				CaptureRegex:       `^1\.2\.`,
				CaptureRegexInvert: true,
			},
			expectedResult: false,
			expectedMatch:  true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := constructor.NewProbe(
				model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
				c.config,
			)
			if err != nil {
				t.Fatalf("constructor returned error: %v", err)
			}

			ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, c.name))
			if p.Start(ctx) != c.expectedResult {
				t.Fatalf("probe should return %v", c.expectedResult)
			}

			result := p.Result()
			data, ok := result.Data.(ResultData)
			if !ok {
				t.Fatalf("probe data should be ResultData, got %T", result.Data)
			}
			if data.Capture == nil || *data.Capture != "1.2.3" {
				t.Fatalf("capture should be %q, got %v", "1.2.3", data.Capture)
			}
			if data.CaptureMatches == nil || *data.CaptureMatches != c.expectedMatch {
				t.Fatalf("capture regex result should be %v", c.expectedMatch)
			}
			p.Finish(ctx)
		})
	}
}

func Test_ConstructorWrongCaptureRegex(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	_, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:               "sh",
			Args:              []string{"-c", "printf test"},
			Regex:             `version:\s+(\d+)`,
			RegexCaptureGroup: 1,
			CaptureRegex:      "[",
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid captureRegex")
	}

	_, err = constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:          "sh",
			Args:         []string{"-c", "printf test"},
			Regex:        `version:\s+(\d+)`,
			CaptureRegex: `^\d+$`,
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error when captureRegex has no regexCaptureGroup")
	}
}

func Test_ConstructorWrongRegexCaptureGroup(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	_, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 1000, Expect: true},
		Config{
			Cmd:               "sh",
			Args:              []string{"-c", "printf test"},
			Regex:             `version:\s+(\d+)`,
			RegexCaptureGroup: 2,
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid regexCaptureGroup")
	}
}

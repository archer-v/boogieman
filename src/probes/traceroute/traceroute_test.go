package traceroute

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

var defPublicTestHost = "8.8.8.8"

func Test_Runner(t *testing.T) {
	gitHubRunner := os.Getenv("GITHUB_ACTIONS") == "true"
	if gitHubRunner {
		defPublicTestHost = "127.0.0.1"
	}
	defOptions := model.ProbeOptions{Timeout: time.Millisecond * 20000, Expect: true}
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	type testCase struct {
		name           string
		config         any
		options        model.ProbeOptions
		expectedResult bool
		expectedError  error
		ctxTimeout     time.Duration
	}

	cases := []testCase{
		{
			"traceroute to localhost host",
			Config{Host: defPublicTestHost, ExpectedHops: []string{defPublicTestHost}, HopTimeout: time.Millisecond * 100, Retries: 1, LogDump: true},
			defOptions,
			true,
			nil,
			0,
		},
		{
			"traceroute to an existent host when config is defined in string",
			defPublicTestHost + "," + defPublicTestHost,
			defOptions,
			true,
			nil,
			0,
		},
		{
			"wrong configuration",
			"aaa",
			defOptions,
			false,
			model.ErrorConfig,
			0,
		},
		{
			"traceroute to wrong host",
			Config{Host: "192.168.10.10", ExpectedHops: []string{"aaa"}, HopTimeout: time.Millisecond * 100, Retries: 2, LogDump: true, MaxHops: 5},
			defOptions,
			false,
			nil,
			0,
		},
		{
			"traceroute to wrong host with context timeout",
			Config{Host: "192.168.10.10", ExpectedHops: []string{"aaa"}, HopTimeout: time.Millisecond * 200, Retries: 1, LogDump: true},
			defOptions,
			false,
			nil,
			time.Millisecond * 500,
		},
	}

	for i, c := range cases {
		caseName := fmt.Sprintf("test %v", i+1)
		fmt.Printf("%v running\n", caseName)
		p, err := constructor.NewProbe(c.options, c.config)
		switch {
		case c.expectedError == nil && err != nil:
			t.Errorf("Probe %v constructor returned error %v", i, err)
			continue
		case err != c.expectedError:
			t.Errorf("Probe %v constructor should return error %v", i, c.expectedError)
			continue
		case err != nil:
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
		ctx = model.ContextWithLogger(ctx, model.NewChainLogger(model.DefaultLogger, fmt.Sprintf("test %v", i+1)))
		rz := p.Start(ctx)
		if cancelFunc != nil {
			cancelFunc()
		}
		if rz != c.expectedResult {
			t.Errorf("Test %v - '%v' should return %v", i, c.name, c.expectedResult)
			continue
		}
		p.Finish(ctx)
		fmt.Printf("%v OK\n", caseName)
	}
}

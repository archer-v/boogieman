package traceroute

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_Runner(t *testing.T) {

	ctx := context.Background()
	defOptions := model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true}
	constructor := constructor{
		probeFactory.BaseConstructor{
			Name: name,
		},
	}

	type testCase struct {
		name           string
		config         any
		options        model.ProbeOptions
		expectedResult bool
		expectedError  error
	}

	cases := []testCase{
		/*
			{
				"traceroute to an existent host",
				Config{Host: "google.com", ExpectedHops: []string{"google.com"}, HopTimeout: time.Millisecond * 200, Retries: 2, LogDump: true},
				defOptions,
				true,
				nil,
			},
			{
				"traceroute to an existent host when config is defined in string",
				"google.com,google.com",
				defOptions,
				true,
				nil,
			},
			{
				"wrong configuration",
				"aaa",
				defOptions,
				false,
				model.ErrorConfig,
			},


		*/
		{
			"traceroute to wrong host",
			Config{Host: "192.168.10.10", ExpectedHops: []string{"aaa"}, HopTimeout: time.Millisecond * 200, Retries: 2, LogDump: true},
			defOptions,
			false,
			nil,
		},
	}

	for i, c := range cases {
		caseName := fmt.Sprintf("test %v", i+1)
		fmt.Printf("Executing case [%v]\n", caseName)
		p, err := constructor.NewProbe(c.options, c.config)
		if c.expectedError == nil && err != nil {
			t.Errorf("Probe %v constructor returned error %v", i, err)
			continue
		} else if err != c.expectedError {
			t.Errorf("Probe %v constructor should return error %v", i, c.expectedError)
			continue
		} else if err != nil {
			continue
		}
		if p.Start(context.WithValue(ctx, "id", caseName)) != c.expectedResult {
			t.Errorf("Probe runner %v should return %v", i, c.expectedResult)
		} else {
			p.Finish(ctx)
		}
	}
}

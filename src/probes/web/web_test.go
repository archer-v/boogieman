package web

import (
	"boogieman/src/model"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWeb_Runner(t *testing.T) {

	ctx := context.Background()
	options := model.ProbeOptions{Timeout: time.Millisecond * 5000}

	type testCase struct {
		config         Config
		expectedResult bool
	}

	cases := []testCase{
		{
			Config{Urls: []string{"https://google.com/", "https://google.com/robots.txt"}},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/"}},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/fail"}},
			false,
		},
		{
			Config{Urls: []string{"https://google.com/", "https://google.com/fail"}},
			false,
		},
	}

	for i, c := range cases {
		p := New(options, c.config)
		if p.Start(context.WithValue(ctx, "id", fmt.Sprintf("test %v", i+1))) != c.expectedResult {
			t.Errorf("Probe runner %v should return %v", i, c.expectedResult)
		}
	}
}

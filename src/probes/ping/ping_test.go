package ping

import (
	"boogieman/src/model"
	"context"
	"fmt"
	"testing"
	"time"
)

func Test_Runner(t *testing.T) {

	ctx := context.Background()
	options := model.ProbeOptions{Timeout: time.Millisecond * 2000, Expect: true}

	type testCase struct {
		config         Config
		expectedResult bool
	}

	cases := []testCase{
		{
			Config{Hosts: []string{"127.0.0.1", "127.0.0.2"}},
			true,
		},
		{
			Config{Hosts: []string{"google.com"}},
			true,
		},
		{
			Config{Hosts: []string{"gsdfsdfsdfsdfsdfsd.com"}},
			false,
		},
		{
			Config{Hosts: []string{"192.168.168.168", "google.com"}},
			false,
		},
	}

	for i, c := range cases {
		p := New(options, c.config)
		ctx := model.ContextWithLogger(ctx, model.NewChainLogger(model.DefaultLogger, fmt.Sprintf("test %v", i+1)))
		if p.Start(ctx) != c.expectedResult {
			t.Errorf("Probe runner %v should return %v", i, c.expectedResult)
		}
	}
}

package web

import (
	"boogieman/src/model"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func Test_Runner(t *testing.T) {

	ctx := context.Background()
	options := model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true}

	type testCase struct {
		config         Config
		expectedResult bool
	}

	cases := []testCase{
		{
			Config{Urls: []string{"https://google.com/", "https://google.com/robots.txt"}, HTTPStatus: http.StatusOK},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/"}, HTTPStatus: http.StatusOK},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/fail"}, HTTPStatus: http.StatusOK},
			false,
		},
		{
			Config{Urls: []string{"https://google.com/fail"}, HTTPStatus: http.StatusNotFound},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/", "https://google.com/fail"}, HTTPStatus: http.StatusOK},
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

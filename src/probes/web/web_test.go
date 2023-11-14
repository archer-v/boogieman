package web

import (
	"boogieman/src/model"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestWeb_Runner(t *testing.T) {

	ctx := context.Background()
	options := model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true}

	type testCase struct {
		config         Config
		expectedResult bool
	}

	cases := []testCase{
		{
			Config{Urls: []string{"https://google.com/", "https://google.com/robots.txt"}, HttpStatus: http.StatusOK},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/"}, HttpStatus: http.StatusOK},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/fail"}, HttpStatus: http.StatusOK},
			false,
		},
		{
			Config{Urls: []string{"https://google.com/fail"}, HttpStatus: http.StatusNotFound},
			true,
		},
		{
			Config{Urls: []string{"https://google.com/", "https://google.com/fail"}, HttpStatus: http.StatusOK},
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

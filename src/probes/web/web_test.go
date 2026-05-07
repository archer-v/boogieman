package web

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func Test_RunnerBodyRegex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("service version: 1.2.3\nstatus: ok"))
	}))
	defer server.Close()

	options := model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true}
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
			name: "body regex matches",
			config: Config{
				Urls:       []string{server.URL},
				HTTPStatus: http.StatusOK,
				BodyRegex:  `version:\s+\d+\.\d+\.\d+`,
			},
			expectedResult: true,
		},
		{
			name: "body regex does not match",
			config: Config{
				Urls:       []string{server.URL},
				HTTPStatus: http.StatusOK,
				BodyRegex:  `maintenance`,
			},
			expectedResult: false,
		},
		{
			name: "inverted body regex fails on match",
			config: Config{
				Urls:            []string{server.URL},
				HTTPStatus:      http.StatusOK,
				BodyRegex:       `status:\s+ok`,
				BodyRegexInvert: true,
			},
			expectedResult: false,
		},
		{
			name: "inverted body regex succeeds without match",
			config: Config{
				Urls:            []string{server.URL},
				HTTPStatus:      http.StatusOK,
				BodyRegex:       `maintenance`,
				BodyRegexInvert: true,
			},
			expectedResult: true,
		},
		{
			name: "status mismatch still fails",
			config: Config{
				Urls:       []string{server.URL},
				HTTPStatus: http.StatusNotFound,
				BodyRegex:  `version`,
			},
			expectedResult: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := constructor.NewProbe(options, c.config)
			if err != nil {
				t.Fatalf("constructor returned error: %v", err)
			}
			ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, c.name))
			if p.Start(ctx) != c.expectedResult {
				t.Fatalf("probe should return %v", c.expectedResult)
			}
		})
	}
}

func Test_ConstructorWrongBodyRegex(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	_, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls:       []string{"https://example.com/"},
			HTTPStatus: http.StatusOK,
			BodyRegex:  "[",
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid bodyRegex")
	}
}

func Test_RunnerBodyRegexCapture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("service version: 1.2.3\nstatus: ok"))
	}))
	defer server.Close()

	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	p, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls:                  []string{server.URL},
			HTTPStatus:            http.StatusOK,
			BodyRegex:             `version:\s+(\d+\.\d+\.\d+)`,
			BodyRegexCaptureGroup: 1,
		},
	)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "body capture"))
	if !p.Start(ctx) {
		t.Fatal("probe should return true")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.Captures[server.URL] != "1.2.3" {
		t.Fatalf("capture should be %q, got %q", "1.2.3", data.Captures[server.URL])
	}
	if _, ok = data.Timings[server.URL]; !ok {
		t.Fatal("timing should be exported")
	}
}

func Test_ConstructorWrongBodyRegexCaptureGroup(t *testing.T) {
	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	_, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls:                  []string{"https://example.com/"},
			HTTPStatus:            http.StatusOK,
			BodyRegex:             `version:\s+(\d+)`,
			BodyRegexCaptureGroup: 2,
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid bodyRegexCaptureGroup")
	}
}

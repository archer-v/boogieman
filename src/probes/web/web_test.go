package web

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func boolPtr(v bool) *bool {
	return &v
}

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

func Test_RunnerRegex(t *testing.T) {
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
				Regex:      `version:\s+\d+\.\d+\.\d+`,
			},
			expectedResult: true,
		},
		{
			name: "body regex does not match",
			config: Config{
				Urls:       []string{server.URL},
				HTTPStatus: http.StatusOK,
				Regex:      `maintenance`,
			},
			expectedResult: false,
		},
		{
			name: "inverted body regex fails on match",
			config: Config{
				Urls:        []string{server.URL},
				HTTPStatus:  http.StatusOK,
				Regex:       `status:\s+ok`,
				RegexInvert: true,
			},
			expectedResult: false,
		},
		{
			name: "inverted body regex succeeds without match",
			config: Config{
				Urls:        []string{server.URL},
				HTTPStatus:  http.StatusOK,
				Regex:       `maintenance`,
				RegexInvert: true,
			},
			expectedResult: true,
		},
		{
			name: "status mismatch still fails",
			config: Config{
				Urls:       []string{server.URL},
				HTTPStatus: http.StatusNotFound,
				Regex:      `version`,
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

func Test_ConstructorWrongRegex(t *testing.T) {
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
			Regex:      "[",
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid regex")
	}
}

func Test_RunnerRegexCapture(t *testing.T) {
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
			Urls:              []string{server.URL},
			HTTPStatus:        http.StatusOK,
			Regex:             `version:\s+(\d+\.\d+\.\d+)`,
			RegexCaptureGroup: 1,
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
	if data.HTTPStatus[server.URL] != http.StatusOK {
		t.Fatalf("http status should be %d, got %d", http.StatusOK, data.HTTPStatus[server.URL])
	}
	if !data.Regex[server.URL] {
		t.Fatal("regex result should be true")
	}
	if _, ok = data.Timings[server.URL]; !ok {
		t.Fatal("timing should be exported")
	}
}

func Test_RunnerHTTPStatusIsOptional(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("status: teapot"))
	}))
	defer server.Close()

	p := New(
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls: []string{server.URL},
		},
	)

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "optional status"))
	if !p.Start(ctx) {
		t.Fatal("probe should return true when HTTPStatus is 0 and endpoint returns an HTTP response")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.HTTPStatus[server.URL] != http.StatusTeapot {
		t.Fatalf("http status should be %d, got %d", http.StatusTeapot, data.HTTPStatus[server.URL])
	}
	if _, ok = data.Timings[server.URL]; !ok {
		t.Fatal("timing should be exported")
	}
}

func Test_RunnerReturnsDataOnFailedCondition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("service version: 1.2.3\nstatus: pending"))
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
			Urls:              []string{server.URL},
			HTTPStatus:        http.StatusOK,
			Regex:             `version:\s+(\d+\.\d+\.\d+)`,
			RegexCaptureGroup: 1,
		},
	)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "failed condition data"))
	if p.Start(ctx) {
		t.Fatal("probe should return false because HTTP status does not match")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.HTTPStatus[server.URL] != http.StatusAccepted {
		t.Fatalf("http status should be %d, got %d", http.StatusAccepted, data.HTTPStatus[server.URL])
	}
	if data.Captures[server.URL] != "1.2.3" {
		t.Fatalf("capture should be %q, got %q", "1.2.3", data.Captures[server.URL])
	}
	if !data.Regex[server.URL] {
		t.Fatal("regex result should be true")
	}
	if _, ok = data.Timings[server.URL]; !ok {
		t.Fatal("timing should be exported")
	}
}

func Test_RunnerReturnsDataOnConnectionFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	url := "http://" + listener.Addr().String()
	if err = listener.Close(); err != nil {
		t.Fatalf("listener close failed: %v", err)
	}

	constructor := constructor{
		probefactory.BaseConstructor{
			Name: name,
		},
	}

	p, err := constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls:              []string{url},
			Regex:             `version:\s+(\d+\.\d+\.\d+)`,
			RegexCaptureGroup: 1,
			CaptureRegex:      `^1\.2\.`,
		},
	)
	if err != nil {
		t.Fatalf("constructor returned error: %v", err)
	}

	ctx := model.ContextWithLogger(context.Background(), model.NewChainLogger(model.DefaultLogger, "connection failure data"))
	if p.Start(ctx) {
		t.Fatal("probe should return false because connection fails")
	}

	result := p.Result()
	data, ok := result.Data.(ResultData)
	if !ok {
		t.Fatalf("probe data should be ResultData, got %T", result.Data)
	}
	if data.Regex[url] {
		t.Fatal("regex result should be false")
	}
	capture, ok := data.Captures[url]
	if !ok {
		t.Fatal("capture should be exported even when connection fails")
	}
	if capture != "" {
		t.Fatalf("capture should be empty, got %q", capture)
	}
	if data.CaptureMatches[url] {
		t.Fatal("capture match should be false")
	}
	if _, ok = data.HTTPStatus[url]; ok {
		t.Fatal("http status should not be exported when endpoint does not respond")
	}
	if _, ok = data.Timings[url]; ok {
		t.Fatal("timing should not be exported when endpoint does not respond")
	}
}

func Test_RunnerRegexDoesNotValidateWhenDisabled(t *testing.T) {
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
			Urls:          []string{server.URL},
			HTTPStatus:    http.StatusOK,
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
	if data.Regex[server.URL] {
		t.Fatal("regex result should be false")
	}
}

func Test_RunnerRegexCaptureAddsEmptyValueWhenNotMatched(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("service version unavailable\nstatus: ok"))
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
			Urls:              []string{server.URL},
			HTTPStatus:        http.StatusOK,
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
	if data.Regex[server.URL] {
		t.Fatal("regex result should be false")
	}
	capture, ok := data.Captures[server.URL]
	if !ok {
		t.Fatal("capture should be exported even when regex does not match")
	}
	if capture != "" {
		t.Fatalf("capture should be empty, got %q", capture)
	}
}

func Test_RunnerCaptureRegex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("service version: 1.2.3\nstatus: ok"))
	}))
	defer server.Close()

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
				Urls:              []string{server.URL},
				HTTPStatus:        http.StatusOK,
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
				Urls:              []string{server.URL},
				HTTPStatus:        http.StatusOK,
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
				Urls:               []string{server.URL},
				HTTPStatus:         http.StatusOK,
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
				Urls:               []string{server.URL},
				HTTPStatus:         http.StatusOK,
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
				model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
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
			if data.Captures[server.URL] != "1.2.3" {
				t.Fatalf("capture should be %q, got %q", "1.2.3", data.Captures[server.URL])
			}
			if data.CaptureMatches[server.URL] != c.expectedMatch {
				t.Fatalf("capture regex result should be %v", c.expectedMatch)
			}
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
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls:              []string{"https://example.com/"},
			HTTPStatus:        http.StatusOK,
			Regex:             `version:\s+(\d+)`,
			RegexCaptureGroup: 1,
			CaptureRegex:      "[",
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid captureRegex")
	}

	_, err = constructor.NewProbe(
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls:         []string{"https://example.com/"},
			HTTPStatus:   http.StatusOK,
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
		model.ProbeOptions{Timeout: time.Millisecond * 5000, Expect: true},
		Config{
			Urls:              []string{"https://example.com/"},
			HTTPStatus:        http.StatusOK,
			Regex:             `version:\s+(\d+)`,
			RegexCaptureGroup: 2,
		},
	)
	if err == nil {
		t.Fatal("constructor should return an error for invalid regexCaptureGroup")
	}
}

//go:build !linux

package web

import (
	"net/http"
	"time"
)

func newHTTPClient(timeout time.Duration, _ int) http.Client {
	return http.Client{Timeout: timeout}
}

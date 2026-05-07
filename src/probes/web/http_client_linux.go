//go:build linux

package web

import (
	"context"
	"net"
	"net/http"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func newHTTPClient(timeout time.Duration, fwMark int) http.Client {
	if fwMark == 0 {
		return http.Client{Timeout: timeout}
	}

	dialer := &net.Dialer{
		Timeout: timeout,
		ControlContext: func(_ context.Context, _, _ string, conn syscall.RawConn) error {
			var sockErr error
			err := conn.Control(func(fd uintptr) {
				sockErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, fwMark)
			})
			if err != nil {
				return err
			}
			return sockErr
		},
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext

	return http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

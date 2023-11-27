package webserver

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

func Run(bindTo string) (srv *http.Server, err error) {
	srv = &http.Server{Addr: bindTo, ReadHeaderTimeout: time.Second}

	startupError := make(chan error)
	go func() {
		e := srv.ListenAndServe()
		if e != http.ErrServerClosed {
			startupError <- e
		}
	}()

	select {
	case <-time.After(time.Millisecond * 100):
	case err = <-startupError:
	}

	if err != nil {
		err = fmt.Errorf("can't start web server: %w", err)
	} else {
		logger.Printf("Webserver is listen on %v\n", bindTo)
	}
	return
}

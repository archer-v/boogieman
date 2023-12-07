package webserver

import (
	"boogieman/src/model"
	"fmt"
	"net/http"
	"time"
)

var logger = model.DefaultLogger

type Handlers []WebServed

type WebServed interface {
	URLPatters() []string
	ServeHTTP(http.ResponseWriter, *http.Request)
}

func Run(bindTo string, handlers Handlers) (srv *http.Server, err error) {
	srv = &http.Server{Addr: bindTo, ReadHeaderTimeout: time.Second}

	logger := model.NewChainLogger(logger, "webserver")

	for _, v := range handlers {
		for _, h := range v.URLPatters() {
			logger.Printf("add endpoint %v", h)
			http.Handle(h, v)
		}
	}

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
		logger.Printf("started and listen on %v\n", bindTo)
	}
	return
}

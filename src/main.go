package main

import (
	"boogieman/src/configuration"
	"boogieman/src/model"
	"boogieman/src/services/prometheus"
	"boogieman/src/services/scheduler"
	"boogieman/src/services/webserver"
	"boogieman/src/util"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pseidemann/finish"
	"io"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

const (
	ExitOk        = 0
	ExitFailed    = 1
	ExitErrConfig = 2
)

const (
	ShutdownWaitingTimeout = 30 * time.Second
)

var gitTag, gitCommit, gitBranch, buildTimestamp string

var finisher = &finish.Finisher{
	Timeout: ShutdownWaitingTimeout,
	Log:     util.FinisherLogger(),
}

func main() {
	version := versionString()
	configuration.AppVersion = version
	configuration.AppDescriptionMessage = "version: " + version

	config, err := configuration.StartupConfiguration()
	if err != nil {
		fmt.Printf("Wrong startup configuration: %v\n", err)
		os.Exit(ExitErrConfig)
	}

	if config.Mode == configuration.StartupModeWrong {
		os.Exit(ExitErrConfig)
	}

	// start in oneRun working mode
	if config.Script != nil {
		runScriptAndExit(config)
	}

	// daemon mode
	schedulerService := scheduler.Run()
	finisher.Add(schedulerService, finish.WithName("scheduler"))

	// prometheus
	prometheusService := prometheus.Run(true, true, schedulerService)

	webService, err := webserver.Run(config.BindTo, []webserver.WebServed{schedulerService, prometheusService})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(ExitErrConfig)
	}
	finisher.Add(webService, finish.WithName("web server"))

	for _, j := range config.ScheduleJobs {
		err = schedulerService.AddJob(j)
		if err != nil {
			log.Printf("[%v] error with creating a scheduling job: %v\n", j.Name, err)
		}
	}

	if config.ExitOnConfigChange {
		watcher, err := util.Watcher(
			[]string{config.ConfigFileName},
			func(path, op string) {
				// exit
				_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			})
		if err != nil {
			log.Printf("error with creating a file watcher: %v\n", err)
			os.Exit(ExitErrConfig)
		}
		finisher.Add(watcher, finish.WithName("file watcher"))
		for _, j := range config.ScheduleJobs {
			_ = watcher.Add(j.ScriptFile)
		}
	}

	finisher.Wait()
	os.Exit(ExitOk)
}

func runScriptAndExit(config configuration.StartupConfig) {
	ctx := context.Background()
	config.JSON = config.JSON || config.PrettyJSON
	if config.JSON {
		// fake logger in order to suppress all log output
		model.DefaultLogger = log.New(io.Discard, "", 0)
	}
	config.Script.Run(ctx)
	if config.JSON {
		var d []byte
		if config.PrettyJSON {
			d, _ = json.MarshalIndent(config.Script.Result(), "", "    ")
		} else {
			d, _ = json.Marshal(config.Script.Result())
		}
		fmt.Println(string(d))
	}

	if config.Script.Result().Success {
		os.Exit(ExitOk)
	}

	os.Exit(ExitFailed)
}

func versionString() (version string) {
	if buildTimestamp == "" {
		version = "DEV"
	} else {
		var ids []string
		if gitTag != "" {
			ids = append(ids, gitTag)
		}
		if gitBranch != "" {
			ids = append(ids, gitBranch)
		}
		if gitCommit != "" {
			ids = append(ids, gitCommit)
		}
		version = fmt.Sprintf("%v, build: %v", strings.Join(ids, "-"), buildTimestamp)
	}
	return
}

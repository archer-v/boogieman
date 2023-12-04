package main

import (
	"boogieman/src/configuration"
	"boogieman/src/services/prometheus"
	"boogieman/src/services/scheduler"
	"boogieman/src/services/webserver"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pseidemann/finish"
	"log"
	"os"
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

var gitTag, gitCommit, gitBranch, buildTimestamp, version string

var finisher = &finish.Finisher{Timeout: ShutdownWaitingTimeout}

func main() {
	if buildTimestamp == "" {
		version = "version: DEV"
	} else {
		version = fmt.Sprintf("version: %v-%v-%v, build: %v", gitTag, gitBranch, gitCommit, buildTimestamp)
	}

	config, err := configuration.StartupConfiguration()
	if err != nil {
		fmt.Print(err.Error())
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
	promCollector := prometheus.Run(true, true)

	webService, err := webserver.Run(config.BindTo, []webserver.WebServed{schedulerService, promCollector})
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(ExitErrConfig)
	}
	finisher.Add(webService, finish.WithName("web server"))

	for _, j := range config.ScheduleJobs {
		err = schedulerService.AddJob(j)
		if err != nil {
			log.Printf("[%v] error with creating a scheduling job: %v", j.Name, err)
		}
	}
	finisher.Wait()
	os.Exit(ExitOk)
}

func runScriptAndExit(config configuration.StartupConfig) {
	ctx := context.Background()
	config.Script.Run(ctx)
	if config.JSON {
		var d []byte
		if config.OutputPretty {
			d, _ = json.MarshalIndent(config.Script, "", "    ")
		} else {
			d, _ = json.Marshal(config.Script)
		}
		fmt.Println(string(d))
	}

	d, _ := json.MarshalIndent(config.Script.Result(), "", "    ")
	fmt.Println(string(d))

	if config.Script.Result().Success {
		os.Exit(ExitOk)
	}

	os.Exit(ExitFailed)
}

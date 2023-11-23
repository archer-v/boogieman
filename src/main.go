package main

import (
	"boogieman/src/configuration"
	"boogieman/src/runner"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron"
	"log"
	"os"
	"time"
)

const (
	ExitOk        = 0
	ExitFailed    = 1
	ExitErrConfig = 2
)

var gitTag, gitCommit, gitBranch, buildTimestamp, version string

func main() {

	if buildTimestamp == "" {
		version = fmt.Sprintf("version: DEV")
	} else {
		version = fmt.Sprintf("version: %v-%v-%v, build: %v", gitTag, gitBranch, gitCommit, buildTimestamp)
	}

	config, err := configuration.StartupConfiguration()
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(ExitErrConfig)
	}

	ctx := context.Background()

	// start in oneRun working mode
	if config.Script != nil {
		runner := runner.NewRunner(config.Script)
		runner.Run(ctx)
		if config.OutputJson {
			var d []byte
			if config.OutputPretty {
				d, _ = json.MarshalIndent(config.Script, "", "    ")
			} else {
				d, _ = json.Marshal(config.Script)
			}
			fmt.Println(string(d))
		}

		if runner.Result.Success {
			os.Exit(0)
		} else {
			os.Exit(ExitFailed)
		}
	}

	// daemon mode
	scheduler := gocron.NewScheduler(time.Local)
	scheduler.TagsUnique()
	scheduler.SingletonModeAll()

	for i, j := range config.ScheduleJobs {
		r := runner.NewRunner(j.Script)
		config.ScheduleJobs[i].CronJob, err = scheduler.Every(j.Schedule).Name(j.Name).DoWithJobDetails(func(r runner.Runner, job *gocron.Job) {
			log.Printf("[%v] starting the job\n", job.GetName())
			r.Run(job.Context())
			log.Printf("[%v] job has been finished\n", job.GetName())
		}, r)
	}

	os.Exit(0)
}

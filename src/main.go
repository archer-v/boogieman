package main

import (
	"boogieman/src/configuration"
	"boogieman/src/runner"
	"context"
	"encoding/json"
	"fmt"
	"os"
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

	config, script, err := configuration.StartupConfiguration()
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(ExitErrConfig)
	}

	ctx := context.Background()

	runner := runner.NewRunner(script)

	runner.Run(ctx)

	if config.OutputJson {
		var d []byte
		if config.OutputPretty {
			d, _ = json.MarshalIndent(script, "", "    ")
		} else {
			d, _ = json.Marshal(script)
		}
		fmt.Println(string(d))
	}

	if runner.Result.Success {
		os.Exit(0)
	} else {
		os.Exit(ExitFailed)
	}
}

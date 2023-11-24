package configuration

import (
	"boogieman/src/model"
	"boogieman/src/probeFactory"
	_ "boogieman/src/probes"
	"errors"
	"fmt"
	"github.com/integrii/flaggy"
	"github.com/vrischmann/envconfig"
	"os"
	"time"
)

var envPrefix = "PROBE"

type startupOptions struct {
	Script              string         `envconfig:"optional"`
	ProbeName           string         `envconfig:"optional"`
	ProbeConf           string         `envconfig:"optional"`
	ProbeOptionsTimeout *time.Duration `envconfig:"optional"`
	ProbeOptionsExpect  *bool          `envconfig:"optional"`
	Config              string         `envconfig:"optional"`
}

type StartupConfig struct {
	OneRun       bool
	OutputJson   bool
	OutputPretty bool
	HttpPort     int
	Script       *model.Script
	ScheduleJobs []ScheduleJob
}

func StartupConfiguration() (config StartupConfig, err error) {

	var o startupOptions
	//parse environment variables to the daemonConfig struct
	if err := envconfig.InitWithPrefix(&o, envPrefix); err != nil {
		panic(err)
	}

	flaggy.DefaultParser.ShowHelpOnUnexpected = true

	oneRun := flaggy.NewSubcommand("oneRun")
	oneRun.Description = "performs a single run, print result and exit"

	oneRun.String(&o.Script, "s", "script", "path to a checklist file in yml format")
	oneRun.String(&o.ProbeName, "n", "probename", "a probeFactory name (ignored if script option is selected)")
	oneRun.String(&o.ProbeConf, "o", "probeconf", "probeFactory daemonConfig data (ignored if script option is selected)")
	oneRun.Duration(o.ProbeOptionsTimeout, "t", "timeout", "operation waiting timeout (ignored if script option is selected)")
	oneRun.Bool(o.ProbeOptionsExpect, "e", "expect", "expected result true|false (ignored if script option is selected)")
	oneRun.Bool(&config.OutputJson, "j", "json", "output result in JSON format")
	oneRun.Bool(&config.OutputPretty, "J", "pretty", "pretty output with indent and CR")

	daemon := flaggy.NewSubcommand("daemon")
	daemon.Description = "start in daemon mode and performs scheduled jobs"
	daemon.String(&o.Config, "c", "config", "path to a configuration file in yml format")

	flaggy.AttachSubcommand(oneRun, 1)
	flaggy.AttachSubcommand(daemon, 1)

	flaggy.Parse()

	if oneRun.Used {
		config.OneRun = true
		if (o.Script == "" && o.ProbeName == "") || (o.Script != "" && o.ProbeName != "") {
			err = errors.New("either 'script' or 'probename' option should be defined")
			flaggy.ShowHelp(err.Error())
			return
		}

		if o.ProbeName != "" {
			var p model.Prober
			d := model.DefaultProbeOptions
			if o.ProbeOptionsExpect != nil {
				d.Expect = *o.ProbeOptionsExpect
			}
			if o.ProbeOptionsTimeout != nil {
				d.Timeout = *o.ProbeOptionsTimeout
			}
			p, err = probeFactory.NewProbe(o.ProbeName, d, o.ProbeConf)
			if err != nil {
				return
			}

			config.Script = &model.Script{}
			config.Script.AddTask(&model.Task{
				Probe: p,
			})
		}

		if o.Script != "" {
			var data []byte
			data, err = os.ReadFile(o.Script)
			if err != nil {
				return
			}
			config.Script, err = ScriptYMLConfiguration(data)
			if err != nil {
				err = fmt.Errorf("can't parse configuration from file: %v", err)
				return
			}
		}
		return
	}

	if daemon.Used {
		if o.Config == "" {
			err = errors.New("configuration file should be defined in a daemon mode")
			flaggy.ShowHelp(err.Error())
			return
		}
		var data []byte
		data, err = os.ReadFile(o.Config)
		if err != nil {
			return
		}

		daemonConfig, e := DaemonYMLConfiguration(data)
		if e != nil {
			err = fmt.Errorf("can't parse configuration from file: %v", e)
			return
		}
		config.HttpPort = daemonConfig.Global.HttpPort
		config.ScheduleJobs = daemonConfig.Jobs
	}
	return
}

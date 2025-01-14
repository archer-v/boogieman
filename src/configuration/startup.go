package configuration

import (
	"boogieman/src/model"
	"boogieman/src/probefactory"
	_ "boogieman/src/probes"
	"errors"
	"fmt"
	"github.com/integrii/flaggy"
	"github.com/vrischmann/envconfig"
	"os"
	"time"
)

var envPrefix = "BOOGIEMAN"
var AppDescriptionMessage = ""
var AppVersion = ""

type StartupMode int

const (
	StartupModeWrong  StartupMode = iota
	StartupModeOneRun StartupMode = iota
	StartupModeDaemon StartupMode = iota
)

type startupOptions struct {
	Script              string
	Probe               string
	ProbeConf           string
	ProbeOptionsTimeout time.Duration
	ProbeOptionsExpect  bool `envconfig:"default=true"`
	Debug               bool
	VerboseLog          bool
	//Config              string
}

type StartupConfig struct {
	Mode           StartupMode
	JSON           bool
	PrettyJSON     bool
	Script         *model.Script
	ScheduleJobs   []model.ScheduleJob
	ConfigFileName string
	GlobalOptions
}

//nolint:funlen
func StartupConfiguration() (config StartupConfig, err error) {
	var o startupOptions

	// parse environment variables
	if err := envconfig.InitWithOptions(&o, envconfig.Options{LeaveNil: true, AllOptional: true, Prefix: envPrefix}); err != nil {
		fmt.Printf("something got wrong with parsing env variables: %v", err)
	}

	flaggy.SetDescription(AppDescriptionMessage)
	flaggy.SetVersion(AppVersion)
	flaggy.DefaultParser.ShowHelpOnUnexpected = true

	oneRun := flaggy.NewSubcommand("oneRun")
	oneRun.Description = "performs a single run, print result and exit"

	oneRun.String(&o.Script, "s", "script", "path to a script file in yml format")
	oneRun.String(&o.Probe, "p", "probe", "single probe to start (ignored if script option is selected)")
	oneRun.String(&o.ProbeConf, "c", "config", "probe configuration string (ignored if script option is selected)")
	oneRun.Duration(&o.ProbeOptionsTimeout, "t", "timeout", "probe waiting timeout (ignored if script option is selected)")
	oneRun.Bool(&o.Debug, "d", "debug", "debug logging")
	oneRun.Bool(&o.VerboseLog, "v", "verbose", "verbose logging")
	oneRun.Bool(&o.ProbeOptionsExpect, "e", "expect", "expected result true|false (ignored if script option is selected)")
	oneRun.Bool(&config.JSON, "j", "json", "output result in JSON format")
	oneRun.Bool(&config.PrettyJSON, "J", "jsonp", "output result in JSON format with indents and CR")

	daemon := flaggy.NewSubcommand("daemon")
	daemon.Description = "start in daemon mode and performs scheduled jobs"
	daemon.String(&config.ConfigFileName, "c", "config", "path to a configuration file in yml format")

	flaggy.AttachSubcommand(oneRun, 1)
	flaggy.AttachSubcommand(daemon, 1)

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	flaggy.Parse()

	switch {
	case oneRun.Used:
		config.Mode = StartupModeOneRun
		if (o.Script == "" && o.Probe == "") || (o.Script != "" && o.Probe != "") {
			err = errors.New("either 'script' or 'probe' option should be defined")
			flaggy.ShowHelp("")
			return
		}
		if o.Probe != "" {
			var p model.Prober
			d := model.DefaultProbeOptions
			d.Expect = o.ProbeOptionsExpect
			d.Debug = o.Debug
			d.VerboseLogging = o.VerboseLog

			if o.ProbeOptionsTimeout != 0 {
				d.Timeout = o.ProbeOptionsTimeout
			}
			p, err = probefactory.NewProbe(o.Probe, d, o.ProbeConf)
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
				err = fmt.Errorf("can't parse configuration from file: %w", err)
				return
			}
		}
		return
	case daemon.Used:
		config.Mode = StartupModeDaemon
		if config.ConfigFileName == "" {
			err = errors.New("configuration file should be defined in a daemon mode")
			flaggy.ShowHelp(err.Error())
			return
		}
		var data []byte
		data, err = os.ReadFile(config.ConfigFileName)
		if err != nil {
			return
		}
		daemonConfig, e := DaemonYMLConfiguration(data)
		if e != nil {
			err = fmt.Errorf("can't parse configuration from file: %w", e)
			return
		}
		config.GlobalOptions = daemonConfig.Global
		config.ScheduleJobs = daemonConfig.Jobs
	default:
		flaggy.ShowHelp("")
		return
	}
	return
}

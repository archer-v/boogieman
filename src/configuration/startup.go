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
	Daemon                bool           `envconfig:"optional"`
	OneCheck              bool           `envconfig:"optional"`
	CheckList             string         `envconfig:"optional"`
	ProbeName             string         `envconfig:"optional"`
	ProbeConf             string         `envconfig:"optional"`
	ProbeOptionsTimeout   *time.Duration `envconfig:"optional"`
	ProbeOptionsStayAlive *bool          `envconfig:"optional"`
	ProbeOptionsExpect    *bool          `envconfig:"optional"`
}

type Config struct {
	Daemon       bool
	OutputJson   bool
	OutputPretty bool
	Prometheus   bool
}

func StartupConfiguration() (config Config, script model.Script, err error) {

	var o startupOptions
	//parse environment variables to the config struct
	if err := envconfig.InitWithPrefix(&o, envPrefix); err != nil {
		panic(err)
	}

	flaggy.DefaultParser.ShowHelpOnUnexpected = true

	//daemonOp := flaggy.NewSubcommand("daemon")
	//daemonOp.Description = "start in a daemon mode to perform regular checks"

	flaggy.String(&o.CheckList, "l", "checklist", "path to a checklist file in yml format")
	flaggy.String(&o.ProbeName, "n", "probename", "a probeFactory name (ignored if checklist option is selected)")
	flaggy.String(&o.ProbeConf, "o", "probeconf", "probeFactory config data (ignored if checklist option is selected)")
	flaggy.Duration(o.ProbeOptionsTimeout, "t", "timeout", "operation waiting timeout (ignored if checklist option is selected)")
	flaggy.Bool(o.ProbeOptionsExpect, "e", "expect", "expected result true|false (ignored if checklist option is selected)")
	flaggy.Bool(&config.OutputJson, "j", "json", "output result in JSON format")
	flaggy.Bool(&config.OutputPretty, "J", "pretty", "pretty output with indent and CR")

	//flaggy.AttachSubcommand(daemonOp, 1)

	flaggy.Parse()

	if (o.CheckList == "" && o.ProbeName == "") || (o.CheckList != "" && o.ProbeName != "") {
		err = errors.New("either 'checklist' or 'probename' option should be defined")
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
		if o.ProbeOptionsStayAlive != nil {
			d.StayAlive = *o.ProbeOptionsStayAlive
		}
		p, err = probeFactory.NewProbe(o.ProbeName, d, o.ProbeConf)
		if err != nil {
			return
		}

		script = model.NewScript()
		script.AddTask(&model.Task{
			Probe: p,
		})
	}

	if o.CheckList != "" {
		var data []byte
		data, err = os.ReadFile(o.CheckList)
		if err != nil {
			return
		}
		script, err = ymlConfiguration(data)
		if err != nil {
			err = fmt.Errorf("can't parse configuration from file: %v", err)
			return
		}
	}

	config.Daemon = o.Daemon

	return
}

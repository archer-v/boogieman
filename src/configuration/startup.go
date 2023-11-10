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

type startupConfig struct {
	Daemon                bool           `envconfig:"optional"`
	OneCheck              bool           `envconfig:"optional"`
	CheckList             string         `envconfig:"optional"`
	ProbeName             string         `envconfig:"optional"`
	ProbeConf             string         `envconfig:"optional"`
	ProbeOptionsTimeout   *time.Duration `envconfig:"optional"`
	ProbeOptionsStayAlive *bool          `envconfig:"optional"`
	ProbeOptionsExpect    *bool          `envconfig:"optional"`
}

func StartupConfiguration() (script model.Script, err error) {

	var c startupConfig
	//parse environment variables to the config struct
	if err := envconfig.InitWithPrefix(&c, envPrefix); err != nil {
		panic(err)
	}

	flaggy.DefaultParser.ShowHelpOnUnexpected = true

	daemonOp := flaggy.NewSubcommand("daemon")
	daemonOp.Description = "start in a daemon mode to perform regular checks"

	flaggy.String(&c.CheckList, "l", "checklist", "path to a checklist file in yml format")
	flaggy.String(&c.ProbeName, "n", "probename", "a probeFactory name (ignored if checklist option is selected)")
	flaggy.String(&c.ProbeConf, "c", "probeconf", "probeFactory config data (ignored if checklist option is selected)")
	flaggy.Duration(c.ProbeOptionsTimeout, "t", "timeout", "operation waiting timeout (ignored if checklist option is selected)")
	flaggy.Bool(c.ProbeOptionsExpect, "e", "expect", "expected result true|false (ignored if checklist option is selected)")
	//flaggy.Bool(&c.ProbeOptionsStayAlive, "a", "stayalive", "leave the probe related flow to stay running after main probe condition is finish (applicable only for some probes)")

	flaggy.AttachSubcommand(daemonOp, 1)

	flaggy.Parse()

	if (c.CheckList == "" && c.ProbeName == "") || (c.CheckList != "" && c.ProbeName != "") {
		err = errors.New("either 'checklist' or 'probename' option should be defined")
		flaggy.ShowHelp(err.Error())
		return
	}

	if c.ProbeName != "" {
		var p model.Prober
		o := model.DefaultProbeOptions
		if c.ProbeOptionsExpect != nil {
			o.Expect = *c.ProbeOptionsExpect
		}
		if c.ProbeOptionsTimeout != nil {
			o.Timeout = *c.ProbeOptionsTimeout
		}
		if c.ProbeOptionsStayAlive != nil {
			o.StayAlive = *c.ProbeOptionsStayAlive
		}
		p, err = probeFactory.NewProbe(c.ProbeName, o, c.ProbeConf)
		if err != nil {
			return
		}

		script = model.NewScript()
		script.AddTask(&model.Task{
			Probe: p,
		})
	}

	if c.CheckList != "" {
		var data []byte
		data, err = os.ReadFile(c.CheckList)
		if err != nil {
			return
		}
		script, err = ymlConfiguration(data)
		if err != nil {
			err = fmt.Errorf("can't parse configuration from file: %v", err)
			return
		}
	}

	script.Daemon = c.Daemon

	return
}

package configuration

import (
	"boogieman/src/model"
	"fmt"
	"github.com/creasty/defaults"
	"os"
	"sigs.k8s.io/yaml" // use it instead gopkg.in/yaml.v3 as it also supports json attributes in struct
	"time"
)

type GlobalOptions struct {
	DefaultSchedule    string `json:"default_schedule"`
	BindTo             string `json:"bind_to" default:"localhost:9091"`
	ExitOnConfigChange bool   `json:"exit_on_config_change" default:"false"`
}

type DaemonConfig struct {
	Global GlobalOptions
	Jobs   []model.ScheduleJob
}

func DaemonYMLConfiguration(data []byte) (config DaemonConfig, err error) {
	if err = defaults.Set(&config); err != nil {
		return
	}
	if err = yaml.Unmarshal(data, &config); err != nil {
		return
	}

	for i, j := range config.Jobs {
		var scriptData []byte
		scriptData, err = os.ReadFile(j.ScriptFile)
		if err != nil {
			return
		}
		// job custom variables is defined
		config.Jobs[i].Script, err = ScriptYMLConfiguration(scriptData, j.Vars)
		if err != nil {
			err = fmt.Errorf("can't parse configuration from %v: %w", j.ScriptFile, err)
			return
		}
		config.Jobs[i].Script.Timeout = time.Millisecond * j.Timeout
		if config.Jobs[i].Schedule == "" {
			config.Jobs[i].Schedule = config.Global.DefaultSchedule
		}
	}
	return
}

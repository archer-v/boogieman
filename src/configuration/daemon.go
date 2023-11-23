package configuration

import (
	"boogieman/src/model"
	"fmt"
	"github.com/creasty/defaults"
	"github.com/go-co-op/gocron"
	"os"
	"sigs.k8s.io/yaml" // use it instead gopkg.in/yaml.v3 as it also supports json attributes in struct
	"time"
)

type options struct {
	DefaultSchedule string
	HttpPort        int `default:"8091"`
}

type ScheduleJob struct {
	Name       string
	ScriptFile string `json:"script"`
	Schedule   string
	Timeout    time.Duration
	CronJob    *gocron.Job   `json:"-"`
	Script     *model.Script `json:"-"`
}

type daemonConfig struct {
	General options
	Jobs    []ScheduleJob
}

func DaemonYMLConfiguration(data []byte) (config daemonConfig, err error) {

	if err = defaults.Set(&config); err != nil {
		return
	}
	if err = yaml.Unmarshal(data, &config); err != nil {
		return
	}

	for _, j := range config.Jobs {
		var scriptData []byte
		scriptData, err = os.ReadFile(j.ScriptFile)
		if err != nil {
			return
		}
		j.Script, err = ScriptYMLConfiguration(scriptData)
		if err != nil {
			err = fmt.Errorf("can't parse configuration from %v: %w", j.ScriptFile, err)
			return
		}
		j.Script.Timeout = time.Millisecond * j.Timeout
		if j.Schedule == "" {
			j.Schedule = config.General.DefaultSchedule
		}
	}
	return
}

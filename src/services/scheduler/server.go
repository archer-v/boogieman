package scheduler

import (
	"boogieman/src/model"
	"boogieman/src/runner"
	"context"
	"errors"
	"github.com/go-co-op/gocron"
	"log"
	"os"
	"time"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

type Scheduler struct {
	*gocron.Scheduler
}

func Run() *Scheduler {
	scheduler := gocron.NewScheduler(time.Local)
	scheduler.TagsUnique()
	scheduler.SingletonModeAll()
	scheduler.StartAsync()
	logger.Println("Scheduler is started")
	return &Scheduler{scheduler}
}
func (s *Scheduler) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.Stop()
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-ctx.Done():
		log.Println("Scheduler finishing timeout")
	}
	return nil
}

func (s *Scheduler) AddJob(j *model.ScheduleJob) (err error) {
	if j.CronJob != nil {
		return errors.New("already added")
	}
	job, err := s.addJob(j.Name, j.Script, j.Schedule, j.Once)
	if err == nil {
		j.CronJob = job
	}
	return
}

func (s *Scheduler) addJob(name string, script *model.Script, schedule string, once bool) (cronJob *gocron.Job, err error) {
	var sj *gocron.Scheduler
	if _, e := time.ParseDuration(schedule); e == nil {
		sj = s.Every(schedule)
	} else {
		sj = s.CronWithSeconds(schedule).WaitForSchedule()
	}
	if once {
		sj = sj.LimitRunsTo(1)
	}
	r := runner.NewRunner(script)
	cronJob, err = sj.Name(name).DoWithJobDetails(func(r runner.Runner, job gocron.Job) {
		log.Printf("[%v] starting the job\n", job.GetName())
		r.Run(job.Context())
		log.Printf("[%v] job has been finished\n", job.GetName())
	}, r)

	return
}

package scheduler

import (
	"boogieman/src/model"
	"context"
	"errors"
	"fmt"
	"github.com/go-co-op/gocron"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)
var defScheduler = Scheduler{
	jobs:        make([]model.ScheduleJob, 0),
	urlPatterns: make(map[string]httpHandler),
}

const (
	httpPathPrefixJob  = "/job"
	httpPathPrefixJobs = "/jobs"
)

type Scheduler struct {
	*gocron.Scheduler
	jobs        []model.ScheduleJob
	urlPatterns map[string]httpHandler
	sync.Mutex
}

type httpHandler func(req *http.Request) (code int, jsonData []byte)

func Run() (s *Scheduler) {
	s = &defScheduler
	if s.Scheduler != nil {
		return
	}
	s.Scheduler = gocron.NewScheduler(time.Local)
	s.Scheduler.TagsUnique()
	s.Scheduler.SingletonModeAll()
	s.Scheduler.StartAsync()

	s.urlPatterns[httpPathPrefixJob] = s.httpJob
	s.urlPatterns[httpPathPrefixJobs] = s.httpJobs

	logger.Println("Scheduler is started")
	return
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

func (s *Scheduler) AddJob(j model.ScheduleJob) (err error) {
	if j.CronJob != nil {
		return errors.New("already added")
	}
	job, err := s.addCronJob(j.Name, j.Script, j.Schedule, j.Once)
	if err != nil {
		return
	}
	j.CronJob = job
	s.addJob(j)
	return
}

func (s *Scheduler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	resCode := http.StatusNotFound
	var jsonData []byte

	for k, v := range s.urlPatterns {
		if req.URL.Path == k {
			resCode, jsonData = v(req)
			break
		}
	}
	if resCode == http.StatusOK {
		res.Header().Set("Content-Type", "application/json")
	}
	res.WriteHeader(resCode)
	if resCode == http.StatusOK {
		_, _ = res.Write(jsonData)
	} else {
		_, _ = fmt.Fprint(res, http.StatusText(resCode))
	}
}

func (s *Scheduler) URLPatters() (p []string) {
	for key := range s.urlPatterns {
		p = append(p, key)
	}
	return
}

func (s *Scheduler) addCronJob(name string, script *model.Script, schedule string, once bool) (cronJob *gocron.Job, err error) {
	var sj *gocron.Scheduler
	if _, e := time.ParseDuration(schedule); e == nil {
		sj = s.Every(schedule)
	} else {
		sj = s.CronWithSeconds(schedule).WaitForSchedule()
	}
	if once {
		sj = sj.LimitRunsTo(1)
	}

	cronJob, err = sj.Name(name).DoWithJobDetails(func(s *model.Script, job gocron.Job) {
		log.Printf("[%v] starting the job\n", job.GetName())
		s.Run(job.Context())
		log.Printf("[%v] job has been finished\n", job.GetName())
	}, script)

	return
}

func (s *Scheduler) addJob(j model.ScheduleJob) {
	s.Lock()
	s.jobs = append(s.jobs, j)
	s.Unlock()
}

func (s *Scheduler) delJob(name string) {
	s.Lock()
	defer s.Unlock()
	idx := -1
	for i, j := range s.jobs {
		if j.Name == name {
			idx = i
			s.Scheduler.RemoveByReference(j.CronJob)
			break
		}
	}
	if idx < 0 {
		return
	}
	s.jobs = append(s.jobs[:idx], s.jobs[idx+1:]...)
}

func (s *Scheduler) getJob(name string) (j model.ScheduleJob, err error) {
	s.Lock()
	defer s.Unlock()
	for _, j := range s.jobs {
		if j.Name == name {
			return j, nil
		}
	}
	err = errors.New("not found")
	return
}

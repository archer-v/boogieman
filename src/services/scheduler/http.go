package scheduler

import (
	"boogieman/src/model"
	"encoding/json"
	"net/http"
)

func (s *Scheduler) httpJob(req *http.Request) (code int, jsonData []byte) {
	code = http.StatusOK
	jobName := req.URL.Query().Get("name")
	if jobName == "" {
		code = http.StatusBadRequest
		return
	}

	j, err := s.getJob(jobName)
	if err != nil {
		code = http.StatusNotFound
		return
	}
	jsonData, err = json.Marshal(j.Script.ResultFinished())
	if err != nil {
		code = http.StatusInternalServerError
		logger.Printf("httpJob: can't create json response: %v\n", err)
	}
	return
}

func (s *Scheduler) httpJobs(*http.Request) (code int, jsonData []byte) {
	code = http.StatusOK

	jobs := make([]model.ScheduleJob, 0)
	for _, j := range s.jobs {
		if j.CronJob != nil {
			j.NextStartAt = j.CronJob.NextRun()
		}
		jobs = append(jobs, j)
	}
	jsonData, err := json.Marshal(jobs)
	if err != nil {
		code = http.StatusInternalServerError
		logger.Printf("httpJobs: can't create json response: %v\n", err)
	}
	return
}

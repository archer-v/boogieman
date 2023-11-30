package scheduler

import (
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

func (s *Scheduler) httpJobs(req *http.Request) (code int, jsonData []byte) {
	code = http.StatusOK

	jsonData, err := json.Marshal(s.jobs)
	if err != nil {
		code = http.StatusInternalServerError
		logger.Printf("httpJobs: can't create json response: %v\n", err)
	}
	return
}

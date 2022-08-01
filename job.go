package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type JobResult struct {
	Code     int
	Duration time.Duration
	Size     int
	Err      error
}

type Job struct {
	Url      string
	Interval time.Duration
	Result   chan JobResult
	Quit     chan bool
	Done     chan bool
}

var (
	intervalRegexp = regexp.MustCompile(`#(\d+)$`)
)

func NewJob(url string, defaultInterval time.Duration) *Job {
	job := new(Job)
	job.Url = url
	job.Interval = defaultInterval
	intervalStr := intervalRegexp.FindString(url)
	if intervalStr != "" {
		job.Url = url[0 : len(url)-len(intervalStr)]
		intervalStr = intervalStr[1:]
		intervalNum, _ := strconv.ParseInt(intervalStr, 10, 64)
		if intervalNum >= 5 {
			job.Interval = time.Duration(intervalNum) * time.Second
		}
	}
	job.Result = make(chan JobResult)
	job.Quit = make(chan bool)
	job.Done = make(chan bool)
	return job
}

func (job *Job) Loop() {
	defer close(job.Done)
	log.Printf("Begin Loop For Job: %s, Interval: %f\n", job.Url, job.Interval.Seconds())
	t := time.NewTicker(job.Interval)
	defer t.Stop()
loop:
	for {
		select {
		case <-job.Quit:
			break loop
		case <-t.C:
			t.Reset(job.Interval)
			job.Result <- job.Run()
		}
	}
	log.Printf("Stop Loop For Job: %s\n", job.Url)
}

type ioReadResult struct {
	body []byte
	err  error
}

func (job *Job) Run() JobResult {
	var result JobResult

	log.Printf("Start Test Url: %s\n", job.Url)
	client := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, _ := http.NewRequest("GET", job.Url, nil)
	req.Header.Set("Cache-Control", "no-cache")
	startTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.Err = err
		log.Printf("client.Do: %v\n", err)
		return result
	}

	result.Code = resp.StatusCode

	resultChan := make(chan ioReadResult)
	go func(resp *http.Response, resultChan chan<- ioReadResult) {
		defer resp.Body.Close()
		defer close(resultChan)
		body, err := io.ReadAll(resp.Body)
		resultChan <- ioReadResult{body: body, err: err}
	}(resp, resultChan)

	select {
	case <-job.Quit:
		log.Printf("Abort")
		result.Err = errors.New("Abort")
		return result
	case respResult := <-resultChan:
		if respResult.err == nil {
			result.Duration = time.Since(startTime)
			result.Size = len(respResult.body)
		} else {
			log.Printf("io.ReadAll: %v\n", respResult.err)
			result.Err = respResult.err
			return result
		}
	}
	log.Printf("Finish Test Url: %s\n  Use Time: %f, Download %d Bytes", job.Url, result.Duration.Seconds(), result.Size)
	return result
}

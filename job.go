package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type Job struct {
	Url      string
	Interval time.Duration
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
	return job
}

type JobResult struct {
	Code     int
	Duration time.Duration
	Size     int
	Err      error
}

func (job *Job) Loop(result chan JobResult, quit chan bool) {
	go func() {
		log.Printf("Begin Loop For Job: %s\n", job.Url)
	loop:
		for {
			select {
			case <-quit:
				break loop
			default:
				result <- job.Run(quit)
				select {
				case <-quit:
					break loop
				case <-time.After(job.Interval):

				}
			}
		}
		log.Printf("Stop Loop For Job: %s\n", job.Url)
	}()
}

func (job *Job) Run(quit chan bool) JobResult {
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
	dataChannel := make(chan []byte)
	go func() {
		defer resp.Body.Close()
		body, err2 := ioutil.ReadAll(resp.Body)
		if err2 == nil {
			dataChannel <- body
		} else {
			log.Printf("ioutil.ReadAll: %v\n", err2)
		}
		close(dataChannel)
	}()
	select {
	case <-quit:
		log.Printf("Abort")
		result.Err = errors.New("Abort")
		return result
	case body := <-dataChannel:
		result.Duration = time.Since(startTime)
		result.Size = len(body)
	}

	if result.Size <= 0 {
		result.Err = errors.New("Empty")
		return result
	}
	log.Printf("Finish Test Url: %s\n  Use Time: %f, Download %d Bytes", job.Url, result.Duration.Seconds(), result.Size)
	return result
}

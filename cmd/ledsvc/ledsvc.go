package main

import (
	"time"
	"log"
	"net/http"
	
	"github.com/robfig/cron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"com.luksamuk.ledcontrol/wsclient"
)

var (
	mtReqsPerformed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledsvc_requests_performed",
		Help: "Total number of requests performed by the service",
	})

	mtSuccessfulRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledsvc_successful_requests",
		Help: "Total number of successful requests",
	})

	mtErroredRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledsvc_error_requests",
		Help: "Total number of requests resulting in errors",
	})

	mtRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "ledsvc_requests_duration",
		Help: "A histogram for request duration",
		Buckets: prometheus.LinearBuckets(.05, .05, 100),
	})
)


func main() {
	log.Print("Starting scheduler")
	
	c := cron.New()

	c.AddFunc("@every 1m", func() {
		log.Print("Running cronjob")

		start := time.Now()
		status, err := wsclient.GetStatus()
		elapsed := time.Since(start)
		duration := elapsed.Seconds()
		
		mtReqsPerformed.Inc()
		mtRequestDuration.Observe(duration)
		
		if err != nil {
			mtErroredRequests.Inc()
		} else {
			mtSuccessfulRequests.Inc()
			log.Print("Status: ", status)
			log.Print("Duration: ", duration)
		}
	})

	c.Start()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}

package main

import (
	"time"
	"log"
	"net/http"
	
	"github.com/robfig/cron"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	client "com.luksamuk.ledcontrol/brokerclient"
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
		Buckets: prometheus.LinearBuckets(0.1, .1, 15),
	})

	mtLedStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_status",
		Help: "A gauge for LED status",
	})

	mtLedProgram = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_program",
		Help: "A gauge for current LED program",
	})

	mtDimmer = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_dim",
		Help: "A gauge for LED intensity",
	})
)


func main() {
	log.Print("Conectando ao Broker MQTT...")
	if err := client.Init("com.luksamuk.ledcontrol/ledsvc"); err != nil {
		log.Fatalf("Não foi possível conectar ao broker: %v", err)
	}
	time.Sleep(time.Second)
	
	log.Print("Preparando coleta de métricas...")
	
	log.Print("Preparando scheduler...")
	
	c := cron.New()

	mtLedProgram.Set(-1)
	mtDimmer.Set(0)

	c.AddFunc("@every 10s", func() {
		log.Print("Verificando status do dispositivo")

		start := time.Now()
		status := client.GetStatus()
		elapsed := time.Since(start)
		duration := elapsed.Seconds()
		
		mtReqsPerformed.Inc()
		mtRequestDuration.Observe(duration)
		
		mtSuccessfulRequests.Inc()

		active := 1.0
		dim := status.Dim
		program := status.Program
		if !status.Blinking {
			active = 0.0
			dim = 0.0
			program = -1
		}
		mtLedStatus.Set(active)

		
		mtDimmer.Set(dim)
		mtLedProgram.Set(float64(program))
	})

	log.Print("Iniciando scheduler...")
	c.Start()

	log.Print("Iniciando servidor de métricas na porta :2112.")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}

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
	"github.com/crazy3lf/colorconv"
)

var (

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

	mtColorRed = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_rgb_red",
		Help: "Intensity of red on current color on the [0.0, 1.0] range",
	})

	mtColorGreen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_rgb_green",
		Help: "Intensity of green on current color on the [0.0, 1.0] range",
	})

	mtColorBlue = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_rgb_blue",
		Help: "Intensity of blue on current color on the [0.0, 1.0] range",
	})

	mtColorHue = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_hsv_hue",
		Help: "Hue of current color on the [0.0, 1.0] range",
	})
	
	mtColorSaturation = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_hsv_saturation",
		Help: "Saturation of current color on the [0.0, 1.0] range",
	})

	mtColorBrightness = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledsvc_hsv_brightness",
		Help: "Brightness (value) of current color on the [0.0, 1.0] range",
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
		log.Print("Atualizando métricas")

		status := client.GetStatus()

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

		if !status.Blinking || (status.Program != 2) {
			mtColorRed.Set(0.0)
			mtColorGreen.Set(0.0)
			mtColorBlue.Set(0.0)
		} else {
			r, g, b, _ := status.Color.RGBA()
			h, s, v := colorconv.ColorToHSV(status.Color)
			
			mtColorRed.Set(float64(r) / 65535.0)
			mtColorGreen.Set(float64(g) / 65535.0)
			mtColorBlue.Set(float64(b) / 65535.0)

			mtColorHue.Set(h / 360.0)
			mtColorSaturation.Set(s)
			mtColorBrightness.Set(v)
		}
	})

	log.Print("Iniciando scheduler...")
	c.Start()

	log.Print("Iniciando servidor de métricas na porta :2112.")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}


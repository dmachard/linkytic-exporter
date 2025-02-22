package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dmachard/go-ticreader"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Define Prometheus metrics for historical mode
	pappMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_historique_papp",
		Help: "Puissance apparente en VA",
	})
	iinstMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_historique_iinst",
		Help: "Intensité Instantanée en A",
	})
	baseMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_historique_base",
		Help: "Index option Base en Wh",
	})

	// Define Prometheus metrics for standard mode
	vticMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_standard_vtic",
		Help: "Version de la TIC",
	})
	sinstsMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_standard_sinsts",
		Help: "Puissance app. Instantanée soutirée en VA",
	})
	eastMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_standard_east",
		Help: "Energie active soutirée totale en Wh",
	})
)

func init() {
	// Register the Prometheus metrics
	prometheus.MustRegister(pappMetric)
	prometheus.MustRegister(iinstMetric)
	prometheus.MustRegister(baseMetric)
	// Register the Prometheus metrics
	prometheus.MustRegister(vticMetric)
	prometheus.MustRegister(sinstsMetric)
	prometheus.MustRegister(eastMetric)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func main() {
	// Read environment variables
	port := getEnvOrDefault("LINKY_TIC_DEVICE", "/dev/ttyACM0")
	modeStr := getEnvOrDefault("LINKY_TIC_MODE", "STANDARD")
	debugMode := getEnvOrDefault("LINKY_DEBUG", "false")

	// debug mode
	debug := strings.ToLower(debugMode) == "true"

	// Convert LINKY_MODE to ticreader mode
	var mode ticreader.LinkyMode
	switch strings.ToUpper(modeStr) {
	case "STANDARD":
		mode = ticreader.ModeStandard
	case "HISTORICAL":
		mode = ticreader.ModeHistorical
	default:
		log.Fatalf("Invalid LINKY_MODE: %s (expected 'HISTORICAL' or 'STANDARD')", modeStr)
	}

	if debug {
		log.Println("DEBUG MODE: ACTIVATED")
	}

	// Start reading TIC data
	log.Printf("Starting TIC reader on %s with mode %s", port, modeStr)
	frameChan, err := ticreader.StartReading(port, mode)
	if err != nil {
		log.Fatalf("Error initializing TIC reader: %v", err)
	}

	// Goroutine to continuously update metrics
	go func() {
		for teleinfo := range frameChan {
			if debug {
				log.Printf("DEBUG: Received TIC Frame: %s Len Dataset: %d", teleinfo.Timestamp, len(teleinfo.Dataset))
			}
			for _, info := range teleinfo.Dataset {
				if debug {
					log.Printf("DEBUG: Dataset - Label: %s, Value: %s, Valid: %t", info.Label, info.Data, info.Valid)
				}

				var value float64
				if _, err := fmt.Sscanf(info.Data, "%f", &value); err != nil {
					log.Printf("ERROR: Failed to parse value for %s: %v", info.Label, err)
					continue
				}

				switch info.Label {
				case "PAPP":
					pappMetric.Set(value)
				case "IINST":
					iinstMetric.Set(value)
				case "BASE":
					baseMetric.Set(value)
				case "VTIC":
					vticMetric.Set(value)
				case "EAST":
					eastMetric.Set(value)
				case "SINSTS":
					sinstsMetric.Set(value)
				}
			}
		}
	}()

	// Expose metrics on /metrics
	http.Handle("/metrics", promhttp.Handler())

	// Start the HTTP server for Prometheus
	portHTTP := "9100"
	log.Printf("Exporter running at: http://localhost:%s/metrics", portHTTP)
	err = http.ListenAndServe(":"+portHTTP, nil)
	if err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

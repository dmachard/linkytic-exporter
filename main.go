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

	// Start reading TIC data
	log.Printf("Starting TIC reader on %s with mode %s", port, modeStr)
	frameChan, err := ticreader.StartReading(port, mode)
	if err != nil {
		log.Fatalf("Error initializing TIC reader: %v", err)
	}

	// Goroutine to continuously update metrics
	go func() {
		for teleinfo := range frameChan {
			for _, info := range teleinfo.Dataset {
				if info.Label == "PAPP" && info.Valid {
					// Convert the value to float
					var value float64
					fmt.Sscanf(info.Data, "%f", &value)
					pappMetric.Set(value)
				}
				if info.Label == "IINST" && info.Valid {
					// Convert the value to float
					var value float64
					fmt.Sscanf(info.Data, "%f", &value)
					iinstMetric.Set(value)
				}
				if info.Label == "BASE" && info.Valid {
					// Convert the value to float
					var value float64
					fmt.Sscanf(info.Data, "%f", &value)
					baseMetric.Set(value)
				}

				if info.Label == "VTIC" && info.Valid {
					// Convert the value to float
					var value float64
					fmt.Sscanf(info.Data, "%f", &value)
					vticMetric.Set(value)
				}
				if info.Label == "EAST" && info.Valid {
					// Convert the value to float
					var value float64
					fmt.Sscanf(info.Data, "%f", &value)
					eastMetric.Set(value)
				}
				if info.Label == "SINSTS" && info.Valid {
					// Convert the value to float
					var value float64
					fmt.Sscanf(info.Data, "%f", &value)
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

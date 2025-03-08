package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dmachard/go-ticreader"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const stateFile = "/tmp/linky_state.json"

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
	eastDayMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "linky_tic_standard_east_day",
		Help: "Energie active soutirée par jour en kWh",
	}, []string{"date"})

	irms1Metric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_standard_irms1",
		Help: "Courant efficace, phase 1 en A",
	})

	urms1Metric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_standard_urms1",
		Help: "Tension efficace, phase 1 en V",
	})

	prefMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "linky_tic_standard_pref",
		Help: "Puissance app. de référence en kVA",
	})

	lastEastDailyValue float64 = -1
	lastResetDate      string  = time.Now().Format("2006-01-02")
)

type State struct {
	LastEastDailyValue float64 `json:"lastEastDailyValue"`
	LastResetDate      string  `json:"lastResetDate"`
}

func init() {
	// Register the Prometheus metrics
	prometheus.MustRegister(pappMetric)
	prometheus.MustRegister(iinstMetric)
	prometheus.MustRegister(baseMetric)
	// Register the Prometheus metrics
	prometheus.MustRegister(vticMetric)
	prometheus.MustRegister(sinstsMetric)
	prometheus.MustRegister(eastMetric)
	prometheus.MustRegister(eastDayMetric)
	prometheus.MustRegister(irms1Metric)
	prometheus.MustRegister(urms1Metric)
	prometheus.MustRegister(prefMetric)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && c != '.' {
			return false
		}
	}
	return true
}

func saveState() {
	state := State{
		LastEastDailyValue: lastEastDailyValue,
		LastResetDate:      lastResetDate,
	}

	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("ERROR: unable to save the state: %v", err)
		return
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		log.Printf("ERROR: Unable to write file: %v", err)
	}
}

func loadState() {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		log.Printf("No state, start from zero")
		return
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		log.Printf("ERROR: unable to load state: %v", err)
		return
	}

	lastEastDailyValue = state.LastEastDailyValue
	lastResetDate = state.LastResetDate
	log.Printf("INFO: Loading state (lastEastDailyValue=%.2f, lastResetDate=%s)", lastEastDailyValue, lastResetDate)
}

func updateDailyMetric(currentValue float64) {
	now := time.Now()
	currentDate := now.Format("2006-01-02")

	// first start
	if lastEastDailyValue == -1 {
		log.Printf("First start, init value: %.2f Wh", currentValue)
		lastEastDailyValue = currentValue
		lastResetDate = currentDate
		eastDayMetric.WithLabelValues(currentDate).Set(0)
		saveState()
		return
	}

	// Reset on new day
	if currentDate != lastResetDate {
		log.Printf("Daily reset (New day: %s)", currentDate)
		lastResetDate = currentDate
		lastEastDailyValue = currentValue
		eastDayMetric.WithLabelValues(currentDate).Set(0) // Reset
		saveState()
	}

	// Daily sum (in kWh)
	if currentValue >= lastEastDailyValue {
		consumption := (currentValue - lastEastDailyValue)
		eastDayMetric.WithLabelValues(currentDate).Set(consumption)
		saveState()
	}
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

	// load previous value
	loadState()

	// Goroutine to continuously update metrics
	go func() {
		for {
			// Start reading TIC data
			log.Printf("Starting TIC reader on %s with mode %s", port, modeStr)
			frameChan, err := ticreader.StartReading(port, mode)
			if err != nil {
				log.Printf("Error initializing TIC reader: %v", err)
				log.Println("Retrying in 5 seconds...")
				time.Sleep(5 * time.Second) // Attendre avant de réessayer
				continue
			}

			for teleinfo := range frameChan {
				if debug {
					log.Printf("")
					log.Printf("DEBUG: Received TIC Frame: %s Len Dataset: %d", teleinfo.Timestamp, len(teleinfo.Dataset))
				}
				for _, info := range teleinfo.Dataset {
					if debug {
						log.Printf("DEBUG: Dataset - Label: %s, Horodate: %s, Value: %s, Valid: %t", info.Label, info.Horodate, info.Data, info.Valid)
					}

					if !info.Valid {
						log.Printf("ERROR: Skipping invalid TIC data - Label: %s - Checksum invalid", info.Label)
						continue
					}

					if isNumeric(info.Data) {
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
							updateDailyMetric(value)
						case "SINSTS":
							sinstsMetric.Set(value)
						case "PREF":
							prefMetric.Set(value)
						case "URMS1":
							urms1Metric.Set(value)
						case "IRMS1":
							irms1Metric.Set(value)
						}
					}
				}
			}

			log.Println("TIC reader stopped. Retrying connection in 5 seconds...")
			time.Sleep(5 * time.Second)
		}
	}()

	// Expose metrics on /metrics
	http.Handle("/metrics", promhttp.Handler())

	// Start the HTTP server for Prometheus
	portHTTP := "9100"
	log.Printf("Exporter running at: http://localhost:%s/metrics", portHTTP)
	err := http.ListenAndServe(":"+portHTTP, nil)
	if err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}
}

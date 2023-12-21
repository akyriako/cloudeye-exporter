package main

import (
	"flag"
	"fmt"
	"github.com/akyriako/cloudeye-exporter/collector"
	"github.com/akyriako/cloudeye-exporter/config"
	"github.com/akyriako/cloudeye-exporter/handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

var (
	cloudConfigFlag  = flag.String("config", "./clouds.yaml", "Path to the cloud configuration file")
	filterEnableFlag = flag.Bool("filter-enable", false, "Enabling monitoring metric filter")
	debugFlag        = flag.Bool("debug", false, "If debug the code.")

	logger *slog.Logger
)

func main() {
	flag.Parse()

	initializeLogger()
	cloudConfig, err := config.GetConfigFromFile(*cloudConfigFlag, *filterEnableFlag)
	if err != nil {
		slog.Error(fmt.Sprintf("Parsing cloud config failed: %s", err.Error()))
		return
	}

	http.HandleFunc(cloudConfig.Global.MetricsPath, handlers.Metrics(*cloudConfigFlag))
	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/ping", handlers.Health)
	http.HandleFunc("/", handlers.Welcome(cloudConfig.Global.MetricsPath))

	slog.Info(fmt.Sprintf("Start server at port%s", cloudConfig.Global.Port))
	if err := http.ListenAndServe(cloudConfig.Global.Port, nil); err != nil {
		slog.Error(fmt.Sprintf("Error occur when start server %s", err.Error()))
		os.Exit(1)
	}
}

func initializeLogger() {
	levelInfo := slog.LevelInfo
	if *debugFlag {
		levelInfo = slog.LevelDebug
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelInfo,
	}))

	slog.SetDefault(logger)
}

func metrics(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("services")
	if target == "" {
		http.Error(w, "'target' parameter must be specified", http.StatusBadRequest)
		return
	}

	targets := strings.Split(target, ",")
	registry := prometheus.NewRegistry()

	slog.Info(fmt.Sprintf("Start to monitor services: %s", targets))
	exporter, err := collector.GetMonitoringCollector(*cloudConfigFlag, targets)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			slog.Error(fmt.Sprintf("Fail to write response body, error: %s", err.Error()))
			return
		}
		return
	}
	registry.MustRegister(exporter)
	if err != nil {
		slog.Error(fmt.Sprintf("Fail to start to morning services: %+v, err: %s", targets, err.Error()))
		return
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

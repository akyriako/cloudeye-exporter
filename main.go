package main

import (
	"flag"
	"fmt"
	"github.com/akyriako/cloudeye-exporter/config"
	"github.com/akyriako/cloudeye-exporter/handlers"
	"log/slog"
	"net/http"
	"os"
)

var (
	cloudConfigFlag  = flag.String("config", "./clouds.yaml", "path to the cloud configuration file")
	enableFilterFlag = flag.Bool("enable-filters", false, "enabling monitoring metric filter")
	debugFlag        = flag.Bool("debug", false, "debug mode")

	logger *slog.Logger
)

func main() {
	flag.Parse()

	initializeLogger()
	cloudConfig, err := config.GetConfigFromFile(*cloudConfigFlag, *enableFilterFlag)
	if err != nil {
		slog.Error(fmt.Sprintf("parsing cloud config failed: %s", err.Error()))
		return
	}

	http.HandleFunc(cloudConfig.Global.MetricsPath, handlers.Metrics(cloudConfig))
	http.HandleFunc("/health", handlers.Health)
	http.HandleFunc("/ping", handlers.Health)
	http.HandleFunc("/", handlers.Welcome(cloudConfig.Global.MetricsPath))

	slog.Info(fmt.Sprintf("cloudeye exporter listening at 0.0.0.0%s%s", cloudConfig.Global.Port, cloudConfig.Global.MetricsPath))
	if err := http.ListenAndServe(cloudConfig.Global.Port, nil); err != nil {
		slog.Error(fmt.Sprintf("error occur when start server %s", err.Error()))
		os.Exit(1)
	}
}

func initializeLogger() {
	levelInfo := slog.LevelInfo
	if *debugFlag {
		levelInfo = slog.LevelDebug
	}

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelInfo,
	}))

	slog.SetDefault(logger)
}

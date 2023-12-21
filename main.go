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

	http.HandleFunc(cloudConfig.Global.MetricsPath, handlers.Metrics(cloudConfig))
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

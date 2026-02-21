package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sergeyshevch/statuspage-exporter/pkg/config"
	"github.com/sergeyshevch/statuspage-exporter/pkg/prober"
	"go.uber.org/zap"
)

const (
	shutdownTimeout   = 5 * time.Second
	readHeaderTimeout = 10 * time.Second
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGKILL,
		syscall.SIGTERM,
	)
	defer cancel()

	log, err := config.InitConfig()
	if err != nil {
		log.Error("Unable to initialize config", zap.Error(err))
		return
	}

	prometheus.MustRegister(collectors.NewBuildInfoCollector())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /probe", prober.Handler(log))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle(`GET /metrics`, promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			Registry: prometheus.DefaultRegisterer,
		}))

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.HTTPPort()),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Start your http server for prometheus.
	go func() {
		log.Info("Http server listening on", zap.String("addr", srv.Addr))

		serverErr := srv.ListenAndServe()
		if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
			log.Panic("Unable to start a http server.", zap.Error(serverErr))
		}
	}()

	<-ctx.Done()
	log.Info("Received shutdown signal. Waiting for workers to terminate...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		log.Panic("Http server Shutdown Failed", zap.Error(err))
	}

	log.Info("Http server stopped")
}

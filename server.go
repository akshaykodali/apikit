package apikit

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

func CreateServer(ctx context.Context, wg *sync.WaitGroup, host, port string, handler http.Handler, logCh chan Log) {
	server := &http.Server{
		Addr:    net.JoinHostPort(host, port),
		Handler: handler,
	}

	wg.Add(1)
	go func() {
		slog.Info(
			"starting http server",
			"addr", net.JoinHostPort(host, port),
		)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error(
				"unable to start server",
				"err", err,
			)
		}
	}()

	go func() {
		defer wg.Done()

		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error(
				"unable to stop server",
				"err", err,
			)
		}

		close(logCh)
	}()
}

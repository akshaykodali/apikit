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
		slog.Info("starting http server")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("error in ListenAndServe", "err", err)
		}
	}()

	go func() {
		defer wg.Done()

		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("error in Shutdown", "err", err)
		}

		close(logCh)
	}()
}

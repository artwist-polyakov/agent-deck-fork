package logging

import (
	"log/slog"
	"net/http"
	_ "net/http/pprof" // Register pprof handlers
)

// startPprof starts a pprof HTTP server on localhost:6060.
// Only called when PprofEnabled is true in config.
func startPprof() {
	go func() {
		addr := "localhost:6060"
		Logger().Info("pprof_server_start", slog.String("addr", addr))
		if err := http.ListenAndServe(addr, nil); err != nil {
			Logger().Error("pprof_server_error", slog.String("error", err.Error()))
		}
	}()
}

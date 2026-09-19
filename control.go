package slogx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// ParseLevel converts common level strings to slog.Level.
// Accepts: trace, debug, info, warn/warning, error, fatal (case-insensitive),
// or a numeric slog.Level value.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	case "fatal":
		return LevelFatal, nil
	default:
		if n, err := strconv.Atoi(s); err == nil {
			return slog.Level(n), nil
		}
		return 0, fmt.Errorf("slogx: unknown level %q", s)
	}
}

// SetLevelString parses and applies a level string.
func (l *Logger) SetLevelString(s string) error {
	lvl, err := ParseLevel(s)
	if err != nil {
		return err
	}
	l.SetLevel(lvl)
	return nil
}

// LevelHTTPHandler serves runtime level get/set for operators.
//
//	GET  /level        → {"level":"INFO","level_int":0}
//	PUT  /level?level=debug  or JSON {"level":"debug"}
//
// Bind behind an internal listener only (not public ingress).
func (l *Logger) LevelHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			cur := l.Config().Level
			_, _ = fmt.Fprintf(w, `{"level":%q,"level_int":%d}`+"\n", getLevelName(cur, l.Config().LevelNames), int(cur))
		case http.MethodPut, http.MethodPost, http.MethodPatch:
			lvlStr := r.URL.Query().Get("level")
			if lvlStr == "" {
				var body struct {
					Level string `json:"level"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				lvlStr = body.Level
			}
			if err := l.SetLevelString(lvlStr); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			cur := l.Config().Level
			_, _ = fmt.Fprintf(w, `{"ok":true,"level":%q,"level_int":%d}`+"\n", getLevelName(cur, l.Config().LevelNames), int(cur))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// ListenLevelHTTP starts a dedicated HTTP server for LevelHTTPHandler on addr.
// Empty addr defaults to 127.0.0.1:0 (ephemeral port). Returns the bound address
// and a shutdown function. The server also shuts down when ctx is cancelled.
func (l *Logger) ListenLevelHTTP(ctx context.Context, addr string) (actualAddr string, shutdown func(context.Context) error, err error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	mux := http.NewServeMux()
	mux.Handle("/", l.LevelHTTPHandler())
	mux.Handle("/level", l.LevelHTTPHandler())
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}
	actualAddr = ln.Addr().String()
	go func() { _ = srv.Serve(ln) }()
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()
	return actualAddr, srv.Shutdown, nil
}

// WatchLevelEnv polls an environment variable and applies level changes.
// Stops when ctx is cancelled. Empty envKey defaults to LOG_LEVEL;
// empty interval defaults to 2s.
func (l *Logger) WatchLevelEnv(ctx context.Context, envKey string, interval time.Duration) {
	if envKey == "" {
		envKey = "LOG_LEVEL"
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	var last string
	apply := func() {
		v := strings.TrimSpace(os.Getenv(envKey))
		if v == "" || v == last {
			return
		}
		if err := l.SetLevelString(v); err == nil {
			last = v
		}
	}
	apply()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			apply()
		}
	}
}

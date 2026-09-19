package slogx

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	lvl, err := ParseLevel("debug")
	require.NoError(t, err)
	assert.Equal(t, slog.LevelDebug, lvl)
	_, err = ParseLevel("nope")
	assert.Error(t, err)

	for _, s := range []string{"trace", "info", "warn", "warning", "error", "fatal", "0"} {
		_, err := ParseLevel(s)
		require.NoError(t, err, s)
	}
}

func TestSetLevelStringAndHTTP(t *testing.T) {
	l := New(WithOutput(io.Discard), WithLevel(slog.LevelInfo))
	require.NoError(t, l.SetLevelString("debug"))
	assert.Equal(t, slog.LevelDebug, l.Level())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addr, shutdown, err := l.ListenLevelHTTP(ctx, "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = shutdown(context.Background()) }()

	res, err := http.Get("http://" + addr + "/level")
	require.NoError(t, err)
	defer res.Body.Close()
	var got map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	assert.Equal(t, "DEBUG", got["level"])

	req, err := http.NewRequest(http.MethodPut, "http://"+addr+"/level?level=warn", nil)
	require.NoError(t, err)
	res2, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res2.Body.Close()
	assert.Equal(t, http.StatusOK, res2.StatusCode)
	assert.Equal(t, slog.LevelWarn, l.Level())
}

func TestWatchLevelEnv(t *testing.T) {
	l := New(WithOutput(io.Discard), WithLevel(slog.LevelInfo))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Setenv("SLOGX_TEST_LEVEL", "error")
	go l.WatchLevelEnv(ctx, "SLOGX_TEST_LEVEL", 50*time.Millisecond)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if l.Level() == slog.LevelError {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("level not updated: %v", l.Level())
}

func TestCorporateMaskingOption(t *testing.T) {
	l := New(WithCorporateMasking(), WithOutput(io.Discard))
	_, ok := l.Config().Masker.(*CorporateMasker)
	assert.True(t, ok)
	_, ok = l.Config().MaskKeys["password"]
	assert.True(t, ok)
}

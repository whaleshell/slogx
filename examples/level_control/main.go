// Package main demonstrates live log-level control via HTTP and env watch.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/zorneth/slogx"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slogx.New(
		slogx.WithFormat(slogx.FormatJSON),
		slogx.WithLevel(slog.LevelInfo),
		slogx.WithCorporateMasking(),
	)

	addr, shutdown, err := log.ListenLevelHTTP(ctx, "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	fmt.Println("level endpoint:", "http://"+addr+"/level")

	go func() {
		time.Sleep(200 * time.Millisecond)
		req, _ := http.NewRequest(http.MethodPut, "http://"+addr+"/level?level=debug", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	for i := 0; i < 5; i++ {
		log.Debug("tick", "i", i)
		log.Info("tick-info", "i", i, "token", "sk-live-example-secret")
		time.Sleep(300 * time.Millisecond)
	}
}

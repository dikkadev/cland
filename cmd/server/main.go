package main

import (
	"log/slog"

	"github.com/dikkadev/cland/pkg/exchange"
	"github.com/dikkadev/prettyslog"
)

func main() {
	logger := prettyslog.NewPrettyslogHandler("cland", prettyslog.WithLevel(slog.LevelDebug))

	slog.SetDefault(slog.New(logger))

	handler := exchange.NewHandler("./tmp/input", "./tmp/error")
	err := handler.Start()
	if err != nil {
		panic(err)
	}

	select {}
}

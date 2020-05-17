package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GoodCodingFriends/animekai/annict"
	"github.com/GoodCodingFriends/animekai/config"
	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/GoodCodingFriends/animekai/server"
	"github.com/GoodCodingFriends/animekai/statistics"
	"github.com/kelseyhightower/envconfig"
	"github.com/morikuni/failure"
	"go.uber.org/zap"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v", err)
		os.Exit(1)
	}
}

func realMain() error {
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		return failure.Translate(err, errors.Internal)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return failure.Translate(err, errors.Internal)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("failed to sync", zap.Error(err))
		}
	}()

	handler := server.New(logger, statistics.New(annict.New(cfg.AnnictToken, cfg.AnnictEndpoint)))
	log.Printf("server listen in :8080")
	srv := &http.Server{Addr: ":8080", Handler: handler}
	return srv.ListenAndServe()
}

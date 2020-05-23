package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

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

	// if cfg.Env.IsDev() {
	// 	cfg.AnnictEndpoint = testutil.RunAnnictServer(&testing.RuntimeT{})
	// 	logger.Info("dummy Annict server is enabled", zap.String("addr", cfg.AnnictEndpoint))
	// }

	annictService := annict.New(cfg.AnnictToken, cfg.AnnictEndpoint)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := annictService.Stop(ctx); err != nil {
			logger.Error("failed to stop annict service", zap.Error(err))
		}
	}()

	handler := server.New(
		logger,
		statistics.New(annictService),
		cfg.Env.IsDev(),
	)
	logger.Info("server listen in :8000")
	srv := &http.Server{Addr: ":8000", Handler: handler}
	return srv.ListenAndServe()
}

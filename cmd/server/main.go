package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoodCodingFriends/animekai/annict"
	"github.com/GoodCodingFriends/animekai/config"
	"github.com/GoodCodingFriends/animekai/errors"
	"github.com/GoodCodingFriends/animekai/server"
	"github.com/GoodCodingFriends/animekai/slack"
	"github.com/GoodCodingFriends/animekai/statistics"
	"github.com/GoodCodingFriends/animekai/testutil"
	"github.com/kelseyhightower/envconfig"
	"github.com/mitchellh/go-testing-interface"
	"github.com/morikuni/failure"
	"github.com/rakyll/statik/fs"
	"go.uber.org/zap"

	_ "github.com/GoodCodingFriends/animekai/statik"
)

func main() {
	if err := realMain(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v", err)
		os.Exit(1)
	}
}

func realMain() error { //nolint:funlen
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

	if cfg.Env.IsDev() {
		cfg.AnnictEndpoint = testutil.RunAnnictServer(&testing.RuntimeT{}, nil)
		logger.Info("dummy Annict server is enabled", zap.String("addr", cfg.AnnictEndpoint))
	}

	var statikFS http.FileSystem
	if cfg.Env.IsProd() {
		fs, err := fs.New()
		if err != nil {
			return failure.Wrap(err)
		}
		statikFS = fs
	}

	annictService := annict.New(cfg.AnnictToken, cfg.AnnictEndpoint)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := annictService.Stop(ctx); err != nil {
			logger.Error("failed to stop annict service", zap.Error(err))
		}
	}()

	slackService := slack.NewCommandHandler(logger, cfg.SlackSigningSecret, cfg.SlackWebhookURL, annictService)

	handler := server.New(
		logger,
		statistics.New(annictService),
		slackService,
		statikFS,
		!cfg.Env.IsProd(),
	)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: handler}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigCh:
		case <-ctx.Done():
			return
		}

		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		if err := srv.Shutdown(cctx); err != nil {
			log.Printf("srv.Shutdown returned an error: %s", err)
		}
	}()

	logger.Info("server listen in :" + cfg.Port)
	return srv.ListenAndServe()
}

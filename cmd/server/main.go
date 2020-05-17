package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/GoodCodingFriends/animekai/annict"
	"github.com/GoodCodingFriends/animekai/config"
	"github.com/GoodCodingFriends/animekai/server"
	"github.com/GoodCodingFriends/animekai/statistics"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %s", err)
		os.Exit(1)
	}

	srv := server.New(statistics.New(annict.New(cfg.AnnictToken, cfg.AnnictEndpoint)))
	log.Fatal(http.ListenAndServe(":8080", srv))
}

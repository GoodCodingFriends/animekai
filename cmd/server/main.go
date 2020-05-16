package main

import (
	"log"
	"net/http"

	"github.com/GoodCodingFriends/animekai/server"
	"github.com/GoodCodingFriends/animekai/statistics"
)

func main() {
	srv := server.New(statistics.New())
	log.Fatal(http.ListenAndServe(":8080", srv))
}

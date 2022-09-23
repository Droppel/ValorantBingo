package main

import (
	"Bingo/bingo"
	"Bingo/bot"
	"Bingo/config"
	"Bingo/httpserver"
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {
	var err error

	rand.Seed(time.Now().UnixNano())
	if err != nil {
		log.WithError(err).Error("Failed to parse the config")
		return
	}

	logLevel, err := log.ParseLevel(config.Json.LogLevel)
	if err != nil {
		log.WithError(err).Error("Could not parse the loglevel")
		return
	}
	log.SetLevel(logLevel)

	bingo.Bingos = make(map[string]*bingo.Bingo)
	rand.Seed(time.Now().UnixNano())

	go bot.InitBot()

	httpserver.Listen()
}

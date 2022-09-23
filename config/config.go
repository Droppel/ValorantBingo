package config

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

type config struct {
	StoragePath  string       `json:"storagePath"`
	LogLevel     string       `json:"logLevel"`
	GameSettings gameSettings `json:"gameSettings"`
}

type gameSettings struct {
	TotalRerolls int `json:"totalRerolls"`
}

var Json config

func init() {

	configFile, err := ioutil.ReadFile("config.json")
	if err != nil {
		log.WithError(err).Fatal("Failed to read config file")
		return
	}

	Json = config{}
	err = json.Unmarshal(configFile, &Json)
	if err != nil {
		log.WithError(err).Fatal("Failed to unmarshal config")
		return
	}
}

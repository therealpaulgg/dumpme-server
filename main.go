package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/therealpaulgg/dumpme-server/models"
	"github.com/therealpaulgg/dumpme-server/router"
	"github.com/therealpaulgg/dumpme-server/services"
)

func main() {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic("Could not open config file (config.json).")
	}
	var config models.Configuration
	err = json.Unmarshal(file, &config)
	if err != nil {
		panic(err.Error())
	}
	if config.StorageType == "filesystem" {
		var saver *models.LocalStorageSaverAES
		if config.StoragePath != "" {
			_, err := os.Stat(config.StoragePath)
			if err != nil {
				panic(err.Error())
			}
			saver = &models.LocalStorageSaverAES{StoragePath: strings.TrimRight(strings.TrimRight(config.StoragePath, "/"), "\\")}
		} else {
			if _, err := os.Stat("dump"); os.IsNotExist(err) {
				err = os.Mkdir("dump", 0755)
				if err != nil {
					panic(err.Error())
				}
			}
			saver = &models.LocalStorageSaverAES{StoragePath: "dump"}
		}
		saver.SecretKey = config.SecretKey
		services.EncryptedFileSaver = saver
	}
	var port int
	if config.Port != 0 {
		if config.Port <= 0 || config.Port > 65535 {
			panic("Port is invalid (must be between 1 and 65535).")
		}
		port = config.Port
	} else {
		port = 8080
	}
	var environment string
	if config.Environment != "" {
		environment = config.Environment
	} else {
		environment = "production"
	}
	r := router.Router{Port: port, Env: environment}
	r.Launch()
}

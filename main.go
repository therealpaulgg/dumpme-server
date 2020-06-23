package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/alecthomas/units"
	"github.com/therealpaulgg/dumpme-server/models"
	"github.com/therealpaulgg/dumpme-server/router"
	"github.com/therealpaulgg/dumpme-server/services"
	"github.com/therealpaulgg/dumpme-server/services/filesystem"
)

func main() {
	// Attempt to read JSON file.
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic("Could not open config file (config.json).")
	}
	var config models.Configuration
	services.Configuration = models.AppConfiguration{}
	err = json.Unmarshal(file, &config)
	if err != nil {
		panic(err.Error())
	}
	// There are two modes for storage:
	// 1. Filesystem
	// 2. Buckets (like Amazon S3 and DigitalOcean Spaces) (not implemented yet)
	if config.StorageType == "filesystem" {
		var saver *filesystem.LocalStorageSaverAES

		if config.StoragePath != "" {
			_, err := os.Stat(config.StoragePath)
			if err != nil {
				panic(err.Error())
			}
			saver = &filesystem.LocalStorageSaverAES{StoragePath: strings.TrimRight(strings.TrimRight(config.StoragePath, "/"), "\\")}
		} else {
			if _, err := os.Stat("dump"); os.IsNotExist(err) {
				err = os.Mkdir("dump", 0755)
				if err != nil {
					panic(err.Error())
				}
			}
			saver = &filesystem.LocalStorageSaverAES{StoragePath: "dump"}
		}
		services.EncryptedFileSaver = saver
		if config.FileLimit == "-1" {
			services.Configuration.FileLimit = -1
		} else if config.FileLimit != "" {
			limit, err := units.ParseStrictBytes(config.FileLimit)
			if err != nil {
				panic(err.Error())
			}

			services.Configuration.FileLimit = limit
		} else {
			services.Configuration.FileLimit = 500000000
		}
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
	// currently does nothing
	var environment string
	if config.Environment != "" {
		environment = config.Environment
	} else {
		environment = "production"
	}
	// launch HTTP server
	r := router.Router{Port: port, Env: environment}
	r.Launch()
}

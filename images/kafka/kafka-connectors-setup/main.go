package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/mesosphere/kudo-kafka-operator/images/kafka/kafka-connectors-setup/pkgs/config"
	"github.com/mesosphere/kudo-kafka-operator/images/kafka/kafka-connectors-setup/pkgs/utils"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

var (
	app           = kingpin.New("kafka-connectors-setup", "A command-line kafka connectors configuration parser and setup helper.")
	configFlag    = app.Flag("config", "Path to the configuration file.").Required().File()
	_             = app.Command("download", "Downloads the resources needed for connectors.")
	register      = app.Command("register", "Register the connectors in kafka-connect.")
	endpoint      = register.Arg("endpoint", "Kafka Connect REST endpoint").Required().String()
	configuration = &config.ConfigurationSetup{
		Utils:      &utils.UtilsImpl{},
		ConfigFile: &config.ConfigFile{},
	}
)

func main() {
	log.Printf("Running kafka-connectors-setup...")
	parsed, _ := app.Parse(os.Args[1:])
	if *configFlag != nil {
		b, _ := ioutil.ReadAll(*configFlag)
		if err := yaml.Unmarshal(b, configuration.ConfigFile); err != nil {
			log.Fatalf("Error in parsing configuration file: %v", err)
		}
		switch parsed {
		case "register":
			log.Printf("Registering to endpoint %s", *endpoint)
			configuration.RegisterConnectors(*endpoint)

		case "download":
			downloadDirectory, _ := os.Getwd()
			configuration.DownloadConnectorResources(downloadDirectory)
			log.Printf("Parsing download only connector resources to '%s'", downloadDirectory)
			configuration.DownloadResources(downloadDirectory)
		}
	} else {
		log.Fatalf("No configuration file provided or the file cannot be accessed")
	}
}

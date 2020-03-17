package config

import (
	"log"
	"path"

	"github.com/mesosphere/kudo-kafka-operator/images/kafka/kafka-connectors-setup/pkgs/utils"
)

// Connector Struct
type Connector struct {
	Resources []string    `json:"resources,omitempty" yaml:"resources,omitempty"`
	Config    interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// ConfigFile Struct
type ConfigFile struct {
	Connectors map[string]Connector `json:"connectors,omitempty" yaml:"connectors,omitempty"`
	Resources  []string             `json:"resources,omitempty" yaml:"resources,omitempty"`
}

// ConfigurationSetup Struct
type ConfigurationSetup struct {
	Utils      utils.Utils
	ConfigFile *ConfigFile
}

// RegisterConnectors Registers Connectors to endpoint
func (c *ConfigurationSetup) RegisterConnectors(endpoint string) {
	for name, connector := range c.ConfigFile.Connectors {
		log.Printf("Registering connector: %s\n", name)
		err := c.Utils.RegisterConnector(endpoint, connector.Config)
		if err != nil {
			log.Fatalf("Error in registering connector: %v", err)
		}
	}
}

// DownloadConnectorResources Download Connector Resources
func (c *ConfigurationSetup) DownloadConnectorResources(downloadDirectory string) {
	for name, connector := range c.ConfigFile.Connectors {
		log.Printf("Parsing connector: %s", name)
		for _, resource := range connector.Resources {
			log.Printf("Downloading file: %s\n", resource)
			filename, err := c.Utils.DownloadFile(downloadDirectory, resource)
			if err != nil {
				log.Fatalf("Error in downloading resource %s: %v", resource, err)
			}
			log.Printf("Extracting file: %s\n", resource)
			err = c.Utils.ExtractFile(path.Join(downloadDirectory, filename), downloadDirectory)
			if err != nil {
				log.Fatalf("Error in extracting file: %v", err)
			}
		}
	}
}

// DownloadResources Download Resources
func (c *ConfigurationSetup) DownloadResources(downloadDirectory string) {
	for _, resource := range c.ConfigFile.Resources {
		log.Printf("Downloading file: %s\n", resource)
		filename, err := c.Utils.DownloadFile(downloadDirectory, resource)
		if err != nil {
			log.Fatalf("Error in downloading resource %s: %v", resource, err)
		}
		log.Printf("Extracting file: %s\n", resource)
		err = c.Utils.ExtractFile(path.Join(downloadDirectory, filename), downloadDirectory)
		if err != nil {
			log.Fatalf("Error in extracting file: %v", err)
		}
	}
}

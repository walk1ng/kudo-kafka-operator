package main

import (
	"github.com/mesosphere/kudo-kafka-operator/images/kafka-utils/pkgs/client"
	"github.com/mesosphere/kudo-kafka-operator/images/kafka-utils/pkgs/service"
	log "github.com/sirupsen/logrus"
)

const (
	KAFKA_HOME = "/opt/kafka"
)

func main() {
	log.Infoln("Running kafka-utils...")

	k8sClient, err := client.GetKubernetesClient()
	if err != nil {
		log.Fatalf("Error initializing client: %+v", err)
	}
	kafkaService := service.KafkaService{
		Client: k8sClient,
		Env:    &service.EnvironmentImpl{},
	}
	log.Infoln("Running kafka-utils...")
	err = kafkaService.WriteIngressToPath(KAFKA_HOME)
	if err != nil {
		log.Errorf("could not run the kafka utils bootstrap: %v", err)
	} else {
		log.Infoln("Finished the kafka-utils bootstrap.")
	}
}

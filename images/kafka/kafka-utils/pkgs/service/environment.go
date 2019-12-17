package service

import "os"

//go:generate mockgen -destination=../mocks/environment_mock.go -package=mocks github.com/mesosphere/kudo-kafka-operator/images/kafka-utils/pkgs/service Environment

type Environment interface {
	GetHostName() string
	GetNamespace() string
	GetExternalIngressPort() string
	GetNodeName() string
}

type EnvironmentImpl struct{}

func (c *EnvironmentImpl) GetHostName() string {
	return os.Getenv("HOSTNAME")
}

func (c *EnvironmentImpl) GetNamespace() string {
	return os.Getenv("NAMESPACE")
}

func (c *EnvironmentImpl) GetExternalIngressPort() string {
	return os.Getenv("EXTERNAL_INGRESS_PORT")
}

func (c *EnvironmentImpl) GetNodeName() string {
	return os.Getenv("NODE_NAME")
}

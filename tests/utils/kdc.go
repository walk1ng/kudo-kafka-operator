package utils

import (
	"errors"
	"os"
	"strings"
)

// KDCClient Struct defining the KDC Client
type KDCClient struct {
	Namespace string
}

const (
	PodName       = "kdc"
	ContainerName = "kdc"
)

// setNamespace Set Namespace
func (k *KDCClient) SetNamespace(Namespace string) {
	k.Namespace = Namespace
}

// Deploy Use it to deploy the kdc server
func (k *KDCClient) Deploy() error {
	repoRoot, exists := os.LookupEnv("REPO_ROOT")

	if exists {
		Apply(repoRoot+"/tests/suites/kafka_kerberos/resources/kdc.yaml", k.Namespace)
		return KClient.WaitForPod("kdc", k.Namespace, 240)
	}

	return errors.New("environment variable REPO_ROOT is not set")
}

// TearDown Use it to destroy the kdc server
func (k *KDCClient) TearDown() error {
	repoRoot, exists := os.LookupEnv("REPO_ROOT")

	if exists {
		Delete(repoRoot+"/tests/suites/kafka_kerberos/resources/kdc.yaml", k.Namespace)
		return nil
	}

	return errors.New("environment variable REPO_ROOT is not set")
}

// CreateKeytabSecret Pass it string array of principals and it will create a keytab secret
func (k *KDCClient) CreateKeytabSecret(principals []string, serviceName string, secretName string) error {
	//
	command := "printf \"" + strings.Join(principals, "\n") + "\n\" > /kdc/" + serviceName + "-principals.txt;" +
		"cat /kdc/" + serviceName + "-principals.txt | while read line; do /usr/sbin/kadmin -l add --use-defaults --random-password $line; done;" +
		"rm /kdc/" + serviceName + ".keytab;" +
		"cat /kdc/" + serviceName + "-principals.txt | while read line; do /usr/sbin/kadmin -l ext -k /kdc/" + serviceName + ".keytab $line; done;"

	_, err := KClient.ExecInPod(k.Namespace, PodName, ContainerName, []string{"/bin/sh", "-c", command})
	if err != nil {
		return err
	}
	stdout, err := KClient.ExecInPod(k.Namespace, PodName, ContainerName, []string{"/bin/sh", "-c", "cat /kdc/" + serviceName + ".keytab | base64 -w 0"})
	if err != nil {
		return err
	}
	KClient.createSecret(secretName, []string{"kafka.keytab", stdout}, k.Namespace)
	return nil
}

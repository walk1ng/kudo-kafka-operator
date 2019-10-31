package utils

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	defaultKakfaRetry         = 3
	defaultKakfaRetryInterval = 1 * time.Second
	defaultNamespace          = "default"
	defaultInstanceName       = "kafka"
	defaultKerberosEnabled    = false
)

type KafkaClient struct {
	kClient *KubernetesTestClient
	conf    *KafkaClientConfiguration
}

type KafkaClientConfiguration struct {
	Retry           *int
	RetryInterval   *time.Duration
	Namespace       *string
	InstanceName    *string
	KerberosEnabled bool
}

func NewKafkaClient(kuberentesTestClient *KubernetesTestClient, configuration *KafkaClientConfiguration) *KafkaClient {
	if configuration.RetryInterval == nil {
		configuration.RetryInterval = &defaultKakfaRetryInterval
	}
	if configuration.Retry == nil {
		configuration.Retry = &defaultKakfaRetry
	}
	if configuration.Namespace == nil {
		configuration.Namespace = &defaultNamespace
	}
	if configuration.InstanceName == nil {
		configuration.InstanceName = &defaultInstanceName
	}
	return &KafkaClient{
		kClient: kuberentesTestClient,
		conf:    configuration,
	}
}

func getKeytabsConfigCommand() string {
	return "printf 'KafkaClient {\ncom.sun.security.auth.module.Krb5LoginModule required\nuseKeyTab=true\nstoreKey=true\nuseTicketCache=false\nkeyTab=\"kafka.keytab\"\nprincipal=\"kafka/%s@LOCAL\";\n};' $(hostname -f) > /tmp/kafka_client_jaas.conf;" +
		"export KAFKA_OPTS=\"-Djava.security.auth.login.config=/tmp/kafka_client_jaas.conf -Djava.security.krb5.conf=${KAFKA_HOME}/config/krb5.conf\";" +
		"KAFKA_PRODUCER_CONFIG_OPTIONS=\"--producer-property sasl.mechanism=GSSAPI --producer-property security.protocol=SASL_PLAINTEXT --producer-property sasl.kerberos.service.name=kafka\";" +
		"KAFKA_CONSUMER_CONFIG_OPTIONS=\"--consumer-property sasl.mechanism=GSSAPI --consumer-property security.protocol=SASL_PLAINTEXT --consumer-property sasl.kerberos.service.name=kafka\";"
}

func (c *KafkaClient) WriteInTopic(podName, container, topicName, message string) (string, error) {
	return Retry(*c.conf.Retry, *c.conf.RetryInterval, ">>", func() (string, error) {
		return c.writeInTopic(podName, container, topicName, message)
	})
}

func (c *KafkaClient) writeInTopic(podName, container, topicName, message string) (string, error) {
	port, err := c.kClient.GetParamForKudoInstance(*c.conf.InstanceName, *c.conf.Namespace, "BROKER_PORT")
	if err != nil {
		logrus.Error(fmt.Sprintf("Error getting BROKER_PORT for kafka: %v\n", err))
		return "", err
	}
	command := []string{}
	if c.conf.KerberosEnabled {
		command = []string{
			"bash", "-c", getKeytabsConfigCommand() + fmt.Sprintf("echo '%s' | /opt/kafka/bin/kafka-console-producer.sh $KAFKA_PRODUCER_CONFIG_OPTIONS --broker-list "+
				"$(hostname -f):%s --topic %s", message, port, topicName),
		}
	} else {
		command = []string{
			"bash", "-c", fmt.Sprintf("echo '%s' | /opt/kafka/bin/kafka-console-producer.sh --broker-list "+
				"$(hostname -f):%s --topic %s", message, port, topicName),
		}
	}
	logrus.Println(command)
	return c.ExecInPod(*c.conf.Namespace, podName, container, command)
}

func (c *KafkaClient) ReadFromTopic(podName, container, topicName, message string) (string, error) {
	return Retry(*c.conf.Retry, *c.conf.RetryInterval, message, func() (string, error) {
		return c.readFromTopic(podName, container, topicName)
	})
}

func (c *KafkaClient) readFromTopic(podName, container, topicName string) (string, error) {
	port, err := c.kClient.GetParamForKudoInstance(*c.conf.InstanceName, *c.conf.Namespace, "BROKER_PORT")
	if err != nil {
		logrus.Error(fmt.Sprintf("Error getting BROKER_PORT for kafka: %v\n", err))
		return "", err
	}
	command := []string{}
	if c.conf.KerberosEnabled {
		command = []string{
			"bash", "-c", getKeytabsConfigCommand() + fmt.Sprintf("/opt/kafka/bin/kafka-console-consumer.sh $KAFKA_CONSUMER_CONFIG_OPTIONS --bootstrap-server "+
				"$(hostname -f):%s --topic %s --from-beginning --timeout-ms 5000", port, topicName),
		}
	} else {
		command = []string{
			"bash", "-c", fmt.Sprintf("/opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server "+
				"$(hostname -f):%s --topic %s --from-beginning --timeout-ms 5000", port, topicName),
		}
	}
	logrus.Println(command)
	return c.ExecInPod(*c.conf.Namespace, podName, container, command)
}

func (c *KafkaClient) CreateTopic(podName, container, topicName, replicationAssignment string) (string, error) {
	return Retry(*c.conf.Retry, *c.conf.RetryInterval, "Created topic", func() (string, error) {
		return c.createTopic(podName, container, topicName, replicationAssignment)
	})
}

func (c *KafkaClient) createTopic(podName, container, topicName, replicationAssignment string) (string, error) {
	command := []string{
		"bash", "-c", fmt.Sprintf("/opt/kafka/bin/kafka-topics.sh --create --zookeeper ${KAFKA_ZK_URI} "+
			"--topic %s --replica-assignment %s", topicName, replicationAssignment),
	}
	logrus.Println(command)
	return c.ExecInPod(*c.conf.Namespace, podName, container, command)
}

func (c *KafkaClient) WaitForBrokersToBeRegisteredWithService(podName, container string, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(3 * time.Second)
	for {
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("Timeout while waiting for broker %s to be registered", podName))
		case <-tick:
			if c.BrokerAddressIsRegistered(podName, container) {
				return nil
			}
		}
	}
}

func (c *KafkaClient) BrokerAddressIsRegistered(podName, container string) bool {
	port, err := c.kClient.GetParamForKudoInstance(*c.conf.InstanceName, *c.conf.Namespace, "BROKER_PORT")
	if err != nil {
		logrus.Error(fmt.Sprintf("Error getting BROKER_PORT for kafka: %v\n", err))
		return false
	}
	command := []string{
		"bash", "-c", fmt.Sprintf("/opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server=$(hostname -f):%s", port),
	}
	logrus.Println(command)
	output, _ := c.ExecInPod(*c.conf.Namespace, podName, container, command)
	logrus.Println(output)
	if strings.Contains(output, "Error connecting to node") {
		return false
	}
	return true
}

func (c *KafkaClient) ExecInPod(namespace, name, container string, commands []string) (string, error) {
	req := c.kClient.Clientset.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command:   commands,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
			Container: container,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(KubeConfig, http.MethodPost, req.URL())
	if err != nil {
		return "", err
	}

	stdOut := bytes.Buffer{}
	stdErr := bytes.Buffer{}

	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:             nil,
		Stdout:            bufio.NewWriter(&stdOut),
		Stderr:            bufio.NewWriter(&stdErr),
		Tty:               true,
		TerminalSizeQueue: nil,
	})

	if stdErr.Len() > 0 {
		stdErrString := stdErr.String()
		logrus.Infoln(stdErrString)
		return "", errors.New(stdErrString)
	}

	stdoutString := stdOut.String()
	logrus.Infoln(stdoutString)
	return stdoutString, nil
}

func InstallKudoOperator(namespace, name, operatorPathEnv string, parameters map[string]string) {
	validateFrameworkEnvVariable(operatorPathEnv)
	KClient.InstallOperatorFromPath(os.Getenv(operatorPathEnv), namespace, name, parameters)
	KClient.LogObjectsOfKinds(namespace, []string{"svc", "pdb", "operatorversions", "operators", "instance"})
}

func DeleteInstances(namespace, name string) {
	KClient.DeleteInstance(namespace, name)
}

func GetKafkaKeyabs(namespace string) []string {
	return []string{
		"livenessProbe/kafka-kafka-0.kafka-svc." + namespace + ".svc.cluster.local@LOCAL",
		"livenessProbe/kafka-kafka-1.kafka-svc." + namespace + ".svc.cluster.local@LOCAL",
		"livenessProbe/kafka-kafka-2.kafka-svc." + namespace + ".svc.cluster.local@LOCAL",
		"kafka/kafka-kafka-0.kafka-svc." + namespace + ".svc.cluster.local@LOCAL",
		"kafka/kafka-kafka-1.kafka-svc." + namespace + ".svc.cluster.local@LOCAL",
		"kafka/kafka-kafka-2.kafka-svc." + namespace + ".svc.cluster.local@LOCAL",
	}
}

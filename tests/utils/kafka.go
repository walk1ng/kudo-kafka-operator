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
	defaultKafkaRetry         = 3
	defaultKafkaRetryInterval = 1 * time.Second
	defaultNamespace          = "default"
	defaultInstanceName       = "kafka"
	DefaultStatefulReadyWait  = 300 * time.Second
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
	TLSEnabled      bool
}

func NewKafkaClient(kuberentesTestClient *KubernetesTestClient, configuration *KafkaClientConfiguration) *KafkaClient {
	if configuration.RetryInterval == nil {
		configuration.RetryInterval = &defaultKafkaRetryInterval
	}
	if configuration.Retry == nil {
		configuration.Retry = &defaultKafkaRetry
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

func (c *KafkaClient) getClientConfigurationCommand() string {
	conf := ""
	securityProtocol := ""
	if c.conf.TLSEnabled {
		conf = "KAFKA_PRODUCER_CONFIG_OPTIONS=\"--producer-property security.protocol=$SECURITY_PROTOCOL --producer-property ssl.keystore.location=/home/kafka/tls/kafka.server.keystore.jks " +
			"--producer-property ssl.keystore.password=changeit --producer-property ssl.key.password=changeit --producer-property ssl.truststore.location=/home/kafka/tls/kafka.server.truststore.jks " +
			"--producer-property ssl.truststore.password=changeit\"; KAFKA_CONSUMER_CONFIG_OPTIONS=\"--consumer-property security.protocol=$SECURITY_PROTOCOL " +
			"--consumer-property ssl.keystore.location=/home/kafka/tls/kafka.server.keystore.jks --consumer-property ssl.keystore.password=changeit --consumer-property ssl.key.password=changeit " +
			"--consumer-property ssl.truststore.location=/home/kafka/tls/kafka.server.truststore.jks --consumer-property ssl.truststore.password=changeit\";"
		securityProtocol = "SSL"
	}
	if c.conf.KerberosEnabled {
		conf = conf + "printf 'KafkaClient {\ncom.sun.security.auth.module.Krb5LoginModule required\nuseKeyTab=true\nstoreKey=true\nuseTicketCache=false\nkeyTab=\"kafka.keytab\"\nprincipal=\"kafka/%s@LOCAL\";\n};' " +
			"$(hostname -f) > /tmp/kafka_client_jaas.conf; export KAFKA_OPTS=\"-Djava.security.auth.login.config=/tmp/kafka_client_jaas.conf -Djava.security.krb5.conf=${KAFKA_HOME}/config/krb5.conf\";" +
			"KAFKA_PRODUCER_CONFIG_OPTIONS=\"$KAFKA_PRODUCER_CONFIG_OPTIONS --producer-property sasl.mechanism=GSSAPI --producer-property security.protocol=$SECURITY_PROTOCOL --producer-property sasl.kerberos.service.name=kafka\";" +
			"KAFKA_CONSUMER_CONFIG_OPTIONS=\"$KAFKA_CONSUMER_CONFIG_OPTIONS --consumer-property sasl.mechanism=GSSAPI --consumer-property security.protocol=$SECURITY_PROTOCOL --consumer-property sasl.kerberos.service.name=kafka\";"
		securityProtocol = "SASL_PLAINTEXT"
	}
	if c.conf.TLSEnabled && c.conf.KerberosEnabled {
		securityProtocol = "SASL_SSL"
	}
	return fmt.Sprintf("SECURITY_PROTOCOL=%s;%s", securityProtocol, conf)
}

func (c *KafkaClient) WriteInTopic(podName, container, topicName, message string) (string, error) {
	return Retry(*c.conf.Retry, *c.conf.RetryInterval, ">>", func() (string, error) {
		return c.writeInTopic(podName, container, topicName, message)
	})
}

func (c *KafkaClient) writeInTopic(podName, container, topicName, message string) (string, error) {
	var port string
	var err error
	if c.conf.TLSEnabled {
		port, err = c.kClient.GetParamForKudoInstance(*c.conf.InstanceName, *c.conf.Namespace, "BROKER_PORT_TLS")
	} else {
		port, err = c.kClient.GetParamForKudoInstance(*c.conf.InstanceName, *c.conf.Namespace, "BROKER_PORT")
	}
	if err != nil {
		logrus.Error(fmt.Sprintf("Error getting BROKER_PORT for kafka: %v\n", err))
		return "", err
	}
	var command []string
	if c.conf.KerberosEnabled {
		command = []string{
			"bash", "-c", c.getClientConfigurationCommand() + fmt.Sprintf("echo '%s' | /opt/kafka/bin/kafka-console-producer.sh $KAFKA_PRODUCER_CONFIG_OPTIONS --broker-list "+
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
	port, err := c.GetPortForInstance()
	if err != nil {
		logrus.Error(fmt.Sprintf("Error getting BROKER_PORT for kafka: %v\n", err))
		return "", err
	}
	var command []string
	if c.conf.KerberosEnabled {
		command = []string{
			"bash", "-c", c.getClientConfigurationCommand() + fmt.Sprintf("/opt/kafka/bin/kafka-console-consumer.sh $KAFKA_CONSUMER_CONFIG_OPTIONS --bootstrap-server "+
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

func (c *KafkaClient) DescribeTopic(podName, container, topicName string) (string, error) {
	return c.describeTopic(podName, container, topicName)
}

func (c *KafkaClient) createTopic(podName, container, topicName, replicationAssignment string) (string, error) {
	kafkaArgs := fmt.Sprintf("/opt/kafka/bin/kafka-topics.sh --create --zookeeper ${KAFKA_ZK_URI} "+
		"--topic %s", topicName)
	command := []string{"bash", "-c"}
	if strings.Contains(replicationAssignment, ":") {
		kafkaArgs = fmt.Sprintf("%s --replica-assignment %s", kafkaArgs, replicationAssignment)
	} else {
		kafkaArgs = fmt.Sprintf("%s --partitions 1 --replication-factor 1", kafkaArgs)
	}
	command = append(command, kafkaArgs)
	logrus.Println(command)
	return c.ExecInPod(*c.conf.Namespace, podName, container, command)
}

func (c *KafkaClient) describeTopic(podName, container, topicName string) (string, error) {
	kafkaArgs := fmt.Sprintf("/opt/kafka/bin/kafka-topics.sh --describe --zookeeper ${KAFKA_ZK_URI} "+
		"--topic %s ", topicName)
	command := []string{"bash", "-c"}
	command = append(command, kafkaArgs)
	logrus.Println(command)
	return c.ExecInPod(*c.conf.Namespace, podName, container, command)
}

func (c *KafkaClient) WaitForBrokersToBeRegisteredWithService(podName, container string, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(3 * time.Second) //nolint
	for {
		select {
		case <-timeout:
			return fmt.Errorf("Timeout while waiting for broker %s to be registered", podName)
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
	return !strings.Contains(output, "Error connecting to node")
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
	if err != nil {
		return "", err
	}

	if stdErr.Len() > 0 {
		stdErrString := stdErr.String()
		logrus.Infoln(stdErrString)
		return "", errors.New(stdErrString)
	}

	stdoutString := stdOut.String()
	logrus.Infoln(stdoutString)
	return stdoutString, nil
}
func (c *KafkaClient) GetPortForInstance() (string, error) {
	if c.conf.TLSEnabled {
		return c.kClient.GetParamForKudoInstance(*c.conf.InstanceName, *c.conf.Namespace, "BROKER_PORT_TLS")
	}
	return c.kClient.GetParamForKudoInstance(*c.conf.InstanceName, *c.conf.Namespace, "BROKER_PORT")
}
func (c *KafkaClient) CheckForValueInFile(expectedValue, filePath, ns, pod, containerName string) (string, error) {
	return Retry(*c.conf.Retry, *c.conf.RetryInterval, expectedValue, func() (string, error) {
		return c.ExecInPod(ns, pod, containerName,
			[]string{"grep", expectedValue, filePath})
	})

}

func InstallKudoOperator(namespace, name, operatorPathEnv string, parameters map[string]string) {
	validateFrameworkEnvVariable(operatorPathEnv)
	KClient.InstallOperatorFromPath(os.Getenv(operatorPathEnv), namespace, name, parameters)
	KClient.LogObjectsOfKinds(namespace, []string{"svc", "pdb", "operatorversions", "operators", "instance"})
}

func DeleteInstances(namespace, name string) {
	KClient.DeleteInstance(namespace, name)
}

func DeletePVCs(containsString string) error {
	return KClient.DeletePVCs(containsString)
}

func GetKafkaKeyTabs(brokers int, namespace string) []string {
	keyTabs := []string{}
	for i := 0; i < brokers; i++ {
		keyTabs = append(keyTabs,
			fmt.Sprintf("livenessProbe/kafka-kafka-%d.kafka-svc.%s.svc.cluster.local@LOCAL", i, namespace),
			fmt.Sprintf("kafka/kafka-kafka-%d.kafka-svc.%s.svc.cluster.local@LOCAL", i, namespace),
		)
	}
	return keyTabs
}

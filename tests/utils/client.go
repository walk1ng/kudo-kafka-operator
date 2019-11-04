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

	"github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1beta2"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	EMPTY_CONDITION = ""
	KAFKA_INSTANCE  = "kafka"
	ZK_INSTANCE     = "zookeeper-instance"
)

var (
	kubeconfig    = os.Getenv("KUBECONFIG")
	client, _     = GetKubernetesClient()
	KClient       = &KubernetesTestClient{client}
	KubeConfig, _ = BuildKubeConfig(kubeconfig)
)

type KubernetesTestClient struct {
	*kubernetes.Clientset
}

func BuildKubeConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		log.Infof("kubeconfig file: %s", kubeconfig)
		client, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.Fatalf("error creating kubernetes client from %s: %v", kubeconfig, err)
			return nil, err
		}
		return client, err
	}
	log.Infof("kubeconfig file: using InClusterConfig.")
	return rest.InClusterConfig()
}

func GetKubernetesClient() (*kubernetes.Clientset, error) {
	clientSet, err := kubernetes.NewForConfig(KubeConfig)
	if err != nil {
		log.Fatalf("error creating kubernetes client: %v", err)
		return nil, err
	}
	log.Infof("kubernetes client configured.")
	return clientSet, nil
}

func (c *KubernetesTestClient) createSecret(name string, data []string, namespace string) {
	c.CoreV1().Secrets(namespace).Create(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		StringData: map[string]string{
			data[0]: data[1],
		},
	})
	_, err := c.CoreV1().Secrets(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Warningf("%v", err)
	}
}

func (c *KubernetesTestClient) WaitForPod(name, namespace string, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			c.PrintLogsOfNamespace(namespace)
			return fmt.Errorf("Timeout while waiting for pod [%s/%s] count to be 1", namespace, name)
		case <-tick:
			if KClient.CheckIfPodExists(name, namespace) {
				return nil
			}
		}
	}
}

func (c *KubernetesTestClient) WaitForContainerToBeReady(containerName, podName, namespace string, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			c.PrintLogsOfNamespace(namespace)
			return fmt.Errorf("Timeout while waiting for container [%s/%s/%s] status to be READY", namespace, podName, containerName)
		case <-tick:
			pod, err := KClient.GetPod(podName, namespace)
			if kerrors.IsNotFound(err) {
				log.Warningf("Found 0 pods with name %s in namespace %s .", podName, namespace)
				continue
			} else if err != nil {
				log.Warningf("%v", err)
				continue
			}
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.Name == containerName {
					if containerStatus.Ready {
						log.Infof("Found 1 ready container of name %s in pod %s in %s namespace.", containerName, podName, namespace)
						return nil
					}
				}
			}
			log.Infof("Found 0 ready container of name %s in pod %s in %s namespace.", containerName, podName, namespace)
		}
	}
}

func (c *KubernetesTestClient) WaitForStatefulSetCount(name, namespace string, count int, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			c.PrintLogsOfNamespace(namespace)
			return errors.New(fmt.Sprintf("Timeout while waiting for statefulset [%s/%s] count to be %d", namespace, name, count))
		case <-tick:
			if count == KClient.GetStatefulSetCount(name, namespace) {
				return nil
			}
		}
	}
}

func (c *KubernetesTestClient) WaitForStatefulSetReadyReplicasCount(name, namespace string, count int, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-timeout:
			c.PrintLogsOfNamespace(namespace)
			return errors.New(fmt.Sprintf("Timeout while waiting for statefulset [%s/%s] ready replicas count to be %d", namespace, name, count))
		case <-tick:
			if count == KClient.GetStatefulSetReadyReplicasCount(name, namespace) {
				return nil
			}
		}
	}
}

func (c *KubernetesTestClient) CheckIfPodExists(name, namespace string) bool {
	_, err := c.GetPod(name, namespace)
	if kerrors.IsNotFound(err) {
		log.Warningf("Found 0 pods with name %s in namespace %s .", name, namespace)
		return false
	} else if err != nil {
		log.Warningf("%v", err)
		return false
	}
	log.Infof("Found 1 pod with name %s in %s namespace", name, namespace)
	return true
}

func (c *KubernetesTestClient) GetPod(name, namespace string) (*v1.Pod, error) {
	return c.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
}

func (c *KubernetesTestClient) GetStatefulSetCount(name, namespace string) int {
	statefulSet := c.GetStatefulSet(name, namespace)
	if statefulSet == nil {
		log.Warningf("Found 0 replicas for statefulset %s in namespace %s .", name, namespace)
		return 0
	}
	log.Infof("Found %d  replicas of the %s in %s namespace", *statefulSet.Spec.Replicas, name, namespace)
	return int(*statefulSet.Spec.Replicas)
}

func (c *KubernetesTestClient) GetStatefulSetReadyReplicasCount(name, namespace string) int {
	statefulSet := c.GetStatefulSet(name, namespace)
	if statefulSet == nil {
		log.Warningf("Found 0 replicas for statefulset %s in %s namespace.", name, namespace)
		return 0
	}
	log.Infof("Found %d/%d ready replicas of the %s in %s namespace.", statefulSet.Status.ReadyReplicas, *statefulSet.Spec.Replicas, name, namespace)
	return int(statefulSet.Status.ReadyReplicas)
}

func (c *KubernetesTestClient) GetStatefulSet(name, namespace string) *v1beta2.StatefulSet {
	statefulSet, err := c.AppsV1beta2().StatefulSets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Warningf("%v", err)
		return nil
	}
	return statefulSet
}

func (c *KubernetesTestClient) ListStatefulSets(namespace string) {
	statefulSets, err := c.AppsV1beta2().StatefulSets(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Warningf("%v", err)
	}
	log.Infoln(statefulSets)
}

func (c *KubernetesTestClient) GetServices(name string, namespace string) *v1.ServiceList {
	listOptions := metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", name)}
	services, err := c.CoreV1().Services(namespace).List(listOptions)
	if err != nil {
		log.Fatalf("Error listing services for [%s] in namespace [%s]: %v", name, namespace, err.Error())
	}
	return services
}

func (c *KubernetesTestClient) GetServicesCount(name string, namespace string) int {
	services := c.GetServices(name, namespace)
	if services == nil {
		log.Warningf("Found 0 services %s in %s namespace  ", name, namespace)
		return 0
	}
	log.Infof("Found %d  services of the %s in %s namespace", len(services.Items), name, namespace)
	return len(services.Items)
}

func Setup(namespace string) {
	InstallKudoOperator(namespace, ZK_INSTANCE, ZK_FRAMEWORK_DIR_ENV, map[string]string{
		"MEMORY": "256Mi",
		"CPUS":   "0.25",
	})
	KClient.WaitForStatefulSetCount(suites.DefaultZkStatefulSetName, namespace, 3, 30)
	InstallKudoOperator(namespace, KAFKA_INSTANCE, KAFKA_FRAMEWORK_DIR_ENV, map[string]string{
		"BROKER_MEM":  "1Gi",
		"BROKER_CPUS": "0.25",
	})
	KClient.WaitForStatefulSetCount(suites.DefaultKafkaStatefulSetName, namespace, 3, 30)
}

func SetupWithKerberos(namespace string) {
	InstallKudoOperator(namespace, ZK_INSTANCE, ZK_FRAMEWORK_DIR_ENV, map[string]string{})
	KClient.WaitForStatefulSetCount(suites.DefaultZkStatefulSetName, namespace, 3, 30)
	InstallKudoOperator(namespace, KAFKA_INSTANCE, KAFKA_FRAMEWORK_DIR_ENV, map[string]string{
		"KERBEROS_ENABLED":       "true",
		"KERBEROS_KDC_HOSTNAME":  "kdc-service",
		"KERBEROS_KDC_PORT":      "2500",
		"KERBEROS_KEYTAB_SECRET": "base64-kafka-keytab-secret",
	})
	KClient.WaitForStatefulSetCount(suites.DefaultKafkaStatefulSetName, namespace, 3, 30)
}

func TearDown(namespace string) {
	DeleteInstances(namespace, ZK_INSTANCE)
	KClient.WaitForStatefulSetCount(fmt.Sprintf("%s-%s", ZK_INSTANCE, ZK_INSTANCE), namespace, 0, 30)
	DeleteInstances(namespace, KAFKA_INSTANCE)
	KClient.WaitForStatefulSetCount(fmt.Sprintf("%s-%s", KAFKA_INSTANCE, KAFKA_INSTANCE), namespace, 0, 30)
}

func Retry(attempts int, sleep time.Duration, condition string, f func() (string, error)) (resp string, err error) {
	for i := 0; ; i++ {
		resp, err = f()
		if err == nil {
			if len(condition) > 0 && strings.Contains(resp, condition) {
				return resp, nil
			}
			if len(condition) == 0 {
				return resp, nil
			}
		}
		if i >= (attempts - 1) {
			break
		}
		time.Sleep(sleep)
		log.Infoln("retrying after error:", err)
	}
	return resp, fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func (c *KubernetesTestClient) DeletePVCs(containsString string) error {
	pvcs, err := c.CoreV1().PersistentVolumeClaims("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pvc := range pvcs.Items {
		if strings.Contains(pvc.GetName(), containsString) {
			err = c.CoreV1().PersistentVolumeClaims(pvc.GetNamespace()).Delete(pvc.GetName(), &metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *KubernetesTestClient) ExecInPod(namespace string, podName string, containerName string, commands []string) (string, error) {
	req := c.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command:   commands,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
			Container: containerName,
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

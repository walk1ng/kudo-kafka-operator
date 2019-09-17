package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mesosphere/data-services-kudo/frameworks/kafka/tests/suites"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1beta2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	EMPTY_CONDITION = ""
	KAFKA_INSTANCE  = "kafka"
	ZK_INSTANCE     = "zk"
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
	InstallKudoOperator(namespace, ZK_INSTANCE, ZK_FRAMEWORK_DIR_ENV, "")
	KClient.WaitForStatefulSetCount(suites.DefaultZkStatefulSetName, namespace, 3, 30)
	InstallKudoOperator(namespace, KAFKA_INSTANCE, KAFKA_FRAMEWORK_DIR_ENV, "")
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

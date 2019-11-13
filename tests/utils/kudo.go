package utils

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	kudoClient *versioned.Clientset
)

func (c *KubernetesTestClient) GetInstancesInNamespace(namespace string) (*v1beta1.InstanceList, error) {
	instancesClient := kudoClient.KudoV1beta1().Instances(namespace)
	instancesList, err := instancesClient.List(metav1.ListOptions{})
	if err != nil {
		log.Errorf("error getting kudo instances in namespace kubernetes client: %v", err)
		return nil, err
	}
	return instancesList, nil
}

func (c *KubernetesTestClient) GetParamForKudoInstance(name, namespace, param string) (string, error) {
	instancesClient := kudoClient.KudoV1beta1().Instances(namespace)
	instance, err := instancesClient.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("error getting kudo instance in namespace kubernetes client: %v", err)
		return "", err
	}

	if len(instance.Spec.Parameters[param]) == 0 {
		return c.GetParamForKudoFrameworkVersion(name, namespace, param)
	}
	return instance.Spec.Parameters[param], nil
}

func (c *KubernetesTestClient) GetParamForKudoFrameworkVersion(name, namespace, param string) (string, error) {
	frameworkVersionClient := kudoClient.KudoV1beta1().OperatorVersions(namespace)
	operatorVersionsList, err := frameworkVersionClient.List(metav1.ListOptions{})
	kafkaFrameworkVersion := ""
	for _, value := range operatorVersionsList.Items {
		if strings.Contains(value.Name, defaultInstanceName) && len(kafkaFrameworkVersion) > 0 {
			return "", errors.New("unsupported: two versions of Kafka OperatorVersion")
		}
		if strings.Contains(value.Name, defaultInstanceName) {
			kafkaFrameworkVersion = value.Name
		}
	}
	if kafkaFrameworkVersion == "" {
		log.Errorf("error finding a OperatorVersion for Kafka")
		return "", errors.New("no OperatorVersion for Kafka found")
	}
	log.Infof("Using OperatorVersion %s\n", kafkaFrameworkVersion)
	version, err := frameworkVersionClient.Get(kafkaFrameworkVersion, metav1.GetOptions{})
	if err != nil {
		log.Errorf("error getting kudo instance in namespace kubernetes client: %v", err)
		return "", err
	}

	for _, value := range version.Spec.Parameters {
		if value.Name == param {
			return *value.Default, nil
		}
	}
	return "", nil
}

func (c *KubernetesTestClient) UpdateInstancesCount(name, namespace string, count int) error {
	_, err := Retry(3, 0*time.Second, EMPTY_CONDITION, func() (string, error) {
		return updateInstancesCount(name, namespace, count)
	})
	return err
}

func updateInstancesCount(name, namespace string, count int) (string, error) {
	instancesClient := kudoClient.KudoV1beta1().Instances(namespace)
	instance, err := instancesClient.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("error getting kudo instance in namespace %s for instance %s kubernetes client: %v", namespace, name, err)
		return "", err
	}

	params := make(map[string]string)
	for k, v := range instance.Spec.Parameters {
		params[k] = v
	}
	params["BROKER_COUNT"] = strconv.Itoa(count)
	instance.Spec.Parameters = params

	_, err = instancesClient.Update(instance)
	if err != nil {
		log.Errorf("error updating kudo instance in namespace %s for instance %s kubernetes client: %v", namespace, name, err)
		return "", err
	}
	log.Infof("Updated the instances of %s/%s to %d", namespace, name, count)
	return "updated", nil
}

func (c *KubernetesTestClient) InstallOperatorFromPath(resourcesAbsoluteDirectoryPath, namespace, name string, params map[string]string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))
	log.Info(fmt.Sprintf("Installing framework from PATH: %s", resourcesAbsoluteDirectoryPath))

	install_cmd := []string{"kudo", "install", fmt.Sprintf("--instance=%s", name),
		resourcesAbsoluteDirectoryPath, fmt.Sprintf("--namespace=%s", namespace)}

	for key, val := range params {
		install_cmd = append(install_cmd, "-p", fmt.Sprintf("%s=%s", key, val))
	}

	cmd := exec.Command(kubectlPath, install_cmd...)
	out, err := cmd.Output()
	if err != nil {
		log.Error(string(err.(*exec.ExitError).Stderr))
	}
	log.Info(fmt.Sprintf("Response: %s", string(out)))
}

func (c *KubernetesTestClient) DeleteInstance(namespace, name string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))
	cmd := exec.Command(kubectlPath, "delete", "instances", name, fmt.Sprintf("--namespace=%s", namespace))
	out, err := cmd.Output()
	if err != nil {
		log.Error(string(err.(*exec.ExitError).Stderr))
	}
	log.Info(fmt.Sprintf("Response: %s", string(out)))
}

func (c *KubernetesTestClient) LogObjectsOfKinds(namespace string, components []string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))
	for _, objectKind := range components {
		cmd := exec.Command(kubectlPath, "get", objectKind, fmt.Sprintf("--namespace=%s", namespace))
		log.Infoln(fmt.Sprintf("get %s --namespace=%s", objectKind, namespace))
		out, err := cmd.Output()
		if err != nil {
			log.Error(string(err.(*exec.ExitError).Stderr))
		}
		log.Info(fmt.Sprintf(string(out)))
	}
}

func (c *KubernetesTestClient) PrintLogsOfPod(containerName, podName, namespace string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))

	cmd := exec.Command(kubectlPath, "logs", containerName, podName, fmt.Sprintf("--namespace=%s", namespace))
	log.Infoln(fmt.Sprintf("logs %s -c %s --namespace=%s", podName, containerName, namespace))
	out, err := cmd.Output()
	if err != nil {
		log.Errorf("%v", err)
	}
	log.Info(string(out))
}

func (c *KubernetesTestClient) PrintLogsOfNamespace(namespace string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))

	cmd := exec.Command(kubectlPath, "logs", fmt.Sprintf("-l heritage=kudo"), fmt.Sprintf("--namespace=%s", namespace))
	log.Infoln(fmt.Sprintf("logs %s --heritage=kudo", namespace))
	out, err := cmd.Output()
	if err != nil {
		log.Error(string(err.(*exec.ExitError).Stderr))
	}
	log.Info(fmt.Sprintf(string(out)))

}

func init() {
	kudoClient, _ = versioned.NewForConfig(KubeConfig)
}

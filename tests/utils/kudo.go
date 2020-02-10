package utils

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
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
		return c.GetParamForKudoFrameworkVersion(instance.Spec.OperatorVersion, namespace, param)
	}
	log.Info(fmt.Sprintf("Parameter %s Value is %s", param, instance.Spec.Parameters[param]))
	return instance.Spec.Parameters[param], nil
}

func (c *KubernetesTestClient) GetParamForKudoFrameworkVersion(ref corev1.ObjectReference, namespace, param string) (string, error) {
	operatorVersion, err := kudoClient.KudoV1beta1().OperatorVersions(namespace).Get(ref.Name, metav1.GetOptions{})

	if err != nil {
		log.Errorf("error getting kudo opeartor version in namespace kubernetes client: %v", err)
		return "", err
	}

	for _, value := range operatorVersion.Spec.Parameters {
		if value.Name == param {
			return *value.Default, nil
		}
	}
	return "", nil
}

func (c *KubernetesTestClient) GetOperatorVersionForKudoInstance(name, namespace string) (string, error) {
	instancesClient := kudoClient.KudoV1beta1().Instances(namespace)
	instance, err := instancesClient.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("error getting kudo instance in namespace kubernetes client: %v", err)
		return "", err
	}

	operatorVersion, err := kudoClient.KudoV1beta1().OperatorVersions(namespace).Get(instance.Spec.OperatorVersion.Name, metav1.GetOptions{})

	if err != nil {
		log.Errorf("error getting kudo opeartor version in namespace kubernetes client: %v", err)
		return "", err
	}

	log.Infof("Version: %s for %s", operatorVersion.Spec.Version, name)

	return operatorVersion.Spec.Version, nil
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

func (c *KubernetesTestClient) UpdateInstanceParams(name, namespace string, mapParam map[string]string) error {
	_, err := Retry(3, 0*time.Second, EMPTY_CONDITION, func() (string, error) {
		return updateInstanceParams(name, namespace, mapParam)
	})
	return err
}

func updateInstanceParams(name, namespace string, mapParam map[string]string) (string, error) {
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
	for k, v := range mapParam {
		params[k] = v
	}
	instance.Spec.Parameters = params

	_, err = instancesClient.Update(instance)
	if err != nil {
		log.Errorf("error updating kudo instance in namespace %s for instance %s kubernetes client: %v", namespace, name, err)
		return "", err
	}
	log.Infof("Updated the parameter(s) %s of %s/%s", mapParam, namespace, name)
	return "updated", nil
}

func (c *KubernetesTestClient) InstallOperatorFromPath(resourcesAbsoluteDirectoryPath, namespace, name string, params map[string]string) {
	log.Info(fmt.Sprintf("Installing framework from PATH: %s", resourcesAbsoluteDirectoryPath))
	c.installOrUpgradeOperator("install", namespace, resourcesAbsoluteDirectoryPath, name, "", params)
}

func (c *KubernetesTestClient) InstallOperatorFromRepository(namespace, operatorName, instanceName, version string, params map[string]string) {
	c.installOrUpgradeOperator("install", namespace, operatorName, instanceName, version, params)
}

func (c *KubernetesTestClient) UpgardeInstanceFromPath(resourcesAbsoluteDirectoryPath, namespace, name string, params map[string]string) {
	log.Info(fmt.Sprintf("Upgrading framework from PATH: %s", resourcesAbsoluteDirectoryPath))
	c.installOrUpgradeOperator("upgrade", namespace, resourcesAbsoluteDirectoryPath, name, "", params)
}

func (c *KubernetesTestClient) UpgardeInstanceFromRepository(namespace, operatorName, instanceName, version string, params map[string]string) {
	c.installOrUpgradeOperator("upgrade", namespace, operatorName, instanceName, version, params)
}

func (c *KubernetesTestClient) installOrUpgradeOperator(operation, namespace, operatorNameOrPath, instanceName, version string, params map[string]string) {
	if operation != "install" && operation != "upgrade" {
		log.Error(fmt.Sprintf("Operation not recognized: %s", operation))
		return
	}
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))

	install_cmd := []string{
		"kudo",
		operation,
		operatorNameOrPath,
		fmt.Sprintf("--instance=%s", instanceName),
		fmt.Sprintf("--namespace=%s", namespace),
	}

	if version != "" {
		install_cmd = append(install_cmd, fmt.Sprintf("--operator-version=%s", version))
	}

	for key, val := range params {
		install_cmd = append(install_cmd, "-p", fmt.Sprintf("%s=%s", key, val))
	}

	cmd := exec.Command(kubectlPath, install_cmd...)
	log.Infoln(cmd.Args)
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
	log.Infoln(cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		log.Error(string(err.(*exec.ExitError).Stderr))
	}
	log.Info(fmt.Sprintf("Response: %s", string(out)))
}

func (c *KubernetesTestClient) WaitForReadyStatus(name, namespace string, timeoutSeconds time.Duration) error {
	timeout := time.After(timeoutSeconds * time.Second)
	tick := time.Tick(2 * time.Second)
	for {
		select {
		case <-timeout:
			c.PrintLogsOfNamespace(namespace)
			return fmt.Errorf("Timeout while waiting for %s/%s plan status to be Complete", namespace, name)
		case <-tick:
			status, _ := c.GetPlanStatusForInstance(name, namespace)
			log.Info(fmt.Sprintf("Got status %s of instance %s in namespace %s", status, name, namespace))
			if status == v1beta1.ExecutionComplete {
				return nil
			}
		}
	}
}

func (c *KubernetesTestClient) GetPlanStatusForInstance(name, namespace string) (v1beta1.ExecutionStatus, error) {
	instancesClient := kudoClient.KudoV1beta1().Instances(namespace)
	instance, err := instancesClient.Get(name, metav1.GetOptions{})
	if err != nil {
		log.Errorf("error getting kudo instance in namespace %s for instance %s kubernetes client: %v", namespace, name, err)
		return "", err
	}
	return instance.Status.AggregatedStatus.Status, nil
}

func (c *KubernetesTestClient) LogObjectsOfKinds(namespace string, components []string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))
	for _, objectKind := range components {
		cmd := exec.Command(kubectlPath, "get", objectKind, fmt.Sprintf("--namespace=%s", namespace))
		log.Infoln(cmd.Args)
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

	cmd := exec.Command(kubectlPath, "logs", podName, "-c", containerName, fmt.Sprintf("--namespace=%s", namespace))
	log.Infoln(cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		log.Errorf("%v", err)
	}
	log.Info(string(out))
}

func (c *KubernetesTestClient) PrintLogsOfNamespace(namespace string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))

	cmd := exec.Command(kubectlPath, "logs", fmt.Sprintf("-l heritage=kudo"), "-c", suites.DefaultContainerName, fmt.Sprintf("--namespace=%s", namespace))
	log.Infoln(cmd.Args)
	out, err := cmd.Output()
	if err != nil {
		log.Error(string(err.(*exec.ExitError).Stderr))
	}
	log.Info(fmt.Sprintf(string(out)))

}

func init() {
	kudoClient, _ = versioned.NewForConfig(KubeConfig)
}

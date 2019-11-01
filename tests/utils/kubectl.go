package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const (
	KAFKA_FRAMEWORK_DIR_ENV = "KAFKA_FRAMEWORK_DIR"
	ZK_FRAMEWORK_DIR_ENV    = "ZK_FRAMEWORK_DIR"
)

var (
	kafkaFrameworkDir = os.Getenv(KAFKA_FRAMEWORK_DIR_ENV)
	zkFrameworkDir    = os.Getenv(ZK_FRAMEWORK_DIR_ENV)
)

type environment struct {
	kubectlPath string
	namespace   string
}

func Apply(resourcesAbsoluteDirectoryPath, namespace string) {
	applyManifests(resourcesAbsoluteDirectoryPath, "apply", namespace)
}

func Delete(resourcesAbsoluteDirectoryPath, namespace string) {
	applyManifests(resourcesAbsoluteDirectoryPath, "delete", namespace)
}

func applyManifests(resourcesAbsoluteDirectoryPath, action, namespace string) {
	kubectlPath := getKubectlPath()
	log.Info(fmt.Sprintf("Using kubectl from path: %s", kubectlPath))
	log.Info(fmt.Sprintf("Applying templates in directory: %s", resourcesAbsoluteDirectoryPath))
	env := environment{kubectlPath, namespace}
	if action == "apply" {
		filepath.Walk(resourcesAbsoluteDirectoryPath, env.applyManifest)
	} else if action == "delete" {
		filepath.Walk(resourcesAbsoluteDirectoryPath, env.deleteManifest)
	}
}

func getKubectlPath() string {
	validateFrameworkEnvVariable("KUBECTL_PATH")
	return os.Getenv("KUBECTL_PATH")
}

func (env *environment) apply(filePath string, info os.FileInfo, err error, delete bool) error {
	if err != nil {
		log.Error(fmt.Sprintf("Error accessing filePath %q: %v\n", filePath, err))
		return err
	}
	action := "apply"
	if delete {
		action = "delete"
	}
	if !info.IsDir() {
		log.Info(fmt.Sprintf("%s Template: %q\n", action, filePath))
		cmd := exec.Command(env.kubectlPath, action, "-f", filePath, "--namespace", env.namespace)
		out, err := cmd.Output()
		if err != nil {
			log.Error(string(err.(*exec.ExitError).Stderr))
		}
		log.Info(fmt.Sprintf("Response: %s", string(out)))
	}

	return nil
}

func (env *environment) applyManifest(filePath string, info os.FileInfo, err error) error {
	return env.apply(filePath, info, err, false)
}

func (env *environment) deleteManifest(filePath string, info os.FileInfo, err error) error {
	return env.apply(filePath, info, err, true)
}

func validateFrameworkEnvVariable(variable string) {
	value := os.Getenv(variable)
	if len(value) == 0 {
		log.Fatalf("cannot find the value for env variable %s", variable)
	}
}

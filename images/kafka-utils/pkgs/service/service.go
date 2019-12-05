package service

//go:generate mockgen -destination=../mocks/service_mock.go -package=mocks github.com/mesosphere/kudo-kafka-operator/images/kafka-utils/pkgs/service Service

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/avast/retry-go"
	v1 "k8s.io/api/core/v1"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

const (
	EXTERNAL_ADVERTISED_LISTENERS_PATH        = "external.advertised.listeners"
	EXTERNAL_LISTENERS                        = "external.listeners"
	EXTERNAL_ADVERTISED_LISTENER_SECURITY_MAP = "external.listener.security.protocol.map"
	EXTERNAL_DNS                              = "external.dns"
	EXTERNAL_INGRESS_PROTOCOL_NAME            = "EXTERNAL_INGRESS"
)

type Service interface {
	WriteIngressToPath(path string) error
}

type KafkaService struct {
	ServiceTypeLoadBalancer string
	Port                    int32
	Client                  kubernetes.Interface
	Env                     Environment
}

func (c *KafkaService) WriteIngressToPath(path string) error {
	hostname := c.Env.GetHostName()
	serviceName := fmt.Sprintf("%s-external", hostname)
	if len(hostname) == 0 {
		return fmt.Errorf("env variable HOSTNAME not found")
	}
	log.Infof("Checking the service created for %s", hostname)
	kafkaServices, err := c.Client.CoreV1().Services(c.Env.GetNamespace()).List(
		metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("metadata.name", serviceName).String(),
		})
	if err != nil {
		log.Errorf("Error listing the services for %s: %v", serviceName, err)
	}

	if len(kafkaServices.Items) == 0 {
		log.Infof("No service found for %s", serviceName)
		return nil
	}

	ingressStatus := []v1.LoadBalancerIngress{}

	for _, kafkaService := range kafkaServices.Items {
		c.ServiceTypeLoadBalancer = string(kafkaService.Spec.Type)
		switch kafkaService.Spec.Type {
		case v1.ServiceTypeLoadBalancer:
			log.Infoln("detected ", v1.ServiceTypeLoadBalancer)
			// loadbalancers might depend on external cloud providers and are not always ready when the broker is bootstrapping
			if kafkaService.Status.LoadBalancer.Size() == 0 {
				log.Infoln("The loadbalancer status is pending... ", v1.ServiceTypeLoadBalancer)
				retry.Do(func() error {
					kafkaSvc, err := c.Client.CoreV1().Services(c.Env.GetNamespace()).Get(kafkaService.Name, metav1.GetOptions{})
					if err != nil {
						return err
					}
					if kafkaSvc.Status.LoadBalancer.Size() == 0 {
						log.Infoln("The loadbalancer status is still pending... ", v1.ServiceTypeLoadBalancer)
						return fmt.Errorf("retry failed: loadbalancer status is pending... ")
					}
					log.Infoln("The loadbalancer status found.")
					ingressStatus = kafkaSvc.Status.LoadBalancer.Ingress
					return nil
				})
			} else {
				ingressStatus = kafkaService.Status.LoadBalancer.Ingress
			}
		case v1.ServiceTypeNodePort:
			log.Infoln("detected ", v1.ServiceTypeNodePort)
			externalIP, err := c.getNodeExternalIP()
			if err != nil {
				return err
			}
			log.Infoln("detected ExternalIP: ", externalIP)
			ingressStatus = []v1.LoadBalancerIngress{
				{
					IP:       externalIP,
					Hostname: "",
				},
			}
			for _, port := range kafkaService.Spec.Ports {
				c.Port = port.NodePort
			}
		case v1.ServiceTypeExternalName:
			log.Infof("detected %s but cannot reach any kafka pods through it", v1.ServiceTypeExternalName)
			return nil
		case v1.ServiceTypeClusterIP:
			log.Infof("detected %s. Currently not supported for ingress. For internal usage use the default headless service", v1.ServiceTypeExternalName)
			return nil
		default:
			log.Infof("service type '%s' detected but not supported", kafkaService.Spec.Type)
			return nil
		}
	}

	c.writeAdvertisedListenersToPath(ingressStatus, fmt.Sprintf("%s/%s", path, EXTERNAL_ADVERTISED_LISTENERS_PATH))
	c.writeListenersToPath(ingressStatus, fmt.Sprintf("%s/%s", path, EXTERNAL_LISTENERS))
	c.writeListenerSecurityProtocolMap(ingressStatus, fmt.Sprintf("%s/%s", path, EXTERNAL_ADVERTISED_LISTENER_SECURITY_MAP))
	c.writeListenerDNS(ingressStatus, fmt.Sprintf("%s/%s", path, EXTERNAL_DNS))

	return nil
}

func (c *KafkaService) writeAdvertisedListenersToPath(ingresses []v1.LoadBalancerIngress, path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("failed creating file '%s': %s", path, err)
		return err
	}
	dataWriter := bufio.NewWriter(file)
	var port string
	if c.Port == 0 {
		port = c.Env.GetExternalIngressPort()
	} else {
		port = strconv.FormatInt(int64(c.Port), 10)
	}

	for _, ingress := range ingresses {
		if len(ingress.Hostname) > 0 {
			dataWriter.WriteString(fmt.Sprintf("%s://%s:%s", EXTERNAL_INGRESS_PROTOCOL_NAME, ingress.Hostname, port))
		}
		if len(ingress.IP) > 0 {
			dataWriter.WriteString(fmt.Sprintf("%s://%s:%s", EXTERNAL_INGRESS_PROTOCOL_NAME, ingress.IP, port))
		}
	}
	dataWriter.Flush()
	file.Close()
	log.Infof("created the %s file", path)
	return nil
}

func (c *KafkaService) writeListenersToPath(ingresses []v1.LoadBalancerIngress, path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("failed creating file '%s': %s", path, err)
		return err
	}
	datawriter := bufio.NewWriter(file)
	var port string
	if c.Port == 0 {
		port = c.Env.GetExternalIngressPort()
	} else {
		port = strconv.FormatInt(int64(c.Port), 10)
	}

	for _, ingress := range ingresses {
		if len(ingress.Hostname) > 0 {
			datawriter.WriteString(fmt.Sprintf("%s://0.0.0.0:%s", EXTERNAL_INGRESS_PROTOCOL_NAME, port))
		}
		if len(ingress.IP) > 0 {
			datawriter.WriteString(fmt.Sprintf("%s://0.0.0.0:%s", EXTERNAL_INGRESS_PROTOCOL_NAME, port))
		}
	}

	datawriter.Flush()
	file.Close()
	log.Infof("created the %s file", path)
	return nil
}

func (c *KafkaService) writeListenerSecurityProtocolMap(ingresses []v1.LoadBalancerIngress, path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("failed creating file '%s': %s", path, err)
		return err
	}
	datawriter := bufio.NewWriter(file)
	datawriter.WriteString(c.getSecurityProtocolMap())
	datawriter.Flush()
	file.Close()
	log.Infof("created the %s file", path)
	return nil
}
func (c *KafkaService) getNodeExternalIP() (string, error) {
	node, err := c.Client.CoreV1().Nodes().Get(c.Env.GetNodeName(), metav1.GetOptions{})
	if err != nil {
		log.Errorf("error fetching nodes external IP :%s", err)
		return "", err
	}
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP {
			return address.Address, nil
		}
	}
	return "", fmt.Errorf("no node found with name '%s'", c.Env.GetNodeName())
}
func (c *KafkaService) writeListenerDNS(ingresses []v1.LoadBalancerIngress, path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Errorf("failed creating file '%s': %s", path, err)
		return err
	}
	dataWriter := bufio.NewWriter(file)

	for _, ingress := range ingresses {
		if len(ingress.Hostname) > 0 {
			dataWriter.WriteString(ingress.Hostname)
		}
		if len(ingress.IP) > 0 {
			dataWriter.WriteString(ingress.IP)
		}
	}
	dataWriter.Flush()
	file.Close()
	log.Infof("created the %s file", path)
	return nil
}

func (c *KafkaService) getSecurityProtocolMap() string {
	securityMaps := os.Getenv("LISTENER_SECURITY_PROTOCOL_MAP")
	if len(securityMaps) > 0 {
		log.Infoln("detected internal LISTENER_SECURITY_PROTOCOL_MAP:  ", securityMaps)
		securityMapList := strings.Split(securityMaps, ",")
		for _, securityProtocolMap := range securityMapList {
			securityProtocolMapValues := strings.Split(securityProtocolMap, ":")
			if len(securityProtocolMapValues) != 2 {
				log.Infoln("cannot detect the security protocol type from:  ", securityProtocolMap)
			}
			if securityProtocolMapValues[0] == "INTERNAL" {
				return fmt.Sprintf("%s:%s", EXTERNAL_INGRESS_PROTOCOL_NAME, securityProtocolMapValues[1])
			}
		}
	}
	log.Infoln("no 'INTERNAL' value for LISTENER_SECURITY_PROTOCOL_MAP detected")
	return ""
}

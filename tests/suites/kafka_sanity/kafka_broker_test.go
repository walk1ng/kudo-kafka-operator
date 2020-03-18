package kafka_sanity

import (
	"fmt"
	"os"
	"testing"

	. "github.com/mesosphere/kudo-kafka-operator/tests/suites"
	"github.com/mesosphere/kudo-kafka-operator/tests/utils"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	topicName       string
	customNamespace = "kafka-sanity-test"
	repoRoot, _     = os.LookupEnv("REPO_ROOT")
	resources       = "tests/suites/kafka_sanity/resources"
	utilsContainer  = "alpine"
)

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Sanity Checks]", func() {
		Context("default installation", func() {
			It("service should have count 1", func() {
				kudoInstances, _ := utils.KClient.GetInstancesInNamespace(customNamespace)
				for _, instance := range kudoInstances.Items {
					log.Printf("%s kudo instance in namespace %s ", instance.Name, instance.Namespace)
					log.Printf("%s kudo instance with parameters %v ", instance.Name, instance.Spec.Parameters)
				}
				Expect(utils.KClient.GetServicesCount("kafka-svc", customNamespace)).To(Equal(1))
			})
			It("statefulset should have 3 replicas", func() {
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("statefulset should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("Create a topic and read and write a message", func() {
				kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
					Namespace: utils.String(customNamespace),
				})
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(0), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(1), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(2), DefaultContainerName, 100)
				topicSuffix, _ := utils.GetRandString(6)
				topicName = fmt.Sprintf("test-topic-%s", topicSuffix)
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				out, err := kafkaClient.CreateTopic(GetBrokerPodName(0), DefaultContainerName, topicName, "0")
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("Created topic"))
				messageToTest := "CheckThisMessage"
				_, err = kafkaClient.WriteInTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
			})
		})

		Context("service access", func() {
			kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
				Namespace: utils.String(customNamespace),
			})
			It("should have dns resolution using service", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				output, err := kafkaClient.ExecInPod(customNamespace, utilsContainer, utilsContainer,
					[]string{"nslookup", fmt.Sprintf("kafka-svc.%s.svc.cluster.local", customNamespace)})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring(fmt.Sprintf("kafka-kafka-0.kafka-svc.%s.svc.cluster.local", customNamespace)))
				Expect(output).To(ContainSubstring(fmt.Sprintf("kafka-kafka-1.kafka-svc.%s.svc.cluster.local", customNamespace)))
				Expect(output).To(ContainSubstring(fmt.Sprintf("kafka-kafka-2.kafka-svc.%s.svc.cluster.local", customNamespace)))
			})
			It("each pod should be accessible", func() {
				output, err := kafkaClient.ExecInPod(customNamespace, utilsContainer, utilsContainer,
					[]string{"nslookup", fmt.Sprintf("kafka-kafka-0.kafka-svc.%s.svc.cluster.local", customNamespace)})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("Address 1:"))
				output, err = kafkaClient.ExecInPod(customNamespace, utilsContainer, utilsContainer,
					[]string{"nslookup", fmt.Sprintf("kafka-kafka-1.kafka-svc.%s.svc.cluster.local", customNamespace)})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("Address 1:"))
				output, err = kafkaClient.ExecInPod(customNamespace, utilsContainer, utilsContainer,
					[]string{"nslookup", fmt.Sprintf("kafka-kafka-2.kafka-svc.%s.svc.cluster.local", customNamespace)})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("Address 1:"))
			})
			It("Check for metrics endpoint", func() {
				out, err := kafkaClient.ExecInPod(customNamespace, GetBrokerPodName(0), DefaultContainerName, []string{"bash", "-c", "curl -v --silent localhost:9094 2>&1"})
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("kafka_controller_ControllerStats_LeaderElectionRateAndTimeMs"))
			})
		})

		Context("storage options", func() {
			kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
				Namespace: utils.String(customNamespace),
			})
			It("verify that access mode is ReadWriteOnce", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				sts, err := utils.KClient.AppsV1().StatefulSets(customNamespace).Get(DefaultKafkaStatefulSetName, metav1.GetOptions{})
				Expect(err).To(BeNil())
				Expect(len(sts.Spec.VolumeClaimTemplates)).To(BeNumerically("==", 1))
				for _, pvc := range sts.Spec.VolumeClaimTemplates {
					for _, accessMode := range pvc.Spec.AccessModes {
						Expect(accessMode).To(Equal(v1.ReadWriteOnce))
					}
				}
			})
			It("verify all configmaps are mounted", func() {
				output, err := kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", DefaultContainerName,
					[]string{"findmnt", "/bootstrap"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/bootstrap"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", DefaultContainerName,
					[]string{"findmnt", "/metrics"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/metrics"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", DefaultContainerName,
					[]string{"findmnt", "/health-check-script"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/health-check-script"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", DefaultContainerName,
					[]string{"findmnt", "/config"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/config"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", DefaultContainerName,
					[]string{"findmnt", "/var/lib/kafka"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("/var/lib/kafka"))
			})
		})

		Context("scale and write/read in topics", func() {
			It("statefulset should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("write in broker-3 a message with replication 3 and scale down to 3 brokers and read message from another broker", func() {
				kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
					Namespace: utils.String(customNamespace),
				})
				topicSuffix, _ := utils.GetRandString(6)
				topicName := fmt.Sprintf("test-topic-%s", topicSuffix)
				err := utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, customNamespace, 4)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 4, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(3), DefaultContainerName, 100)
				out, err := kafkaClient.CreateTopic(GetBrokerPodName(3), DefaultContainerName, topicName, "0:1:2")
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("Created topic"))
				messageToTest := "ReplicatedMessage"
				_, err = kafkaClient.WriteInTopic(GetBrokerPodName(3), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(3), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
				err = utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, customNamespace, 3)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, 240)
				// in case the broker with id 3 was the active controller we should wait for the new active controller
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(0), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(1), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(2), DefaultContainerName, 100)
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
			})
		})

		Context("change the instance BROKER_COUNT param from 3 to 1 and from 1 to 3", func() {
			It("should have 3 replicas", func() {
				err := utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, customNamespace, 1)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, customNamespace, 3)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace, 3, 240)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
		})

		Context("Update LOG_RETENTION_HOURS param from default 168 to 200", func() {
			kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
				Namespace: utils.String(customNamespace),
			})
			It("LOG_RETENTION_HOURS should change from 168 to 200", func() {
				err := utils.KClient.WaitForStatus(DefaultKudoKafkaInstance, customNamespace, v1beta1.ExecutionComplete, 300)
				Expect(err).To(BeNil())
				currentParamVal, _ := utils.KClient.GetParamForKudoInstance(DefaultKudoKafkaInstance, customNamespace, "LOG_RETENTION_HOURS")
				log.Printf("Current Parameter %s value is : %s ", "LOG_RETENTION_HOURS", currentParamVal)
				updatedParamVal := "200"
				err = utils.KClient.UpdateInstanceParams(DefaultKudoKafkaInstance, customNamespace, map[string]string{"LOG_RETENTION_HOURS": updatedParamVal})
				Expect(err).To(BeNil())
				Expect(updatedParamVal).NotTo(Equal(currentParamVal))
				newValue, _ := utils.KClient.GetParamForKudoInstance(DefaultKudoKafkaInstance, customNamespace, "LOG_RETENTION_HOURS")
				Expect(updatedParamVal).To(BeEquivalentTo(newValue))
				err = utils.KClient.WaitForStatus(DefaultKudoKafkaInstance, customNamespace, v1beta1.ExecutionInProgress, 300)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatus(DefaultKudoKafkaInstance, customNamespace, v1beta1.ExecutionComplete, 300)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetParamForKudoInstance(DefaultKudoKafkaInstance, customNamespace, "LOG_RETENTION_HOURS")).To(Equal("200"))
			})
			It("statefulset should have 3 replicas", func() {
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("Check parameter value again", func() {
				out, err := kafkaClient.ExecInPod(customNamespace, GetBrokerPodName(2), DefaultContainerName,
					[]string{"grep", "log.retention.hours", "/var/lib/kafka/data/server.log"})
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(fmt.Sprintf("%s = %s", "log.retention.hours", "200")))
			})
		})

	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.CreateNamespace(customNamespace, false)
	utils.Apply(fmt.Sprintf("%s/%s", repoRoot, resources), customNamespace)
	utils.Setup(customNamespace)
})

var _ = AfterSuite(func() {
	utils.TearDown(customNamespace)
	utils.Delete(fmt.Sprintf("%s/%s", repoRoot, resources), customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-brokers"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaBrokers Suite", []Reporter{junitReporter})
}

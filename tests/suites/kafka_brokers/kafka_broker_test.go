package kafka_brokers

import (
	"fmt"
	"testing"

	. "github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"
)

var topicName string

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Sanity Checks]", func() {
		Context("default installation", func() {
			It("service should have count 1", func() {
				kudoInstances, _ := utils.KClient.GetInstancesInNamespace(TestNamespace)
				for _, instance := range kudoInstances.Items {
					log.Printf("%s kudo instance in namespace %s ", instance.Name, instance.Namespace)
					log.Printf("%s kudo instance with parameters %v ", instance.Name, instance.Spec.Parameters)
				}
				Expect(utils.KClient.GetServicesCount("kafka-svc", TestNamespace)).To(Equal(1))
			})
			It("statefulset should have 3 replicas", func() {
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, TestNamespace)).To(Equal(3))
			})
			It("statefulset should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, TestNamespace, 3, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, TestNamespace, 3, 240)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, TestNamespace)).To(Equal(3))
			})
			It("Create a topic and read and write a message", func() {
				kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
					Namespace: utils.String(TestNamespace),
				})

				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(0), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(1), DefaultContainerName, 100)
				kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(2), DefaultContainerName, 100)
				topicSuffix, _ := utils.GetRandString(6)
				topicName = fmt.Sprintf("test-topic-%s", topicSuffix)
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, TestNamespace, 3, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, TestNamespace, 3, 240)
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
		Context("change the instance BROKER_COUNT param from 3 to 1 and from 1 to 3", func() {
			It("should have 3 replicas", func() {
				err := utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, TestNamespace, 1)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, TestNamespace, 3, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, TestNamespace, 1, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.UpdateInstancesCount(DefaultKudoKafkaInstance, TestNamespace, 3)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetCount(DefaultKafkaStatefulSetName, TestNamespace, 3, 30)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, TestNamespace)).To(Equal(3))
			})
			It("should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, TestNamespace, 3, 240)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, TestNamespace)).To(Equal(3))
			})
		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(TestNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.CreateNamespace(TestNamespace, false)
	utils.Setup(TestNamespace)
})

var _ = AfterSuite(func() {
	utils.TearDown(TestNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-brokers"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaBrokers Suite", []Reporter{junitReporter})
}

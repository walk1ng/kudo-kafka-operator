package kafkaupgrade

import (
	"fmt"
	"os"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

	. "github.com/mesosphere/kudo-kafka-operator/tests/suites"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
)

var (
	topicName              string
	customNamespace        = "kafka-upgrade-test"
	defaultOperatorVersion = "1.1.1"
	kafkaClient            = utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
		Namespace: utils.String(customNamespace),
	})
)

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Upgrade Checks]", func() {
		Context("default installation", func() {
			It("service should have count 1", func() {
				err := utils.KClient.WaitForStatus(utils.KafkaInstance, customNamespace, v1beta1.ExecutionComplete, 300)
				Expect(err).To(BeNil())
				kudoInstances, _ := utils.KClient.GetInstancesInNamespace(customNamespace)
				for _, instance := range kudoInstances.Items {
					log.Printf("%s kudo instance in namespace %s ", instance.Name, instance.Namespace)
					log.Printf("%s kudo instance with parameters %v ", instance.Name, instance.Spec.Parameters)
				}
				Expect(utils.KClient.GetServicesCount("kafka-svc", customNamespace)).To(Equal(1))
			})
			It("statefulset should have 1 replicas", func() {
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(1))
			})
			It("statefulset should have 1 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(1))
			})
			It("Create a topic and write a message", func() {
				err := kafkaClient.WaitForBrokersToBeRegisteredWithService(GetBrokerPodName(0), DefaultContainerName, 100)
				Expect(err).To(BeNil())
				topicSuffix, _ := utils.GetRandString(6)
				topicName = fmt.Sprintf("test-topic-%s", topicSuffix)
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
				Expect(err).To(BeNil())
				out, err := kafkaClient.CreateTopic(GetBrokerPodName(0), DefaultContainerName, topicName, "")
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("Created topic"))
				messageToTest := "CheckThisMessage"
				_, err = kafkaClient.WriteInTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
			})
		})
		Context("upgrade operator", func() {
			It("operator version should change", func() {
				currentOperatorVersion, _ := utils.KClient.GetOperatorVersionForKudoInstance(utils.KafkaInstance, customNamespace)
				utils.KClient.UpgardeInstanceFromPath(os.Getenv(utils.KafkaFrameworkDirEnv), customNamespace, utils.KafkaInstance, map[string]string{})
				err := utils.KClient.WaitForStatus(utils.KafkaInstance, customNamespace, v1beta1.ExecutionComplete, 300)
				Expect(err).To(BeNil())
				newOperatorVersion, _ := utils.KClient.GetOperatorVersionForKudoInstance(utils.KafkaInstance, customNamespace)
				Expect(newOperatorVersion).NotTo(BeNil())
				Expect(newOperatorVersion).NotTo(Equal(currentOperatorVersion))
			})
			It("statefulset should have 1 replicas", func() {
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(1))
			})
			It("statefulset should have 1 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(1))
			})
			It("read a message", func() {
				messageToTest := "CheckThisMessage"
				out, err := kafkaClient.ReadFromTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
			})
		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	_, err := utils.KClient.CreateNamespace(customNamespace, false)
	Expect(err).To(BeNil())
	utils.InstallKudoOperator(customNamespace, utils.ZkInstance, utils.ZkFrameworkDirEnv, map[string]string{
		"MEMORY":     "256Mi",
		"CPUS":       "0.25",
		"NODE_COUNT": "1",
	})
	err = utils.KClient.WaitForStatefulSetCount(DefaultZkStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
	Expect(err).To(BeNil())
	utils.KClient.InstallOperatorFromRepository(customNamespace, "kafka", utils.KafkaInstance, defaultOperatorVersion, map[string]string{
		"BROKER_MEM":                       "512Mi",
		"BROKER_CPUS":                      "0.25",
		"ZOOKEEPER_URI":                    "zookeeper-instance-zookeeper-0.zookeeper-instance-hs:2181",
		"BROKER_COUNT":                     "1",
		"OFFSETS_TOPIC_REPLICATION_FACTOR": "1",
	})
	err = utils.KClient.WaitForStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace, 1, utils.DefaultStatefulReadyWait)
	Expect(err).To(BeNil())
	utils.KClient.LogObjectsOfKinds(customNamespace, []string{"svc", "pdb", "operatorversions", "operators", "instance"})
})

var _ = AfterSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-upgrade"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaUpgrade Suite", []Reporter{junitReporter})
}

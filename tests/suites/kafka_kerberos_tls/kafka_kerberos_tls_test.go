package kafka_kerberos

import (
	"fmt"
	"testing"

	. "github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var (
	customNamespace = "kerberos-tls-ns"
	krb5Client      = &utils.KDCClient{
		Namespace: customNamespace,
	}
	zkNodeCount      = 1
	kafkaBrokerCount = 1
)

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Kerberos TLS Checks]", func() {
		Context("kerberos-tls-ns installation", func() {
			kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
				Namespace:       utils.String(customNamespace),
				KerberosEnabled: true,
				TLSEnabled:      true,
			})
			It("kdc service should have count 1", func() {
				Expect(utils.KClient.WaitForContainerToBeReady("kdc", "kdc", customNamespace, 200)).To(BeNil())
				Expect(utils.KClient.CheckIfPodExists("kdc", customNamespace)).To(Equal(true))
				Expect(utils.KClient.GetServicesCount("kdc-service", customNamespace)).To(Equal(1))
				utils.KClient.PrintLogsOfPod("kdc", "kdc", customNamespace)
				Expect(krb5Client.CreateKeytabSecret(utils.GetKafkaKeyTabs(1, customNamespace), "kafka", "base64-kafka-keytab-secret")).To(BeNil())
			})
			It("Kafka and Zookeeper statefulset should have 1 replica each with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, zkNodeCount, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, kafkaBrokerCount, utils.DefaultStatefulReadyWaitSeconds)
				Expect(err).To(BeNil())
			})
			It("verify the SSL listener", func() {
				output, err := kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", DefaultContainerName,
					[]string{"grep", "ListenerName", "/var/lib/kafka/data/server.log"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("ListenerName(INTERNAL),SASL_SSL"))
			})
			It("write and read a message with replication 1 in broker-0", func() {
				topicSuffix, _ := utils.GetRandString(6)
				topicName := fmt.Sprintf("test-topic-%s", topicSuffix)
				out, err := kafkaClient.CreateTopic(GetBrokerPodName(0), DefaultContainerName, topicName, "0")
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("Created topic"))
				messageToTest := "KerberosTLSMessage"
				_, err = kafkaClient.WriteInTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(0), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring(messageToTest))
			})
		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.CreateNamespace(customNamespace, false)
	utils.KClient.CreateTLSCertSecret(customNamespace, "kafka-tls", "Kafka")
	Expect(krb5Client.Deploy()).To(BeNil())
	utils.SetupWithKerberos(customNamespace, true)
})

var _ = AfterSuite(func() {
	utils.TearDown(customNamespace)
	Expect(krb5Client.TearDown()).To(BeNil())
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.DeleteNamespace(customNamespace)
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-kerberos-tls"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaKerberosTLS Suite", []Reporter{junitReporter})
}

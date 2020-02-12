package kafka_kerberos

import (
	"fmt"
	"testing"
	"os"

	. "github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	client "github.com/kudobuilder/test-tools/pkg/client"
	testtools "github.com/kudobuilder/test-tools/pkg/kudo"
)

var (
	customNamespace = "kerberos-ns"
	krb5Client      = &utils.KDCClient{
		Namespace: customNamespace,
	}
	clienttools, e = client.NewForConfig(os.Getenv("KUBECONFIG"))
	OperatorZk  = testtools.Operator{}
	OperatorKafka  = testtools.Operator{}
)

var _ = Describe("KafkaTest", func() {
	Describe("[Kafka Kerberos Checks]", func() {
		Context("kerberos-ns installation", func() {
			It("kdc service should have count 1", func() {
				Expect(utils.KClient.WaitForContainerToBeReady("kdc", "kdc", customNamespace, 200)).To(BeNil())
				Expect(utils.KClient.CheckIfPodExists("kdc", customNamespace)).To(Equal(true))
				Expect(utils.KClient.GetServicesCount("kdc-service", customNamespace)).To(Equal(1))
				utils.KClient.PrintLogsOfPod("kdc", "kdc", customNamespace)
				Expect(krb5Client.CreateKeytabSecret(utils.GetKafkaKeyabs(customNamespace), "kafka", "base64-kafka-keytab-secret")).To(BeNil())
				// Kafka deployPlan is expected not to be completed as it has a dependency over "base64-kafka-keytab-secret" to be created by KDC client
				// Checking if deployplan completes now.
				err := OperatorKafka.Instance.WaitForPlanComplete("deploy")
				Expect(err).To(BeNil())
			})
			It("Kafka and Zookeeper statefulset should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, 300)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("write and read a message with replication 3 in broker-0", func() {
				kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
					Namespace:       utils.String(customNamespace),
					KerberosEnabled: true,
				})
				topicSuffix, _ := utils.GetRandString(6)
				topicName := fmt.Sprintf("test-topic-%s", topicSuffix)
				out, err := kafkaClient.CreateTopic(GetBrokerPodName(1), DefaultContainerName, topicName, "1:0:2")
				Expect(err).To(BeNil())
				Expect(out).To(ContainSubstring("Created topic"))
				messageToTest := "KerberosMessage"
				_, err = kafkaClient.WriteInTopic(GetBrokerPodName(1), DefaultContainerName, topicName, messageToTest)
				Expect(err).To(BeNil())
				out, err = kafkaClient.ReadFromTopic(GetBrokerPodName(1), DefaultContainerName, topicName, messageToTest)
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
	Expect(krb5Client.Deploy()).To(BeNil())
	//utils.SetupWithKerberos(customNamespace)
	OperatorZk, e = testtools.InstallOperator(os.Getenv(utils.ZK_FRAMEWORK_DIR_ENV)).
			WithNamespace(customNamespace).
			WithInstance(utils.ZK_INSTANCE).
			Do(clienttools)
	Expect(e).To(BeNil())
	err := OperatorZk.Instance.WaitForPlanInProgress("deploy")
	Expect(err).To(BeNil())
	err = OperatorZk.Instance.WaitForPlanComplete("deploy")
	Expect(err).To(BeNil())
	utils.KClient.WaitForStatefulSetCount(DefaultZkStatefulSetName, customNamespace, 3, 30)
	OperatorKafka, e = testtools.InstallOperator(os.Getenv(utils.KAFKA_FRAMEWORK_DIR_ENV)).
		WithNamespace(customNamespace).
		WithInstance(utils.KAFKA_INSTANCE).
		WithParameters(map[string]string{
			"KERBEROS_ENABLED":       "true",
			"KERBEROS_KDC_HOSTNAME":  "kdc-service",
			"KERBEROS_KDC_PORT":      "2500",
			"KERBEROS_KEYTAB_SECRET": "base64-kafka-keytab-secret",
		}).
		Do(clienttools)
	Expect(e).To(BeNil())
	err = OperatorKafka.Instance.WaitForPlanInProgress("deploy")
	Expect(err).To(BeNil())
	utils.KClient.WaitForStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace, 3, 30)
})

var _ = AfterSuite(func() {
	//utils.TearDown(customNamespace)
	err := OperatorZk.Uninstall()
	Expect(err).To(BeNil())
	utils.KClient.WaitForStatefulSetCount(DefaultZkStatefulSetName, customNamespace, 0, 30)
	err = OperatorKafka.Uninstall()
	Expect(err).To(BeNil())
	utils.KClient.WaitForStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace, 0, 30)
	Expect(krb5Client.TearDown()).To(BeNil())
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.DeleteNamespace(customNamespace)
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-kerberos"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaKerberos Suite", []Reporter{junitReporter})
}

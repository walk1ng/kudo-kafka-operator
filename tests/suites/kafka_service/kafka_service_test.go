package kafka_service

import (
	"fmt"
	"os"
	"testing"

	"github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
)

var (
	customNamespace = "kafka-service-test"
	repoRoot, _     = os.LookupEnv("REPO_ROOT")
	resources       = "tests/suites/kafka_service/resources"
	utilsContainer  = "alpine"
)

var _ = Describe("KafkaService", func() {
	Describe("[Kafka Service]", func() {
		Context("service access", func() {
			kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
				Namespace: utils.String(customNamespace),
			})
			It("should have dns resolution using service", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(suites.DefaultKafkaStatefulSetName, customNamespace, 3, 300)
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
		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(customNamespace)
	utils.KClient.CreateNamespace(customNamespace, false)
	utils.Apply(fmt.Sprintf("%s/%s", repoRoot, resources), customNamespace)
	utils.Setup(customNamespace)
})

var _ = AfterSuite(func() {
	utils.TearDown(customNamespace)
	utils.Delete(fmt.Sprintf("%s/%s", repoRoot, resources), customNamespace)
	utils.KClient.DeleteNamespace(customNamespace)
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-service"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaService Suite", []Reporter{junitReporter})
}

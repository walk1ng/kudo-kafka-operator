package kafka_storage

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/mesosphere/kudo-kafka-operator/tests/suites"

	"github.com/mesosphere/kudo-kafka-operator/tests/utils"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	customNamespace = "kafka-storage-test"
)

var _ = Describe("KafkaStorage", func() {
	Describe("[Kafka Storage]", func() {
		Context("storage options", func() {
			kafkaClient := utils.NewKafkaClient(utils.KClient, &utils.KafkaClientConfiguration{
				Namespace: utils.String(customNamespace),
			})
			It("verify that access mode is ReadWriteOnce", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(suites.DefaultKafkaStatefulSetName, customNamespace, 3, 300)
				Expect(err).To(BeNil())
				sts, err := utils.KClient.AppsV1().StatefulSets(customNamespace).Get(suites.DefaultKafkaStatefulSetName, metav1.GetOptions{})
				Expect(err).To(BeNil())
				Expect(len(sts.Spec.VolumeClaimTemplates)).To(BeNumerically("==", 1))
				for _, pvc := range sts.Spec.VolumeClaimTemplates {
					for _, accessMode := range pvc.Spec.AccessModes {
						Expect(accessMode).To(Equal(v1.ReadWriteOnce))
					}
				}
			})
			It("verify all configmaps are mounted", func() {
				output, err := kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"findmnt", "/bootstrap"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/bootstrap"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"findmnt", "/metrics"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/metrics"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"findmnt", "/health-check-script"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/health-check-script"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"findmnt", "/config"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("kubernetes.io~configmap/config"))
				output, err = kafkaClient.ExecInPod(customNamespace, "kafka-kafka-0", suites.DefaultContainerName,
					[]string{"findmnt", "/var/lib/kafka"})
				Expect(err).To(BeNil())
				Expect(output).To(ContainSubstring("/var/lib/kafka"))
			})

		})
	})
})

var _ = BeforeSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.CreateNamespace(customNamespace, false)
	utils.Setup(customNamespace)
})

var _ = AfterSuite(func() {
	utils.TearDown(customNamespace)
	Expect(utils.DeletePVCs("data-dir")).To(BeNil())
	utils.KClient.DeleteNamespace(customNamespace)
})

func TestService(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-storage"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaStorage Suite", []Reporter{junitReporter})
}

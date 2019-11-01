package kafka_install

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

var customNamespace = "custom-ns"

var _ = Describe("KafkaInstallTest", func() {
	Describe("[Kafka Custom Namespace]", func() {
		Context("custom-ns installation", func() {
			It("custom-ns service should have count 1", func() {
				kudoInstances, _ := utils.KClient.GetInstancesInNamespace(customNamespace)
				for _, instance := range kudoInstances.Items {
					log.Printf("%s kudo instance in namespace %s ", instance.Name, instance.Namespace)
					log.Printf("%s kudo instance with parameters %v ", instance.Name, instance.Spec.Parameters)
				}
				Expect(utils.KClient.GetServicesCount("kafka-svc", customNamespace)).To(Equal(1))
			})
			It("custom-ns statefulset should have 3 replicas", func() {
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
			})
			It("custom-ns statefulset should have 3 replicas with status READY", func() {
				err := utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultZkStatefulSetName, customNamespace, 3, 240)
				Expect(err).To(BeNil())
				err = utils.KClient.WaitForStatefulSetReadyReplicasCount(DefaultKafkaStatefulSetName, customNamespace, 3, 240)
				Expect(err).To(BeNil())
				Expect(utils.KClient.GetStatefulSetCount(DefaultKafkaStatefulSetName, customNamespace)).To(Equal(3))
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
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-install"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaInstall Suite", []Reporter{junitReporter})
}

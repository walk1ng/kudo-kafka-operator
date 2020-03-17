package config

import (
	"fmt"
	"testing"

	"github.com/mesosphere/kudo-kafka-operator/images/kafka/kafka-connectors-setup/pkgs/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var _ = Describe("[Kafka KafkaService]", func() {

	var (
		mockCtrl      *gomock.Controller
		mockUtils     *mocks.MockUtils
		configuration *ConfigurationSetup
	)

	Context("Kafka Connectors Configuration", func() {
		It("Tests sample configuration", func() {
			configuration.RegisterConnectors("http://www.sample.endpoint")
			configuration.DownloadConnectorResources("/tmp")
			configuration.DownloadResources("/tmp")
		})
	})

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockUtils = mocks.NewMockUtils(mockCtrl)

		sampleConfig := `{
			"connectors": {
				"sample-connector-1": {
					"resources": [
						"http://foo.bar/resource1.zip",
						"http://foo.bar/resource2.zip"
					],
					"config": {
						"name": "sample-connector-1",
						"config": {
							"connector.class": "foo.bar",
							"tasks.max": "1",
							"topics": "sample_connector_1_topic"
						}
					}
				}
			},
			"resources": [
				"http://foo.bar/resource0.zip"
			]
		}`
		configuration = &ConfigurationSetup{
			Utils:      mockUtils,
			ConfigFile: &ConfigFile{},
		}
		err := yaml.Unmarshal([]byte(sampleConfig), configuration.ConfigFile)
		Expect(err).To(BeNil())

		mockUtils.EXPECT().RegisterConnector("http://www.sample.endpoint", configuration.ConfigFile.Connectors["sample-connector-1"].Config).Return(nil).AnyTimes()
		mockUtils.EXPECT().DownloadFile("/tmp", "http://foo.bar/resource1.zip").Return("resource1.zip", nil).AnyTimes()
		mockUtils.EXPECT().ExtractFile("/tmp/resource1.zip", "/tmp").Return(nil).AnyTimes()
		mockUtils.EXPECT().DownloadFile("/tmp", "http://foo.bar/resource2.zip").Return("resource2.zip", nil).AnyTimes()
		mockUtils.EXPECT().ExtractFile("/tmp/resource2.zip", "/tmp").Return(nil).AnyTimes()
		mockUtils.EXPECT().DownloadFile("/tmp", "http://foo.bar/resource0.zip").Return("resource0.zip", nil).AnyTimes()
		mockUtils.EXPECT().ExtractFile("/tmp/resource0.zip", "/tmp").Return(nil).AnyTimes()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})
})

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("%s-junit.xml", "kafka-connectors-setup"))
	RunSpecsWithDefaultAndCustomReporters(t, "KafkaConnectorsSetup Suite", []Reporter{junitReporter})
}

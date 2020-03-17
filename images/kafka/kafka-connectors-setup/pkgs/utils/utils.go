package utils

//go:generate mockgen -destination=../mocks/utils_mock.go -package=mocks github.com/mesosphere/kudo-kafka-operator/images/kafka/kafka-connectors-setup/pkgs/utils Utils

// Utils interface
type Utils interface {
	DownloadFile(downloadDirectory, url string) (string, error)
	ExtractFile(filepath, destination string) error
	RegisterConnector(endpoint string, data interface{}) error
}

// UtilsImpl struct
type UtilsImpl struct{}

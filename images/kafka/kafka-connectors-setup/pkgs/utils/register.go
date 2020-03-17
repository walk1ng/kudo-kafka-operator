package utils

import (
	"bytes"
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

// RegisterConnector register connectors to kafka-connect endpoint
func (u *UtilsImpl) RegisterConnector(endpoint string, data interface{}) error {
	if data == nil {
		return fmt.Errorf("register data is empty for endpoint %s", endpoint)
	}
	payloadBytes, err := jsoniter.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", endpoint, "connectors"), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

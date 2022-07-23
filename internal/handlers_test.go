package internal

import (
	"io/ioutil"
	"net/http"
	"testing"
)

func TestV1HelloWorld(t *testing.T) {
	expectedResponse := `{"message":"Hello, World!"}`
	// Create a request to our api
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8000/v1/", nil)
	if err != nil {
		t.Error("Error creating request")
	}
	req.Header.Set("test", "key")
	resp, err := client.Do(req)
	if err != nil {
		t.Error("Error sending request")
	}
	defer resp.Body.Close()

	// check the response code
	if resp.StatusCode != 200 {
		t.Error("Error response status code, expected 200, got", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error("Error reading response body")
	}
	// check the response body
	if string(body) != expectedResponse {
		t.Error("Error response body got: ", string(body), " expected: ", expectedResponse)
	}
}

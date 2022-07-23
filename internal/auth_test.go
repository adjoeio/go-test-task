package internal

import (
	"net/http"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8000/v1/", nil)
	if err != nil {
		t.Error("Error creating request")
	}
	t.Run("should return 401 if token is not provided", func(t *testing.T) {
		req.Header.Set("test", "")
		resp, err := client.Do(req)
		if err != nil {
			t.Error("Error sending request")
		}
		defer resp.Body.Close()
		if resp.StatusCode != 401 {
			t.Error("Error response status code")
		}
	})
	t.Run("should return 200 if token is provided", func(t *testing.T) {
		req.Header.Set("test", "key")
		resp, err := client.Do(req)
		if err != nil {
			t.Error("Error sending request")
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Error("Error response status code")
		}
	})
}

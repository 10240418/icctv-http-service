package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"icctv-http-service/services"
)

func TestPathWritesReturnBadRequestForInvalidNames(t *testing.T) {
	controller := NewOrangePiController(services.NewOrangePiService(nil, nil))
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		for _, name := range []string{"SO_MAIN", "channel_SO_MAIN", "channel01", "Channel1", "channel0"} {
			t.Run(method+"/"+name, func(t *testing.T) {
				body := `{"config":{"source":"rtsp://camera.example/SO_MAIN"}}`
				target := "/api/orangepi/remote/paths?id=1&token=test-token&name=" + name
				if method == http.MethodPost {
					body = `{"name":"` + name + `","config":{"source":"rtsp://camera.example/SO_MAIN"}}`
				}
				request := httptest.NewRequest(method, target, strings.NewReader(body))
				response := httptest.NewRecorder()
				if method == http.MethodPost {
					controller.RemoteAddMediaMTXPath(response, request)
				} else {
					controller.RemoteUpdateMediaMTXPath(response, request)
				}
				if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "positive integer") {
					t.Fatalf("expected 400 for %s: %d %s", name, response.Code, response.Body.String())
				}
			})
		}
	}
}

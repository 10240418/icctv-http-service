package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"icctv-http-service/controllers"
	"icctv-http-service/middlewares"
	"icctv-http-service/services"
)

func TestRTSPSourceRoutesRequireAdmin(t *testing.T) {
	mux := http.NewServeMux()
	SetupRoutes(mux, ControllerSet{
		RTSPSource: controllers.NewRTSPSourceController(nil),
	}, MiddlewareSet{
		Auth: middlewares.NewAuthMiddleware(services.NewAuthService(nil, nil, nil, nil)),
	})
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		for _, bearer := range []string{"", "Bearer invalid"} {
			request := httptest.NewRequest(method, "/api/rtsp-sources?id=1", nil)
			request.Header.Set("Authorization", bearer)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s accepted invalid admin authorization: %d", method, response.Code)
			}
		}
	}
}

package routes

import (
	"net/http"

	"icctv-http-service/controllers"
	"icctv-http-service/middlewares"
)

// ControllerSet 聚合所有控制器
type ControllerSet struct {
	Auth      *controllers.AuthController
	Admin     *controllers.AdminController
	OrangePi  *controllers.OrangePiController
	Building  *controllers.BuildingController
	Device    *controllers.DeviceController
	PublicNet *controllers.PublicNetController
	NVR       *controllers.NVRController
}

// MiddlewareSet 聚合所有中间件
type MiddlewareSet struct {
	Auth *middlewares.AuthMiddleware
}

// SetupRoutes 注册路由
func SetupRoutes(mux *http.ServeMux, ctrl ControllerSet, mw MiddlewareSet) {
	requireAdmin := func(handler http.HandlerFunc) http.HandlerFunc {
		if mw.Auth != nil {
			return mw.Auth.RequireAdmin(handler)
		}
		return handler
	}

	// Auth
	mux.HandleFunc("POST /api/auth/public", ctrl.Auth.PublicToken)       // 生成视频访问 Token (24小时有效)
	mux.HandleFunc("POST /api/auth/public/v2", ctrl.Auth.PublicTokenV2)  // 生成带频道备注的 Token
	mux.HandleFunc("POST /api/auth/permanent", ctrl.Auth.PermanentToken) // 生成永久视频访问 Token
	mux.HandleFunc("POST /api/auth/login", ctrl.Auth.Login)              // 管理员登录

	// Admin
	mux.HandleFunc("GET /api/admin", requireAdmin(ctrl.Admin.List))
	mux.HandleFunc("POST /api/admin", requireAdmin(ctrl.Admin.Create))
	mux.HandleFunc("PUT /api/admin", requireAdmin(ctrl.Admin.Update))
	mux.HandleFunc("DELETE /api/admin", requireAdmin(ctrl.Admin.Delete))

	// Building
	mux.HandleFunc("GET /api/building", requireAdmin(ctrl.Building.List))
	mux.HandleFunc("POST /api/building", requireAdmin(ctrl.Building.Create))
	mux.HandleFunc("PUT /api/building", requireAdmin(ctrl.Building.Update))
	mux.HandleFunc("DELETE /api/building", requireAdmin(ctrl.Building.Delete))

	// Device (OrangePi)
	mux.HandleFunc("GET /api/device", requireAdmin(ctrl.OrangePi.List))
	mux.HandleFunc("POST /api/device", requireAdmin(ctrl.OrangePi.Create))
	mux.HandleFunc("PUT /api/device", requireAdmin(ctrl.OrangePi.Update))
	mux.HandleFunc("DELETE /api/device", requireAdmin(ctrl.OrangePi.Delete))

	// OrangePi Remote
	mux.HandleFunc("POST /api/orangepi/remote/ports", requireAdmin(ctrl.OrangePi.RemoteUpdatePorts))
	mux.HandleFunc("GET /api/orangepi/remote/info", requireAdmin(ctrl.OrangePi.RemoteGetInfo))
	mux.HandleFunc("GET /api/orangepi/remote/health", requireAdmin(ctrl.OrangePi.RemoteHealthCheck))

	// OrangePi Remote MediaMTX Paths
	mux.HandleFunc("GET /api/orangepi/remote/paths", requireAdmin(ctrl.OrangePi.RemoteListMediaMTXPaths))
	mux.HandleFunc("GET /api/orangepi/remote/paths/detail", requireAdmin(ctrl.OrangePi.RemoteGetMediaMTXPath))
	mux.HandleFunc("POST /api/orangepi/remote/paths", requireAdmin(ctrl.OrangePi.RemoteAddMediaMTXPath))
	mux.HandleFunc("PATCH /api/orangepi/remote/paths", requireAdmin(ctrl.OrangePi.RemoteUpdateMediaMTXPath))
	mux.HandleFunc("DELETE /api/orangepi/remote/paths", requireAdmin(ctrl.OrangePi.RemoteDeleteMediaMTXPath))

	// NVR
	mux.HandleFunc("GET /api/nvr", requireAdmin(ctrl.NVR.List))
	mux.HandleFunc("POST /api/nvr", requireAdmin(ctrl.NVR.Create))
	mux.HandleFunc("PUT /api/nvr", requireAdmin(ctrl.NVR.Update))
	mux.HandleFunc("DELETE /api/nvr", requireAdmin(ctrl.NVR.Delete))

	// NVR 细粒度管理接口
	mux.HandleFunc("PUT /api/nvr/admin-user", requireAdmin(ctrl.NVR.UpdateAdminUser))
	mux.HandleFunc("PUT /api/nvr/users", requireAdmin(ctrl.NVR.UpdateUsers))
	mux.HandleFunc("PUT /api/nvr/rtsp-urls", requireAdmin(ctrl.NVR.UpdateRTSPUrls))
	mux.HandleFunc("POST /api/nvr/rtsp-url", requireAdmin(ctrl.NVR.AddRTSPUrl))
	mux.HandleFunc("DELETE /api/nvr/rtsp-url", requireAdmin(ctrl.NVR.RemoveRTSPUrl))
	mux.HandleFunc("POST /api/nvr/user", requireAdmin(ctrl.NVR.AddUser))
	mux.HandleFunc("DELETE /api/nvr/user", requireAdmin(ctrl.NVR.RemoveUser))

	// Building-OrangePi
	mux.HandleFunc("POST /api/bind/building-orangepi", requireAdmin(ctrl.Building.BindOrangePi))
	mux.HandleFunc("DELETE /api/bind/building-orangepi", requireAdmin(ctrl.Building.UnbindOrangePi))
	mux.HandleFunc("GET /api/bind/building-orangepi/{building_id}", requireAdmin(ctrl.Building.GetBuildingOrangePis))
	mux.HandleFunc("PUT /api/bind/building-orangepi/channels", requireAdmin(ctrl.Building.UpdateOrangePiChannels))

	// Building-NVR
	mux.HandleFunc("POST /api/bind/building-nvr", requireAdmin(ctrl.Building.BindNVR))
	mux.HandleFunc("DELETE /api/bind/building-nvr", requireAdmin(ctrl.Building.UnbindNVR))
	mux.HandleFunc("GET /api/bind/building-nvr/{building_id}", requireAdmin(ctrl.Building.GetBuildingNVRs))

	// Device info
	mux.HandleFunc("GET /api/device/info", requireAdmin(ctrl.Device.Summary))

	// Public net
	mux.HandleFunc("GET /api/publicnet/config", requireAdmin(ctrl.PublicNet.Get))
	mux.HandleFunc("PUT /api/publicnet/config", requireAdmin(ctrl.PublicNet.Update))
}

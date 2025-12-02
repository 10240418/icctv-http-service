package controllers

// PublicNetController Methods:
//0. NewPublicNetController(service *services.PublicNetService) -> 注入依赖
//1. Get(w http.ResponseWriter, r *http.Request) -> 获取公网出口 IP
//2. Update(w http.ResponseWriter, r *http.Request) -> 更新公网出口 IP

import (
	"net/http"

	"icctv-http-service/services"
)

// PublicNetControllerInterface 定义公网配置控制器能力
type PublicNetControllerInterface interface {
	Get(w http.ResponseWriter, r *http.Request)    //1.返回公网配置
	Update(w http.ResponseWriter, r *http.Request) //2.接收请求并更新公网配置
}

// PublicNetController 公网配置接口
type PublicNetController struct {
	service *services.PublicNetService
}

// 0. NewPublicNetController 构造控制器并注入 PublicNetService
func NewPublicNetController(service *services.PublicNetService) *PublicNetController {
	return &PublicNetController{service: service}
}

// 1. Get 读取现有公网配置
func (c *PublicNetController) Get(w http.ResponseWriter, r *http.Request) {
	config, err := c.service.Get(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, config)
}

// 2. Update 解析请求参数并调用服务更新公网配置
func (c *PublicNetController) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExternalIP string `json:"external_ip"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	config, err := c.service.Update(r.Context(), req.ExternalIP)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondData(w, http.StatusOK, config)
}

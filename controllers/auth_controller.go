package controllers

// AuthController Methods:
//0. NewAuthController(service *services.AuthService) -> 注入 AuthService
//1. PublicToken(w http.ResponseWriter, r *http.Request) -> 返回公开 Token
//2. Login(w http.ResponseWriter, r *http.Request) -> 管理员登录

import (
	"net/http"

	"icctv-http-service/services"
)

// AuthControllerInterface 定义认证接口能力
type AuthControllerInterface interface {
	PublicToken(w http.ResponseWriter, r *http.Request)    //1.公开 Token (24小时有效)
	PublicTokenV2(w http.ResponseWriter, r *http.Request)  //2.带频道备注的公开 Token
	PermanentToken(w http.ResponseWriter, r *http.Request) //3.永久 Token
	Login(w http.ResponseWriter, r *http.Request)          //4.管理员登录
}

// AuthController 认证接口
type AuthController struct {
	service *services.AuthService
}

// 0. NewAuthController 构造函数
func NewAuthController(service *services.AuthService) *AuthController {
	return &AuthController{service: service}
}

type publicTokenRequest struct {
	ISmartID string `json:"ismartid"` // 建筑ISmartID
	IsStaff  bool   `json:"is_staff"` // 是否为员工
}

// 1. PublicToken 生成视频访问 Token
func (c *AuthController) PublicToken(w http.ResponseWriter, r *http.Request) {
	var req publicTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ISmartID == "" {
		respondError(w, http.StatusBadRequest, "ismartid is required")
		return
	}

	result, err := c.service.GeneratePublicToken(r.Context(), req.ISmartID, req.IsStaff)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondData(w, http.StatusOK, result)
}

// PublicTokenV2 生成带频道备注和大厦级频道权限的视频访问 Token。
func (c *AuthController) PublicTokenV2(w http.ResponseWriter, r *http.Request) {
	var req publicTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ISmartID == "" {
		respondError(w, http.StatusBadRequest, "ismartid is required")
		return
	}
	result, err := c.service.GeneratePublicTokenV2(r.Context(), req.ISmartID, req.IsStaff)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

// 2. PermanentToken 生成永久视频访问 Token
func (c *AuthController) PermanentToken(w http.ResponseWriter, r *http.Request) {
	var req publicTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.ISmartID == "" {
		respondError(w, http.StatusBadRequest, "ismartid is required")
		return
	}

	result, err := c.service.GeneratePermanentToken(r.Context(), req.ISmartID, req.IsStaff)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondData(w, http.StatusOK, result)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 3. Login 管理员登录，返回 JWT
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := c.service.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	respondData(w, http.StatusOK, token)
}

package controllers

// OrangePiController Methods:
//0. NewOrangePiController(service *services.OrangePiService) -> 注入 OrangePiService
//1. List(w http.ResponseWriter, r *http.Request) -> 查询设备列表
//2. Create(w http.ResponseWriter, r *http.Request) -> 创建设备
//3. Update(w http.ResponseWriter, r *http.Request) -> 更新设备
//4. Delete(w http.ResponseWriter, r *http.Request) -> 删除设备

import (
	"net/http"
	"strconv"

	"icctv-http-service/models"
	"icctv-http-service/services"
)

// OrangePiControllerInterface 定义设备接口能力
type OrangePiControllerInterface interface {
	List(w http.ResponseWriter, r *http.Request)              //1.查询设备
	Create(w http.ResponseWriter, r *http.Request)            //2.创建设备
	Update(w http.ResponseWriter, r *http.Request)            //3.更新设备
	Delete(w http.ResponseWriter, r *http.Request)            //4.删除设备
	RemoteUpdatePorts(w http.ResponseWriter, r *http.Request) //5.远程更新端口
	RemoteGetInfo(w http.ResponseWriter, r *http.Request)     //6.远程获取设备信息
	RemoteHealthCheck(w http.ResponseWriter, r *http.Request) //7.远程健康检查
	// MediaMTX 远程管理
	RemoteListMediaMTXPaths(w http.ResponseWriter, r *http.Request)  //9.远程列出paths
	RemoteGetMediaMTXPath(w http.ResponseWriter, r *http.Request)    //10.远程查询单个path
	RemoteAddMediaMTXPath(w http.ResponseWriter, r *http.Request)    //11.远程新增path
	RemoteUpdateMediaMTXPath(w http.ResponseWriter, r *http.Request) //12.远程更新path
	RemoteDeleteMediaMTXPath(w http.ResponseWriter, r *http.Request) //13.远程删除path
}

// OrangePiController 设备接口
type OrangePiController struct {
	service *services.OrangePiService
}

// 0. NewOrangePiController 构造函数
func NewOrangePiController(service *services.OrangePiService) *OrangePiController {
	return &OrangePiController{service: service}
}

// 1. List 查询设备
func (c *OrangePiController) List(w http.ResponseWriter, r *http.Request) {
	ismartId := r.URL.Query().Get("ismartid")
	devices, err := c.service.List(r.Context(), ismartId)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, devices)
}

type orangePiPayload struct {
	ISmartID                   string `json:"ismartid"`
	Name                       string `json:"name"`
	ICCTVAuthServiceRemotePort int    `json:"icctv_auth_service_remote_port"`
	SSHRemotePort              int    `json:"ssh_remote_port"`
	IsActive                   *bool  `json:"is_active"`
	UserChannels               *[]int `json:"user_channels"` // 普通用户可访问的频道列表
	AllChannels                *[]int `json:"all_channels"`  // 所有可用频道列表
}

// 2. Create 创建设备
func (c *OrangePiController) Create(w http.ResponseWriter, r *http.Request) {
	var req orangePiPayload
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	device := models.OrangePi{
		ISmartID:                   req.ISmartID,
		Name:                       req.Name,
		ICCTVAuthServiceRemotePort: req.ICCTVAuthServiceRemotePort,
		SSHRemotePort:              req.SSHRemotePort,
	}
	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}
	if req.UserChannels != nil {
		device.UserChannels = *req.UserChannels
	}
	if req.AllChannels != nil {
		device.AllChannels = *req.AllChannels
	}

	result, err := c.service.Create(r.Context(), device)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondData(w, http.StatusCreated, result)
}

// 3. Update 更新设备
func (c *OrangePiController) Update(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req orangePiPayload
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	device := models.OrangePi{
		ISmartID:                   req.ISmartID,
		Name:                       req.Name,
		ICCTVAuthServiceRemotePort: req.ICCTVAuthServiceRemotePort,
		SSHRemotePort:              req.SSHRemotePort,
	}
	if req.IsActive != nil {
		device.IsActive = *req.IsActive
	}
	if req.UserChannels != nil {
		device.UserChannels = *req.UserChannels
	}
	if req.AllChannels != nil {
		device.AllChannels = *req.AllChannels
	}

	result, err := c.service.Update(r.Context(), id, device)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

type deleteOrangePiRequest struct {
	ID int64 `json:"id"`
}

// 4. Delete 删除设备
func (c *OrangePiController) Delete(w http.ResponseWriter, r *http.Request) {
	var req deleteOrangePiRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := c.service.Delete(r.Context(), req.ID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]bool{"deleted": true})
}

type remoteUpdatePortsRequest struct {
	ID       int64 `json:"id"`
	SSHPort  int   `json:"ssh_remote_port"`
	AuthPort int   `json:"icctv_auth_service_remote_port"`
}

// 5. RemoteUpdatePorts 远程更新设备端口
func (c *OrangePiController) RemoteUpdatePorts(w http.ResponseWriter, r *http.Request) {
	var req remoteUpdatePortsRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 || req.SSHPort == 0 || req.AuthPort == 0 {
		respondError(w, http.StatusBadRequest, "id, ssh_remote_port and icctv_auth_service_remote_port are required")
		return
	}

	result, err := c.service.RemoteUpdatePorts(r.Context(), req.ID, req.SSHPort, req.AuthPort)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

// 6. RemoteGetInfo 远程获取设备信息
func (c *OrangePiController) RemoteGetInfo(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// 获取token参数
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	deviceInfo, err := c.service.RemoteGetInfo(r.Context(), id, token)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, deviceInfo)
}

// 7. RemoteHealthCheck 远程健康检查
func (c *OrangePiController) RemoteHealthCheck(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	healthStatus, err := c.service.RemoteHealthCheck(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, healthStatus)
}

// 9. RemoteListMediaMTXPaths 远程列出 MediaMTX paths
func (c *OrangePiController) RemoteListMediaMTXPaths(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	page := 0
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	itemsPerPage := 50
	if itemsStr := r.URL.Query().Get("items_per_page"); itemsStr != "" {
		if items, err := strconv.Atoi(itemsStr); err == nil && items > 0 && items <= 500 {
			itemsPerPage = items
		}
	}

	result, err := c.service.RemoteListMediaMTXPaths(r.Context(), id, token, page, itemsPerPage)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

// 10. RemoteGetMediaMTXPath 远程查询单个 MediaMTX path
func (c *OrangePiController) RemoteGetMediaMTXPath(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	result, err := c.service.RemoteGetMediaMTXPath(r.Context(), id, token, name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

type remoteMediaMTXPathRequest struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

// 11. RemoteAddMediaMTXPath 远程新增 MediaMTX path
func (c *OrangePiController) RemoteAddMediaMTXPath(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	var req remoteMediaMTXPathRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name == "" || req.Config == nil {
		respondError(w, http.StatusBadRequest, "name and config are required")
		return
	}

	result, err := c.service.RemoteAddMediaMTXPath(r.Context(), id, token, req.Name, req.Config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

type remoteMediaMTXPathUpdateRequest struct {
	Config map[string]interface{} `json:"config"`
}

// 12. RemoteUpdateMediaMTXPath 远程更新 MediaMTX path
func (c *OrangePiController) RemoteUpdateMediaMTXPath(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	var req remoteMediaMTXPathUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if req.Config == nil {
		respondError(w, http.StatusBadRequest, "config is required")
		return
	}

	result, err := c.service.RemoteUpdateMediaMTXPath(r.Context(), id, token, name, req.Config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

// 13. RemoteDeleteMediaMTXPath 远程删除 MediaMTX path
func (c *OrangePiController) RemoteDeleteMediaMTXPath(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, http.StatusBadRequest, "token is required")
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	result, err := c.service.RemoteDeleteMediaMTXPath(r.Context(), id, token, name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

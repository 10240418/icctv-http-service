package controllers

// NVRController Methods:
//0. NewNVRController(service *services.NVRService) -> 注入 NVRService
//1. List(w http.ResponseWriter, r *http.Request) -> 列出NVR
//2. Create(w http.ResponseWriter, r *http.Request) -> 创建NVR
//3. Update(w http.ResponseWriter, r *http.Request) -> 更新NVR
//4. Delete(w http.ResponseWriter, r *http.Request) -> 删除NVR

import (
	"net/http"
	"strconv"

	"icctv-http-service/models"
	"icctv-http-service/services"
	
	"gorm.io/gorm"
)

// NVRControllerInterface 定义NVR接口能力
type NVRControllerInterface interface {
	List(w http.ResponseWriter, r *http.Request)           //1.查询NVR列表/详情
	Create(w http.ResponseWriter, r *http.Request)         //2.创建NVR
	Update(w http.ResponseWriter, r *http.Request)         //3.更新NVR
	Delete(w http.ResponseWriter, r *http.Request)         //4.删除NVR
	UpdateRTSPUrls(w http.ResponseWriter, r *http.Request) //6.专门更新RTSP URLs
	UpdateAdminUser(w http.ResponseWriter, r *http.Request) //7.更新管理员账户
	UpdateUsers(w http.ResponseWriter, r *http.Request)     //8.更新普通用户列表
	AddRTSPUrl(w http.ResponseWriter, r *http.Request)      //9.添加RTSP URL
	RemoveRTSPUrl(w http.ResponseWriter, r *http.Request)   //10.删除RTSP URL
	AddUser(w http.ResponseWriter, r *http.Request)         //11.添加用户
	RemoveUser(w http.ResponseWriter, r *http.Request)      //12.删除用户
}

// NVRController NVR接口
type NVRController struct {
	service *services.NVRService
}

// 0. NewNVRController 构造函数
func NewNVRController(service *services.NVRService) *NVRController {
	return &NVRController{service: service}
}

// 1. List 查询NVR列表或详情
func (c *NVRController) List(w http.ResponseWriter, r *http.Request) {
	// 检查是否有id参数，如果有则返回详情
	idStr := r.URL.Query().Get("id")
	if idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id parameter")
			return
		}
		item, err := c.service.GetByID(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondData(w, http.StatusOK, item)
		return
	}

	// 否则返回列表
	items, err := c.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, items)
}

// 2. Create 创建NVR
func (c *NVRController) Create(w http.ResponseWriter, r *http.Request) {
	var req models.NVR
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := c.service.Create(r.Context(), req)
	respondResult(w, err, http.StatusCreated, item)
}

// 3. Update 更新NVR
func (c *NVRController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromQuery(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req models.NVR
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := c.service.Update(r.Context(), id, req)
	respondResult(w, err, http.StatusOK, item)
}

type deleteNVRRequest struct {
	ID int64 `json:"id"`
}

// 4. Delete 删除NVR
func (c *NVRController) Delete(w http.ResponseWriter, r *http.Request) {
	var req deleteNVRRequest
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

type updateRTSPUrlsRequest struct {
	ID       int64               `json:"id"`
	RTSPUrls []models.ChannelURL `json:"rtsp_urls"`
}

// 6. UpdateRTSPUrls 专门更新RTSP URLs
func (c *NVRController) UpdateRTSPUrls(w http.ResponseWriter, r *http.Request) {
	var req updateRTSPUrlsRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	result, err := c.service.UpdateRTSPUrls(r.Context(), req.ID, req.RTSPUrls)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

type updateAdminUserRequest struct {
	ID        int64            `json:"id"`
	AdminUser models.AdminUser `json:"admin_user"`
}

// 7. UpdateAdminUser 更新管理员账户
func (c *NVRController) UpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	var req updateAdminUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	result, err := c.service.UpdateAdminUser(r.Context(), req.ID, req.AdminUser)
	respondResult(w, err, http.StatusOK, result)
}

type updateUsersRequest struct {
	ID    int64         `json:"id"`
	Users []models.User `json:"users"`
}

// 8. UpdateUsers 更新普通用户列表
func (c *NVRController) UpdateUsers(w http.ResponseWriter, r *http.Request) {
	var req updateUsersRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	result, err := c.service.UpdateUsers(r.Context(), req.ID, req.Users)
	respondResult(w, err, http.StatusOK, result)
}

type addRTSPUrlRequest struct {
	ID  int64              `json:"id"`
	URL models.ChannelURL `json:"url"`
}

// 9. AddRTSPUrl 添加RTSP URL
func (c *NVRController) AddRTSPUrl(w http.ResponseWriter, r *http.Request) {
	var req addRTSPUrlRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	result, err := c.service.AddRTSPUrl(r.Context(), req.ID, req.URL)
	if err != nil {
		if err == gorm.ErrDuplicatedKey {
			respondError(w, http.StatusConflict, "channel already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

type removeRTSPUrlRequest struct {
	ID      int64 `json:"id"`
	Channel int   `json:"channel"`
}

// 10. RemoveRTSPUrl 删除RTSP URL
func (c *NVRController) RemoveRTSPUrl(w http.ResponseWriter, r *http.Request) {
	var req removeRTSPUrlRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	result, err := c.service.RemoveRTSPUrl(r.Context(), req.ID, req.Channel)
	respondResult(w, err, http.StatusOK, result)
}

type addUserRequest struct {
	ID   int64       `json:"id"`
	User models.User `json:"user"`
}

// 11. AddUser 添加用户
func (c *NVRController) AddUser(w http.ResponseWriter, r *http.Request) {
	var req addUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	result, err := c.service.AddUser(r.Context(), req.ID, req.User)
	if err != nil {
		if err == gorm.ErrDuplicatedKey {
			respondError(w, http.StatusConflict, "user already exists")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, result)
}

type removeUserRequest struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

// 12. RemoveUser 删除用户
func (c *NVRController) RemoveUser(w http.ResponseWriter, r *http.Request) {
	var req removeUserRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}
	if req.Username == "" {
		respondError(w, http.StatusBadRequest, "username is required")
		return
	}

	result, err := c.service.RemoveUser(r.Context(), req.ID, req.Username)
	respondResult(w, err, http.StatusOK, result)
}

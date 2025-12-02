package controllers

// AdminController Methods:
//0. NewAdminController(service *services.AdminService) -> 注入 AdminService
//1. List(w http.ResponseWriter, r *http.Request) -> 列表或单条详情
//2. Create(w http.ResponseWriter, r *http.Request) -> 创建管理员
//3. Update(w http.ResponseWriter, r *http.Request) -> 更新管理员
//4. Delete(w http.ResponseWriter, r *http.Request) -> 删除管理员

import (
	"net/http"
	"strconv"

	"icctv-http-service/models"
	"icctv-http-service/services"
)

// AdminControllerInterface 定义管理员接口能力
type AdminControllerInterface interface {
	List(w http.ResponseWriter, r *http.Request)   //1.列表或单条查询
	Create(w http.ResponseWriter, r *http.Request) //2.创建管理员
	Update(w http.ResponseWriter, r *http.Request) //3.更新管理员
	Delete(w http.ResponseWriter, r *http.Request) //4.删除管理员
}

// AdminController 管理员接口
type AdminController struct {
	service *services.AdminService
}

// 0. NewAdminController 构造函数
func NewAdminController(service *services.AdminService) *AdminController {
	return &AdminController{service: service}
}

// 1. List 列表或详情
func (c *AdminController) List(w http.ResponseWriter, r *http.Request) {
	if idStr := r.URL.Query().Get("id"); idStr != "" {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid id")
			return
		}
		admin, err := c.service.GetByID(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondData(w, http.StatusOK, admin)
		return
	}

	query := models.PaginationQuery{
		PageNum:  parseInt(r.URL.Query().Get("pageNum"), 1),
		PageSize: parseInt(r.URL.Query().Get("pageSize"), 20),
		Asc:      r.URL.Query().Get("asc") == "true",
	}
	admins, pageResult, err := c.service.List(r.Context(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]interface{}{
		"items": admins,
		"page":  pageResult,
	})
}

type createAdminRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 2. Create 创建管理员
func (c *AdminController) Create(w http.ResponseWriter, r *http.Request) {
	var req createAdminRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	admin, err := c.service.Create(r.Context(), req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondData(w, http.StatusCreated, admin)
}

type updateAdminRequest struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// 3. Update 更新管理员
func (c *AdminController) Update(w http.ResponseWriter, r *http.Request) {
	var req updateAdminRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.ID == 0 {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	admin, err := c.service.Update(r.Context(), req.ID, req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondData(w, http.StatusOK, admin)
}

type deleteAdminRequest struct {
	ID int64 `json:"id"`
}

// 4. Delete 删除管理员
func (c *AdminController) Delete(w http.ResponseWriter, r *http.Request) {
	var req deleteAdminRequest
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

// parseInt 将字符串解析为 int，失败返回默认值
func parseInt(val string, defaultVal int) int {
	if val == "" {
		return defaultVal
	}
	if num, err := strconv.Atoi(val); err == nil {
		return num
	}
	return defaultVal
}

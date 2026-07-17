package controllers

// BuildingController Methods:
//0. NewBuildingController(service *services.BuildingService) -> 注入 BuildingService
//1. List(w http.ResponseWriter, r *http.Request) -> 列出建筑
//2. Create(w http.ResponseWriter, r *http.Request) -> 创建建筑
//3. Update(w http.ResponseWriter, r *http.Request) -> 更新建筑
//4. Delete(w http.ResponseWriter, r *http.Request) -> 删除建筑

import (
	"net/http"
	"strconv"

	"icctv-http-service/models"
	"icctv-http-service/services"
)

// BuildingControllerInterface 定义建筑接口能力
type BuildingControllerInterface interface {
	List(w http.ResponseWriter, r *http.Request)                   //1.查询建筑
	Create(w http.ResponseWriter, r *http.Request)                 //2.创建建筑
	Update(w http.ResponseWriter, r *http.Request)                 //3.更新建筑
	Delete(w http.ResponseWriter, r *http.Request)                 //4.删除建筑
	BindOrangePi(w http.ResponseWriter, r *http.Request)           //5.绑定OrangePi
	UnbindOrangePi(w http.ResponseWriter, r *http.Request)         //6.解绑OrangePi
	UpdateBind(w http.ResponseWriter, r *http.Request)             //7.更新绑定
	GetBuildingOrangePis(w http.ResponseWriter, r *http.Request)   //8.查询Building关联的OrangePi
	UpdateOrangePiChannels(w http.ResponseWriter, r *http.Request) //9.更新大厦可见频道
	BindNVR(w http.ResponseWriter, r *http.Request)                //10.绑定NVR
	UnbindNVR(w http.ResponseWriter, r *http.Request)              //11.解绑NVR
	GetBuildingNVRs(w http.ResponseWriter, r *http.Request)        //12.查询Building关联的NVR
}

// BuildingController 建筑接口
type BuildingController struct {
	service *services.BuildingService
}

// 0. NewBuildingController 构造函数
func NewBuildingController(service *services.BuildingService) *BuildingController {
	return &BuildingController{service: service}
}

// 1. List 查询建筑
func (c *BuildingController) List(w http.ResponseWriter, r *http.Request) {
	items, err := c.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, items)
}

// 2. Create 创建建筑
func (c *BuildingController) Create(w http.ResponseWriter, r *http.Request) {
	var req models.Building
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := c.service.Create(r.Context(), req)
	respondResult(w, err, http.StatusCreated, item)
}

// 3. Update 更新建筑
func (c *BuildingController) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromQuery(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req models.Building
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := c.service.Update(r.Context(), id, req)
	respondResult(w, err, http.StatusOK, item)
}

type deleteBuildingRequest struct {
	ID int64 `json:"id"`
}

// 4. Delete 删除建筑
func (c *BuildingController) Delete(w http.ResponseWriter, r *http.Request) {
	var req deleteBuildingRequest
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

type bindOrangePiRequest struct {
	BuildingID int64 `json:"building_id"`
	OrangePiID int64 `json:"orangepi_id"`
}

type unbindOrangePiRequest struct {
	BuildingID int64 `json:"building_id"`
	OrangePiID int64 `json:"orangepi_id"`
}

type updateBindRequest struct {
	OrangePiID    int64 `json:"orangepi_id"`
	NewBuildingID int64 `json:"new_building_id"`
}

// 5. BindOrangePi 绑定OrangePi到建筑
func (c *BuildingController) BindOrangePi(w http.ResponseWriter, r *http.Request) {
	var req bindOrangePiRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.BuildingID <= 0 || req.OrangePiID <= 0 {
		respondError(w, http.StatusBadRequest, "building_id and orangepi_id must be positive integers")
		return
	}
	if err := c.service.BindOrangePi(r.Context(), req.BuildingID, req.OrangePiID); err != nil {
		handleServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, map[string]bool{"bound": true})
}

// 6. UnbindOrangePi 解绑OrangePi
func (c *BuildingController) UnbindOrangePi(w http.ResponseWriter, r *http.Request) {
	var req unbindOrangePiRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.OrangePiID <= 0 {
		respondError(w, http.StatusBadRequest, "orangepi_id must be a positive integer")
		return
	}
	if err := c.service.UnbindOrangePi(r.Context(), req.BuildingID, req.OrangePiID); err != nil {
		handleServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, map[string]bool{"unbound": true})
}

// 7. UpdateBind 更新绑定关系
func (c *BuildingController) UpdateBind(w http.ResponseWriter, r *http.Request) {
	var req updateBindRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.OrangePiID == 0 || req.NewBuildingID == 0 {
		respondError(w, http.StatusBadRequest, "orangepi_id and new_building_id are required")
		return
	}
	if err := c.service.UpdateBind(r.Context(), req.OrangePiID, req.NewBuildingID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, map[string]bool{"updated": true})
}

// 8. GetBuildingOrangePis 查询Building关联的所有OrangePi
func (c *BuildingController) GetBuildingOrangePis(w http.ResponseWriter, r *http.Request) {
	// 从路径参数中获取building_id
	buildingIDStr := r.PathValue("building_id")
	if buildingIDStr == "" {
		respondError(w, http.StatusBadRequest, "building_id is required")
		return
	}

	buildingID, err := strconv.ParseInt(buildingIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid building_id")
		return
	}

	orangePis, err := c.service.GetOrangePisByBuildingID(r.Context(), buildingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, orangePis)
}

type updateOrangePiChannelsRequest struct {
	BuildingID int64 `json:"building_id"`
	OrangePiID int64 `json:"orangepi_id"`
	Channels   []int `json:"channels"`
}

// 9. UpdateOrangePiChannels 更新大厦可见频道。
func (c *BuildingController) UpdateOrangePiChannels(w http.ResponseWriter, r *http.Request) {
	var req updateOrangePiChannelsRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.BuildingID <= 0 || req.OrangePiID <= 0 {
		respondError(w, http.StatusBadRequest, "building_id and orangepi_id must be positive integers")
		return
	}
	if err := c.service.UpdateOrangePiChannels(r.Context(), req.BuildingID, req.OrangePiID, req.Channels); err != nil {
		handleServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, map[string]bool{"updated": true})
}

type bindNVRRequest struct {
	BuildingID int64 `json:"building_id"`
	NVRID      int64 `json:"nvr_id"`
}

type unbindNVRRequest struct {
	NVRID int64 `json:"nvr_id"`
}

// 10. BindNVR 绑定NVR到建筑
func (c *BuildingController) BindNVR(w http.ResponseWriter, r *http.Request) {
	var req bindNVRRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.BuildingID <= 0 || req.NVRID <= 0 {
		respondError(w, http.StatusBadRequest, "building_id and nvr_id must be positive integers")
		return
	}
	if err := c.service.BindNVR(r.Context(), req.BuildingID, req.NVRID); err != nil {
		handleServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, map[string]bool{"bound": true})
}

// 11. UnbindNVR 解绑NVR
func (c *BuildingController) UnbindNVR(w http.ResponseWriter, r *http.Request) {
	var req unbindNVRRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.NVRID <= 0 {
		respondError(w, http.StatusBadRequest, "nvr_id must be a positive integer")
		return
	}
	if err := c.service.UnbindNVR(r.Context(), req.NVRID); err != nil {
		handleServiceError(w, err)
		return
	}
	respondData(w, http.StatusOK, map[string]bool{"unbound": true})
}

// 12. GetBuildingNVRs 查询Building关联的所有NVR
func (c *BuildingController) GetBuildingNVRs(w http.ResponseWriter, r *http.Request) {
	// 从路径参数中获取building_id
	buildingIDStr := r.PathValue("building_id")
	if buildingIDStr == "" {
		respondError(w, http.StatusBadRequest, "building_id is required")
		return
	}

	buildingID, err := strconv.ParseInt(buildingIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid building_id")
		return
	}

	nvrs, err := c.service.GetNVRsByBuildingID(r.Context(), buildingID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondData(w, http.StatusOK, nvrs)
}

package services

// NVRService Methods:
//0. NewNVRService(db *gorm.DB) -> 初始化NVR服务
//1. List(ctx context.Context) -> 查询所有NVR
//2. GetByID(ctx context.Context, id int64) -> 根据ID查询NVR
//3. Create(ctx context.Context, payload models.NVR) -> 创建NVR
//4. Update(ctx context.Context, id int64, payload models.NVR) -> 更新NVR信息
//5. Delete(ctx context.Context, id int64) -> 删除NVR

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"icctv-http-service/models"

	"gorm.io/gorm"
)

// NVRServiceInterface 定义NVR业务能力
type NVRServiceInterface interface {
	List(ctx context.Context) ([]models.NVR, error)                                                 //1.查询NVR列表
	GetByID(ctx context.Context, id int64) (*models.NVR, error)                                     //2.根据ID查询NVR
	Create(ctx context.Context, payload models.NVR) (*models.NVR, error)                            //3.创建NVR
	Update(ctx context.Context, id int64, payload models.NVR) (*models.NVR, error)                  //4.更新NVR
	Delete(ctx context.Context, id int64) error                                                     //5.删除NVR
	UpdateRTSPUrls(ctx context.Context, id int64, urls []models.ChannelURL) (*models.NVR, error)    //6.专门更新RTSP URLs
	UpdateAdminUser(ctx context.Context, id int64, adminUser models.AdminUser) (*models.NVR, error) //7.更新管理员账户
	UpdateUsers(ctx context.Context, id int64, users []models.User) (*models.NVR, error)            //8.更新普通用户列表
	AddRTSPUrl(ctx context.Context, id int64, url models.ChannelURL) (*models.NVR, error)           //9.添加RTSP URL
	RemoveRTSPUrl(ctx context.Context, id int64, channel int) (*models.NVR, error)                  //10.删除RTSP URL
	AddUser(ctx context.Context, id int64, user models.User) (*models.NVR, error)                   //11.添加用户
	RemoveUser(ctx context.Context, id int64, username string) (*models.NVR, error)                 //12.删除用户
}

// RTSPGroupSyncFunc synchronizes a saved group to its bound OrangePi.
type RTSPGroupSyncFunc func(context.Context, *models.NVR) error

// NVRService NVR业务逻辑
type NVRService struct {
	db   *gorm.DB
	sync RTSPGroupSyncFunc
}

// 0. NewNVRService 构造函数
func NewNVRService(db *gorm.DB, syncers ...RTSPGroupSyncFunc) *NVRService {
	var syncer RTSPGroupSyncFunc
	if len(syncers) > 0 {
		syncer = syncers[0]
	}
	return &NVRService{db: db, sync: syncer}
}

// 1. List NVR列表
func (s *NVRService) List(ctx context.Context) ([]models.NVR, error) {
	var items []models.NVR
	if err := s.db.WithContext(ctx).Preload("Building").Preload("OrangePi").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		hydrateRTSPPaths(&items[i])
	}
	return items, nil
}

// 2. GetByID 根据ID查询NVR
func (s *NVRService) GetByID(ctx context.Context, id int64) (*models.NVR, error) {
	var item models.NVR
	if err := s.db.WithContext(ctx).Preload("Building").Preload("OrangePi").First(&item, id).Error; err != nil {
		return nil, err
	}
	hydrateRTSPPaths(&item)
	return &item, nil
}

// 3. Create 创建NVR
func (s *NVRService) Create(ctx context.Context, payload models.NVR) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return nil, errors.New("name is required")
	}
	if err := s.validateBinding(ctx, payload.BuildingID, payload.OrangePiID, 0); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		payload.RTSPUrls, err = normalizeLibraryURLs(tx, payload.RTSPUrls)
		if err != nil {
			return err
		}
		return tx.Create(&payload).Error
	}); err != nil {
		return nil, err
	}
	if err := s.syncGroup(ctx, &payload); err != nil {
		return &payload, err
	}
	return &payload, nil
}

// 4. Update 更新NVR
func (s *NVRService) Update(ctx context.Context, id int64, payload models.NVR) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	return s.updateLocked(ctx, id, payload)
}

func (s *NVRService) updateLocked(ctx context.Context, id int64, payload models.NVR) (*models.NVR, error) {
	var item models.NVR
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}
	buildingID := item.BuildingID
	orangePiID := item.OrangePiID
	if payload.BuildingID != 0 {
		buildingID = payload.BuildingID
	}
	if payload.OrangePiID != 0 {
		orangePiID = payload.OrangePiID
	}
	if payload.Binding != nil {
		buildingID = payload.Binding.BuildingID
		orangePiID = payload.Binding.OrangePiID
	}
	if err := s.validateBinding(ctx, buildingID, orangePiID, id); err != nil {
		return nil, err
	}

	if payload.Name != "" {
		item.Name = strings.TrimSpace(payload.Name)
	}
	if payload.URL != "" {
		item.URL = payload.URL
	}
	if payload.BuildingID != 0 {
		item.BuildingID = payload.BuildingID
	}
	if payload.OrangePiID != 0 {
		item.OrangePiID = payload.OrangePiID
	}
	item.BuildingID, item.OrangePiID = buildingID, orangePiID
	if payload.AdminUser.Name != "" || payload.AdminUser.Password != "" {
		item.AdminUser = payload.AdminUser
	}
	if len(payload.Users) > 0 {
		item.Users = payload.Users
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if payload.RTSPUrls != nil {
			payload.RTSPUrls = append([]models.ChannelURL{}, payload.RTSPUrls...)
			for i := range payload.RTSPUrls {
				camera := &payload.RTSPUrls[i]
				if camera.SourceID == 0 {
					for _, old := range item.RTSPUrls {
						if sameRTSPCamera(old, *camera) && old.URL == camera.URL {
							camera.SourceID = old.SourceID
							break
						}
					}
				}
			}
			normalized, err := normalizeLibraryURLs(tx, payload.RTSPUrls)
			if err != nil {
				return err
			}
			removed := removedRTSPUrls(item.RTSPUrls, normalized)
			mergeRTSPSyncState(item.RTSPUrls, normalized)
			item.RTSPUrls = normalized
			item.PendingRTSPDeletes = mergePendingRTSPDeletes(item.PendingRTSPDeletes, removed)
		}
		return tx.Save(&item).Error
	}); err != nil {
		return nil, err
	}
	hydrateRTSPPaths(&item)
	if err := s.syncGroup(ctx, &item); err != nil {
		return &item, err
	}
	return &item, nil
}

func mergeRTSPSyncState(existing, updated []models.ChannelURL) {
	byID := make(map[string]models.ChannelURL, len(existing))
	for _, item := range existing {
		if item.ID != "" {
			byID[item.ID] = item
		}
	}
	for i := range updated {
		old, ok := byID[updated[i].ID]
		if !ok {
			for _, candidate := range existing {
				if sameRTSPCamera(candidate, updated[i]) {
					old = candidate
					ok = true
					break
				}
			}
		}
		if !ok {
			continue
		}
		if updated[i].Path == "" || updated[i].Path == old.Path {
			updated[i].Path = old.Path
			updated[i].SyncStatus = old.SyncStatus
			updated[i].SyncError = old.SyncError
			updated[i].SyncedAt = old.SyncedAt
			updated[i].SyncedOrangePiID = old.SyncedOrangePiID
			updated[i].SyncedSource = old.SyncedSource
			if updated[i].SyncedSource == "" && old.SyncStatus == "success" {
				updated[i].SyncedSource = old.URL
			}
			if updated[i].URL != old.URL || updated[i].SourceID != old.SourceID {
				updated[i].SyncStatus = "pending"
				updated[i].SyncError = ""
			}
			continue
		}
		updated[i].SyncStatus = "pending"
		updated[i].SyncError = ""
		updated[i].SyncedAt = nil
		updated[i].SyncedOrangePiID = 0
		updated[i].SyncedSource = ""
	}
}

func removedRTSPUrls(existing, updated []models.ChannelURL) []models.ChannelURL {
	removed := make([]models.ChannelURL, 0)
	for _, old := range existing {
		if old.Path == "" || (old.SyncStatus != "success" && old.SyncedSource == "") || old.SyncedOrangePiID <= 0 {
			continue
		}
		found := false
		for _, current := range updated {
			if sameRTSPCamera(old, current) {
				// A Path remap is not a camera removal confirmation.
				found = true
				break
			}
		}
		if !found {
			if old.SyncedSource != "" {
				old.URL = old.SyncedSource
			}
			old.SyncStatus = "success"
			removed = append(removed, old)
		}
	}
	return removed
}

func sameRTSPCamera(left, right models.ChannelURL) bool {
	if left.ID != "" && right.ID != "" {
		if left.ID == right.ID {
			return true
		}
	}
	return strings.EqualFold(strings.TrimSpace(left.Path), strings.TrimSpace(right.Path))
}

func mergePendingRTSPDeletes(existing, additions []models.ChannelURL) []models.ChannelURL {
	result := append([]models.ChannelURL(nil), existing...)
	seen := make(map[string]struct{}, len(result)+len(additions))
	for _, item := range result {
		seen[pendingDeleteKey(item)] = struct{}{}
	}
	for _, item := range additions {
		key := pendingDeleteKey(item)
		if item.Path == "" || item.SyncedOrangePiID <= 0 {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		result = append(result, item)
		seen[key] = struct{}{}
	}
	return result
}

func pendingDeleteKey(item models.ChannelURL) string {
	return fmt.Sprintf("%d:%s", item.SyncedOrangePiID, strings.ToLower(strings.TrimSpace(item.Path)))
}

func (s *NVRService) syncGroup(ctx context.Context, item *models.NVR) error {
	if s.sync == nil || item.BuildingID <= 0 || item.OrangePiID <= 0 {
		return nil
	}
	if err := s.ensureOrangePiBuildingLink(ctx, item.OrangePiID, item.BuildingID); err != nil {
		return err
	}
	if err := s.sync(ctx, item); err != nil {
		for i := range item.RTSPUrls {
			if item.RTSPUrls[i].SyncStatus == "" || item.RTSPUrls[i].SyncStatus == "pending" {
				item.RTSPUrls[i].SyncStatus = "error"
				item.RTSPUrls[i].SyncError = err.Error()
			}
		}
		_ = s.db.WithContext(ctx).Save(item).Error
		return err
	}
	return s.db.WithContext(ctx).Save(item).Error
}

func (s *NVRService) ensureOrangePiBuildingLink(ctx context.Context, orangePiID, buildingID int64) error {
	link := models.OrangePiBuilding{OrangePiID: orangePiID, BuildingID: buildingID}
	result := s.db.WithContext(ctx).Where(link).FirstOrCreate(&link)
	return result.Error
}

func (s *NVRService) validateBinding(ctx context.Context, buildingID, orangePiID, nvrID int64) error {
	if buildingID == 0 && orangePiID == 0 {
		return nil
	}
	if buildingID <= 0 || orangePiID <= 0 {
		return errors.New("building_id and orangepi_id must both be set for a synchronized RTSP group")
	}
	var building models.Building
	if err := s.db.WithContext(ctx).First(&building, buildingID).Error; err != nil {
		return fmt.Errorf("building not found: %w", err)
	}
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, orangePiID).Error; err != nil {
		return fmt.Errorf("orangepi not found: %w", err)
	}
	var count int64
	query := s.db.WithContext(ctx).Model(&models.NVR{}).
		Where("building_id = ? AND deleted_at IS NULL", buildingID)
	if nvrID > 0 {
		query = query.Where("id <> ?", nvrID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("building already has an RTSP group")
	}
	return nil
}

// 5. Delete 删除NVR
func (s *NVRService) Delete(ctx context.Context, id int64) error {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	return s.db.WithContext(ctx).Delete(&models.NVR{}, id).Error
}

// 6. UpdateRTSPUrls 专门更新RTSP URLs
func (s *NVRService) UpdateRTSPUrls(ctx context.Context, id int64, urls []models.ChannelURL) (*models.NVR, error) {
	if urls == nil {
		urls = []models.ChannelURL{}
	}
	return s.Update(ctx, id, models.NVR{RTSPUrls: urls})
}

// 7. UpdateAdminUser 更新管理员账户
func (s *NVRService) UpdateAdminUser(ctx context.Context, id int64, adminUser models.AdminUser) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	nvr.AdminUser = adminUser
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
}

// 8. UpdateUsers 更新普通用户列表
func (s *NVRService) UpdateUsers(ctx context.Context, id int64, users []models.User) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	nvr.Users = users
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
}

// 9. AddRTSPUrl 添加RTSP URL
func (s *NVRService) AddRTSPUrl(ctx context.Context, id int64, url models.ChannelURL) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	hydrateRTSPPaths(&nvr)
	if url.Path == "" {
		url.Path = fmt.Sprintf("channel%d", url.Channel)
	}

	// 检查 Path 或通道是否已存在
	for _, existing := range nvr.RTSPUrls {
		if strings.EqualFold(existing.Path, url.Path) || (url.Channel > 0 && existing.Channel == url.Channel) {
			return nil, gorm.ErrDuplicatedKey
		}
	}

	nvr.RTSPUrls = append(nvr.RTSPUrls, url)
	return s.updateLocked(ctx, id, models.NVR{RTSPUrls: nvr.RTSPUrls})
}

func normalizeLibraryURLs(tx *gorm.DB, values []models.ChannelURL) ([]models.ChannelURL, error) {
	if values == nil {
		return nil, nil
	}
	values = append([]models.ChannelURL{}, values...)
	for i := range values {
		if path := strings.TrimSpace(values[i].Path); path != "" {
			if err := ValidateChannelPath(path); err != nil {
				return nil, err
			}
		}
		values[i].SyncStatus = ""
		values[i].SyncError = ""
		values[i].SyncedAt = nil
		values[i].SyncedOrangePiID = 0
		values[i].SyncedSource = ""
	}
	if err := resolveLibrarySources(tx, values); err != nil {
		return nil, err
	}
	return normalizeRTSPURLs(values)
}

func normalizeRTSPURLs(values []models.ChannelURL) ([]models.ChannelURL, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]models.ChannelURL, 0, len(values))
	seenPaths := make(map[string]struct{}, len(values))
	for index, value := range values {
		if strings.TrimSpace(value.ID) == "" {
			value.ID = fmt.Sprintf("camera-%d-%d", time.Now().UnixNano(), index)
		}
		value.Path = strings.TrimSpace(value.Path)
		if value.Path == "" {
			value.Path = fmt.Sprintf("channel%d", index+1)
		}
		value.URL = strings.TrimSpace(value.URL)
		value.Remark = strings.TrimSpace(value.Remark)
		if value.Visible == nil {
			visible := true
			value.Visible = &visible
		}
		value.SyncStatus = strings.TrimSpace(value.SyncStatus)
		value.SyncError = strings.TrimSpace(value.SyncError)
		if value.Path == "" && value.Channel > 0 {
			value.Path = fmt.Sprintf("channel%d", value.Channel)
		}
		if err := ValidateChannelPath(value.Path); err != nil {
			return nil, fmt.Errorf("rtsp_urls[%d]: %w", index, err)
		}
		if value.URL == "" {
			return nil, fmt.Errorf("rtsp_urls[%d].url is required", index)
		}
		if !strings.HasPrefix(strings.ToLower(value.URL), "rtsp://") && !strings.HasPrefix(strings.ToLower(value.URL), "rtsps://") {
			return nil, fmt.Errorf("rtsp_urls[%d].url must start with rtsp:// or rtsps://", index)
		}
		pathKey := strings.ToLower(value.Path)
		if _, exists := seenPaths[pathKey]; exists {
			return nil, fmt.Errorf("duplicate path: %s", value.Path)
		}
		seenPaths[pathKey] = struct{}{}
		value.Channel = numericChannel(value.Path)
		result = append(result, value)
	}
	return result, nil
}

func hydrateRTSPPaths(nvr *models.NVR) {
	for i := range nvr.RTSPUrls {
		nvr.RTSPUrls[i].Path = strings.TrimSpace(nvr.RTSPUrls[i].Path)
		if nvr.RTSPUrls[i].Path == "" && nvr.RTSPUrls[i].Channel > 0 {
			nvr.RTSPUrls[i].Path = fmt.Sprintf("channel%d", nvr.RTSPUrls[i].Channel)
		}
		if nvr.RTSPUrls[i].Channel <= 0 {
			nvr.RTSPUrls[i].Channel = channelFromPath(nvr.RTSPUrls[i].Path)
		}
	}
}

func channelFromPath(path string) int {
	value := strings.ToLower(strings.TrimSpace(path))
	if !strings.HasPrefix(value, "channel") {
		return 0
	}
	channel, err := strconv.Atoi(strings.TrimPrefix(value, "channel"))
	if err != nil || channel <= 0 {
		return 0
	}
	return channel
}

// 10. RemoveRTSPUrl 删除RTSP URL
func (s *NVRService) RemoveRTSPUrl(ctx context.Context, id int64, channel int) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	// 过滤掉指定通道
	newUrls := []models.ChannelURL{}
	for _, url := range nvr.RTSPUrls {
		if url.Channel != channel {
			newUrls = append(newUrls, url)
		}
	}

	return s.updateLocked(ctx, id, models.NVR{RTSPUrls: newUrls})
}

// 11. AddUser 添加用户
func (s *NVRService) AddUser(ctx context.Context, id int64, user models.User) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	// 检查用户是否已存在
	for _, existing := range nvr.Users {
		if existing.Name == user.Name {
			return nil, gorm.ErrDuplicatedKey
		}
	}

	nvr.Users = append(nvr.Users, user)
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
}

// 12. RemoveUser 删除用户
func (s *NVRService) RemoveUser(ctx context.Context, id int64, username string) (*models.NVR, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	// 过滤掉指定用户
	newUsers := []models.User{}
	for _, user := range nvr.Users {
		if user.Name != username {
			newUsers = append(newUsers, user)
		}
	}

	nvr.Users = newUsers
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
}

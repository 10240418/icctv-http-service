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

// NVRService NVR业务逻辑
type NVRService struct {
	db *gorm.DB
}

// 0. NewNVRService 构造函数
func NewNVRService(db *gorm.DB) *NVRService {
	return &NVRService{db: db}
}

// 1. List NVR列表
func (s *NVRService) List(ctx context.Context) ([]models.NVR, error) {
	var items []models.NVR
	if err := s.db.WithContext(ctx).Preload("Building").Find(&items).Error; err != nil {
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
	if err := s.db.WithContext(ctx).Preload("Building").First(&item, id).Error; err != nil {
		return nil, err
	}
	hydrateRTSPPaths(&item)
	return &item, nil
}

// 3. Create 创建NVR
func (s *NVRService) Create(ctx context.Context, payload models.NVR) (*models.NVR, error) {
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return nil, errors.New("name is required")
	}
	var err error
	payload.RTSPUrls, err = normalizeRTSPURLs(payload.RTSPUrls)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(&payload).Error; err != nil {
		return nil, err
	}
	return &payload, nil
}

// 4. Update 更新NVR
func (s *NVRService) Update(ctx context.Context, id int64, payload models.NVR) (*models.NVR, error) {
	var item models.NVR
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
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
	if payload.AdminUser.Name != "" || payload.AdminUser.Password != "" {
		item.AdminUser = payload.AdminUser
	}
	if len(payload.Users) > 0 {
		item.Users = payload.Users
	}
	if payload.RTSPUrls != nil {
		normalized, err := normalizeRTSPURLs(payload.RTSPUrls)
		if err != nil {
			return nil, err
		}
		item.RTSPUrls = normalized
	}

	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	hydrateRTSPPaths(&item)
	return &item, nil
}

// 5. Delete 删除NVR
func (s *NVRService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&models.NVR{}, id).Error
}

// 6. UpdateRTSPUrls 专门更新RTSP URLs
func (s *NVRService) UpdateRTSPUrls(ctx context.Context, id int64, urls []models.ChannelURL) (*models.NVR, error) {
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	normalized, err := normalizeRTSPURLs(urls)
	if err != nil {
		return nil, err
	}

	// 只更新RTSPUrls字段
	nvr.RTSPUrls = normalized
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
}

// 7. UpdateAdminUser 更新管理员账户
func (s *NVRService) UpdateAdminUser(ctx context.Context, id int64, adminUser models.AdminUser) (*models.NVR, error) {
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
	var nvr models.NVR
	if err := s.db.WithContext(ctx).First(&nvr, id).Error; err != nil {
		return nil, err
	}

	normalized, err := normalizeRTSPURLs([]models.ChannelURL{url})
	if err != nil {
		return nil, err
	}
	url = normalized[0]
	hydrateRTSPPaths(&nvr)

	// 检查 Path 或通道是否已存在
	for _, existing := range nvr.RTSPUrls {
		if strings.EqualFold(existing.Path, url.Path) || (url.Channel > 0 && existing.Channel == url.Channel) {
			return nil, gorm.ErrDuplicatedKey
		}
	}

	nvr.RTSPUrls = append(nvr.RTSPUrls, url)
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
}

func normalizeRTSPURLs(values []models.ChannelURL) ([]models.ChannelURL, error) {
	if values == nil {
		return nil, nil
	}
	result := make([]models.ChannelURL, 0, len(values))
	seenPaths := make(map[string]struct{}, len(values))
	for index, value := range values {
		value.Path = strings.TrimSpace(value.Path)
		value.URL = strings.TrimSpace(value.URL)
		value.Remark = strings.TrimSpace(value.Remark)
		if value.Path == "" && value.Channel > 0 {
			value.Path = fmt.Sprintf("channel%d", value.Channel)
		}
		if value.Path == "" {
			return nil, fmt.Errorf("rtsp_urls[%d].path is required", index)
		}
		if strings.ContainsAny(value.Path, " /\\?#") {
			return nil, fmt.Errorf("rtsp_urls[%d].path contains invalid characters", index)
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
		if value.Channel <= 0 {
			value.Channel = channelFromPath(value.Path)
		}
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

	nvr.RTSPUrls = newUrls
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
}

// 11. AddUser 添加用户
func (s *NVRService) AddUser(ctx context.Context, id int64, user models.User) (*models.NVR, error) {
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

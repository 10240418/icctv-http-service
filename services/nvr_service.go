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

	"icctv-http-service/models"

	"gorm.io/gorm"
)

// NVRServiceInterface 定义NVR业务能力
type NVRServiceInterface interface {
	List(ctx context.Context) ([]models.NVR, error)                                              //1.查询NVR列表
	GetByID(ctx context.Context, id int64) (*models.NVR, error)                                  //2.根据ID查询NVR
	Create(ctx context.Context, payload models.NVR) (*models.NVR, error)                         //3.创建NVR
	Update(ctx context.Context, id int64, payload models.NVR) (*models.NVR, error)               //4.更新NVR
	Delete(ctx context.Context, id int64) error                                                  //5.删除NVR
	UpdateRTSPUrls(ctx context.Context, id int64, urls []models.ChannelURL) (*models.NVR, error) //6.专门更新RTSP URLs
	UpdateAdminUser(ctx context.Context, id int64, adminUser models.AdminUser) (*models.NVR, error) //7.更新管理员账户
	UpdateUsers(ctx context.Context, id int64, users []models.User) (*models.NVR, error)         //8.更新普通用户列表
	AddRTSPUrl(ctx context.Context, id int64, url models.ChannelURL) (*models.NVR, error)        //9.添加RTSP URL
	RemoveRTSPUrl(ctx context.Context, id int64, channel int) (*models.NVR, error)               //10.删除RTSP URL
	AddUser(ctx context.Context, id int64, user models.User) (*models.NVR, error)                //11.添加用户
	RemoveUser(ctx context.Context, id int64, username string) (*models.NVR, error)              //12.删除用户
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
	return items, nil
}

// 2. GetByID 根据ID查询NVR
func (s *NVRService) GetByID(ctx context.Context, id int64) (*models.NVR, error) {
	var item models.NVR
	if err := s.db.WithContext(ctx).Preload("Building").First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// 3. Create 创建NVR
func (s *NVRService) Create(ctx context.Context, payload models.NVR) (*models.NVR, error) {
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
		item.Name = payload.Name
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
	if len(payload.RTSPUrls) > 0 {
		item.RTSPUrls = payload.RTSPUrls
	}

	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
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

	// 只更新RTSPUrls字段
	nvr.RTSPUrls = urls
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

	// 检查通道是否已存在
	for _, existing := range nvr.RTSPUrls {
		if existing.Channel == url.Channel {
			return nil, gorm.ErrDuplicatedKey
		}
	}

	nvr.RTSPUrls = append(nvr.RTSPUrls, url)
	if err := s.db.WithContext(ctx).Save(&nvr).Error; err != nil {
		return nil, err
	}

	return &nvr, nil
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

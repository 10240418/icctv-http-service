package services

// BuildingService Methods:
//0. NewBuildingService(db *gorm.DB) -> 初始化建筑服务
//1. List(ctx context.Context) -> 查询所有建筑及其设备
//2. Create(ctx context.Context, payload models.Building) -> 创建建筑
//3. Update(ctx context.Context, id int64, payload models.Building) -> 更新建筑信息
//4. Delete(ctx context.Context, id int64) -> 删除建筑

import (
	"context"
	"errors"
	"fmt"

	"icctv-http-service/models"

	"gorm.io/gorm"
)

// 错误定义
var (
	ErrBuildingNotFound = errors.New("building not found")
	ErrOrangePiNotFound = errors.New("orangepi not found")
	ErrNVRNotFound      = errors.New("nvr not found")
	ErrAlreadyBound     = errors.New("already bound to another building")
	ErrNotBound         = errors.New("not bound to any building")
)

// BuildingServiceInterface 定义建筑业务能力
type BuildingServiceInterface interface {
	List(ctx context.Context) ([]models.Building, error)                                                  //1.查询建筑列表
	Create(ctx context.Context, payload models.Building) (*models.Building, error)                        //2.创建建筑
	Update(ctx context.Context, id int64, payload models.Building) (*models.Building, error)              //3.更新建筑
	Delete(ctx context.Context, id int64) error                                                           //4.删除建筑
	BindOrangePi(ctx context.Context, buildingId int64, orangePiId int64) error                           //5.绑定OrangePi到建筑
	UnbindOrangePi(ctx context.Context, buildingId int64, orangePiId int64) error                         //6.解绑OrangePi
	UpdateBind(ctx context.Context, orangePiId int64, newBuildingId int64) error                          //7.更新绑定关系
	GetOrangePisByBuildingID(ctx context.Context, buildingId int64) ([]models.OrangePi, error)            //8.查询Building关联的OrangePi
	UpdateOrangePiChannels(ctx context.Context, buildingId int64, orangePiId int64, channels []int) error //9.更新大厦可见频道
	BindNVR(ctx context.Context, buildingId int64, nvrId int64) error                                     //10.绑定NVR到建筑
	UnbindNVR(ctx context.Context, nvrId int64) error                                                     //11.解绑NVR
	GetNVRsByBuildingID(ctx context.Context, buildingId int64) ([]models.NVR, error)                      //12.查询Building关联的NVR
}

// BuildingService 建筑业务逻辑
type BuildingService struct {
	db *gorm.DB
}

// 0. NewBuildingService 构造函数
func NewBuildingService(db *gorm.DB) *BuildingService {
	return &BuildingService{db: db}
}

// 1. List 建筑列表
func (s *BuildingService) List(ctx context.Context) ([]models.Building, error) {
	var items []models.Building
	if err := s.db.WithContext(ctx).Preload("OrangePis.Buildings").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		hydrateOrangePis(items[i].OrangePis)
	}
	return items, nil
}

// 2. Create 创建建筑
func (s *BuildingService) Create(ctx context.Context, payload models.Building) (*models.Building, error) {
	if err := s.db.WithContext(ctx).Create(&payload).Error; err != nil {
		return nil, err
	}
	return &payload, nil
}

// 3. Update 更新建筑
func (s *BuildingService) Update(ctx context.Context, id int64, payload models.Building) (*models.Building, error) {
	var item models.Building
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		oldISmartID := item.ISmartID
		if payload.ISmartID != "" {
			item.ISmartID = payload.ISmartID
		}
		if payload.Name != "" {
			item.Name = payload.Name
		}
		if payload.Remark != "" {
			item.Remark = payload.Remark
		}
		if err := tx.Save(&item).Error; err != nil {
			return err
		}
		if oldISmartID != item.ISmartID {
			return tx.Model(&models.OrangePi{}).
				Where("ismart_id = ?", oldISmartID).
				Update("ismart_id", item.ISmartID).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// 4. Delete 删除建筑
func (s *BuildingService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("building_id = ?", id).Delete(&models.OrangePiBuilding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Building{}, id).Error
	})
}

// 5. BindOrangePi 绑定OrangePi到建筑
func (s *BuildingService) BindOrangePi(ctx context.Context, buildingId int64, orangePiId int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查建筑是否存在
		var building models.Building
		if err := tx.First(&building, buildingId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBuildingNotFound
			}
			return err
		}

		// 检查建筑是否有有效的 ISmartID
		if building.ISmartID == "" {
			return errors.New("building has no ismart_id")
		}

		// 检查OrangePi是否存在
		var orangePi models.OrangePi
		if err := tx.First(&orangePi, orangePiId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrangePiNotFound
			}
			return err
		}

		var linkCount int64
		if err := tx.Model(&models.OrangePiBuilding{}).
			Where("orange_pi_id = ?", orangePiId).Count(&linkCount).Error; err != nil {
			return err
		}
		link := models.OrangePiBuilding{OrangePiID: orangePiId, BuildingID: buildingId}
		link.AllowedChannels = defaultBuildingChannels(orangePi)
		if err := tx.FirstOrCreate(&link, link).Error; err != nil {
			return err
		}
		if linkCount == 0 {
			return tx.Model(&orangePi).Update("ismart_id", building.ISmartID).Error
		}
		return nil
	})
}

// 6. UnbindOrangePi 解绑OrangePi
func (s *BuildingService) UnbindOrangePi(ctx context.Context, buildingId int64, orangePiId int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查OrangePi是否存在
		var orangePi models.OrangePi
		if err := tx.First(&orangePi, orangePiId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrangePiNotFound
			}
			return err
		}

		query := tx.Where("orange_pi_id = ?", orangePiId)
		if buildingId > 0 {
			query = query.Where("building_id = ?", buildingId)
		}
		if err := query.Delete(&models.OrangePiBuilding{}).Error; err != nil {
			return err
		}

		var remaining models.Building
		err := tx.
			Joins("JOIN orangepi_buildings ON orangepi_buildings.building_id = buildings.id").
			Where("orangepi_buildings.orange_pi_id = ?", orangePiId).
			Order("buildings.id ASC").First(&remaining).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Model(&orangePi).Update("ismart_id", "unbound_temp").Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&orangePi).Update("ismart_id", remaining.ISmartID).Error
	})
}

// 7. UpdateBind 更新绑定关系
func (s *BuildingService) UpdateBind(ctx context.Context, orangePiId int64, newBuildingId int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var newBuilding models.Building
		if err := tx.First(&newBuilding, newBuildingId).Error; err != nil {
			return err
		}
		var orangePi models.OrangePi
		if err := tx.First(&orangePi, orangePiId).Error; err != nil {
			return err
		}
		if err := tx.Where("orange_pi_id = ?", orangePiId).Delete(&models.OrangePiBuilding{}).Error; err != nil {
			return err
		}
		link := models.OrangePiBuilding{OrangePiID: orangePiId, BuildingID: newBuildingId}
		link.AllowedChannels = defaultBuildingChannels(orangePi)
		if err := tx.Create(&link).Error; err != nil {
			return err
		}
		return tx.Model(&orangePi).Update("ismart_id", newBuilding.ISmartID).Error
	})
}

// 8. GetOrangePisByBuildingID 查询Building关联的所有OrangePi
func (s *BuildingService) GetOrangePisByBuildingID(ctx context.Context, buildingId int64) ([]models.OrangePi, error) {
	// 先查询 building 是否存在并获取其 ismart_id
	var building models.Building
	if err := s.db.WithContext(ctx).First(&building, buildingId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBuildingNotFound
		}
		return nil, err
	}

	var orangePis []models.OrangePi
	if err := s.db.WithContext(ctx).
		Joins("JOIN orangepi_buildings ON orangepi_buildings.orange_pi_id = orangepis.id").
		Where("orangepi_buildings.building_id = ?", building.ID).
		Preload("Buildings").
		Find(&orangePis).Error; err != nil {
		return nil, err
	}
	hydrateOrangePis(orangePis)
	var links []models.OrangePiBuilding
	if err := s.db.WithContext(ctx).Where("building_id = ?", building.ID).Find(&links).Error; err != nil {
		return nil, err
	}
	channelsByOrangePi := make(map[int64][]int, len(links))
	for _, link := range links {
		channelsByOrangePi[link.OrangePiID] = append([]int(nil), link.AllowedChannels...)
	}
	for i := range orangePis {
		orangePis[i].BuildingChannels = channelsByOrangePi[orangePis[i].ID]
	}

	return orangePis, nil
}

// 9. UpdateOrangePiChannels 更新单栋大厦可见的频道集合。
func (s *BuildingService) UpdateOrangePiChannels(ctx context.Context, buildingId int64, orangePiId int64, channels []int) error {
	channels = normalizeChannels(channels)
	if len(channels) == 0 {
		return errors.New("at least one channel is required")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var building models.Building
		if err := tx.First(&building, buildingId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBuildingNotFound
			}
			return err
		}
		var orangePi models.OrangePi
		if err := tx.First(&orangePi, orangePiId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrangePiNotFound
			}
			return err
		}
		available := defaultBuildingChannels(orangePi)
		availableSet := make(map[int]struct{}, len(available))
		for _, channel := range available {
			availableSet[channel] = struct{}{}
		}
		for _, channel := range channels {
			if len(availableSet) > 0 {
				if _, ok := availableSet[channel]; !ok {
					return fmt.Errorf("channel%d is not available on orangepi", channel)
				}
			}
		}

		var link models.OrangePiBuilding
		if err := tx.Where("orange_pi_id = ? AND building_id = ?", orangePiId, buildingId).First(&link).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("orangepi is not bound to this building")
			}
			return err
		}
		link.AllowedChannels = channels
		return tx.Save(&link).Error
	})
}

// 10. BindNVR 绑定NVR到建筑
func (s *BuildingService) BindNVR(ctx context.Context, buildingId int64, nvrId int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查建筑是否存在
		var building models.Building
		if err := tx.First(&building, buildingId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBuildingNotFound
			}
			return err
		}

		// 检查NVR是否存在
		var nvr models.NVR
		if err := tx.First(&nvr, nvrId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNVRNotFound
			}
			return err
		}

		// 如果已经绑定到当前建筑，直接返回成功
		if nvr.BuildingID == buildingId {
			return nil
		}

		// 如果已绑定到其他建筑，直接更新（自动 unbind + bind）
		// 更新NVR的BuildingID
		nvr.BuildingID = buildingId
		return tx.Save(&nvr).Error
	})
}

// 11. UnbindNVR 解绑NVR
func (s *BuildingService) UnbindNVR(ctx context.Context, nvrId int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查NVR是否存在
		var nvr models.NVR
		if err := tx.First(&nvr, nvrId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNVRNotFound
			}
			return err
		}

		// 允许重复 unbind，如果已经是未绑定状态，直接返回成功
		if nvr.BuildingID == 0 {
			return nil
		}

		// 清空BuildingID（设为0）
		nvr.BuildingID = 0
		return tx.Save(&nvr).Error
	})
}

// 12. GetNVRsByBuildingID 查询Building关联的所有NVR
func (s *BuildingService) GetNVRsByBuildingID(ctx context.Context, buildingId int64) ([]models.NVR, error) {
	// 直接通过BuildingID查询NVR，确保包含所有字段
	var nvrs []models.NVR
	if err := s.db.WithContext(ctx).
		Where("building_id = ?", buildingId).
		Find(&nvrs).Error; err != nil {
		return nil, err
	}
	return nvrs, nil
}

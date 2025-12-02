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
	List(ctx context.Context) ([]models.Building, error)                                       //1.查询建筑列表
	Create(ctx context.Context, payload models.Building) (*models.Building, error)             //2.创建建筑
	Update(ctx context.Context, id int64, payload models.Building) (*models.Building, error)   //3.更新建筑
	Delete(ctx context.Context, id int64) error                                                //4.删除建筑
	BindOrangePi(ctx context.Context, buildingId int64, orangePiId int64) error                //5.绑定OrangePi到建筑
	UnbindOrangePi(ctx context.Context, orangePiId int64) error                                //6.解绑OrangePi
	UpdateBind(ctx context.Context, orangePiId int64, newBuildingId int64) error               //7.更新绑定关系
	GetOrangePisByBuildingID(ctx context.Context, buildingId int64) ([]models.OrangePi, error) //8.查询Building关联的OrangePi
	BindNVR(ctx context.Context, buildingId int64, nvrId int64) error                          //9.绑定NVR到建筑
	UnbindNVR(ctx context.Context, nvrId int64) error                                          //10.解绑NVR
	GetNVRsByBuildingID(ctx context.Context, buildingId int64) ([]models.NVR, error)           //11.查询Building关联的NVR
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
	if err := s.db.WithContext(ctx).Preload("OrangePis").Find(&items).Error; err != nil {
		return nil, err
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
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}

	if payload.ISmartID != "" {
		item.ISmartID = payload.ISmartID
	}
	if payload.Name != "" {
		item.Name = payload.Name
	}
	if payload.Remark != "" {
		item.Remark = payload.Remark
	}

	if err := s.db.WithContext(ctx).Save(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// 4. Delete 删除建筑
func (s *BuildingService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&models.Building{}, id).Error
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

		// 如果已经绑定到当前建筑，直接返回成功
		if orangePi.ISmartID == building.ISmartID {
			return nil
		}

		// 如果已绑定到其他建筑，直接更新（自动 unbind + bind）
		// 更新OrangePi的ISmartID
		orangePi.ISmartID = building.ISmartID
		return tx.Save(&orangePi).Error
	})
}

// 6. UnbindOrangePi 解绑OrangePi
func (s *BuildingService) UnbindOrangePi(ctx context.Context, orangePiId int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查OrangePi是否存在
		var orangePi models.OrangePi
		if err := tx.First(&orangePi, orangePiId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrOrangePiNotFound
			}
			return err
		}

		// 获取或创建 unbound_temp 占位 building
		var unboundBuilding models.Building
		err := tx.Where("ismart_id = ?", "unbound_temp").First(&unboundBuilding).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 如果不存在，创建一个
				unboundBuilding = models.Building{
					ISmartID: "unbound_temp",
					Name:     "临时未绑定占位楼栋",
					Remark:   "用于OrangePi未绑定状态的占位",
				}
				if err := tx.Create(&unboundBuilding).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		// 将 ISmartID 设置为 unbound_temp
		orangePi.ISmartID = "unbound_temp"
		return tx.Save(&orangePi).Error
	})
}

// 7. UpdateBind 更新绑定关系
func (s *BuildingService) UpdateBind(ctx context.Context, orangePiId int64, newBuildingId int64) error {
	// 检查新建筑是否存在
	var newBuilding models.Building
	if err := s.db.WithContext(ctx).First(&newBuilding, newBuildingId).Error; err != nil {
		return err
	}

	// 检查OrangePi是否存在
	var orangePi models.OrangePi
	if err := s.db.WithContext(ctx).First(&orangePi, orangePiId).Error; err != nil {
		return err
	}

	// 更新OrangePi的ISmartID
	orangePi.ISmartID = newBuilding.ISmartID
	return s.db.WithContext(ctx).Save(&orangePi).Error
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

	// 通过 ismart_id 查询关联的 OrangePi，确保包含所有字段
	var orangePis []models.OrangePi
	if err := s.db.WithContext(ctx).
		Where("ismart_id = ?", building.ISmartID).
		Find(&orangePis).Error; err != nil {
		return nil, err
	}

	return orangePis, nil
}

// 9. BindNVR 绑定NVR到建筑
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

// 10. UnbindNVR 解绑NVR
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

// 11. GetNVRsByBuildingID 查询Building关联的所有NVR
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

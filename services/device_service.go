package services

// DeviceService Methods:
//0. NewDeviceService(db *gorm.DB) -> 初始化设备仪表板服务
//1. Summary(ctx context.Context) -> 统计设备关键指标

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// DeviceServiceInterface 定义设备仪表板能力
type DeviceServiceInterface interface {
	Summary(ctx context.Context) (DeviceInfoSummary, error) //1.统计设备概览
}

// DeviceService 设备仪表板
type DeviceService struct {
	db *gorm.DB
}

// DeviceInfoSummary 设备概览
type DeviceInfoSummary struct {
	TotalDevices    int       `json:"totalDevices"`
	ActiveDevices   int       `json:"activeDevices"`
	BuildingBounded int       `json:"buildingBounded"`
	LastSync        time.Time `json:"lastSync"`
}

// 0. NewDeviceService 构造函数
func NewDeviceService(db *gorm.DB) *DeviceService {
	return &DeviceService{db: db}
}

// 1. Summary 获取设备概览数据
func (s *DeviceService) Summary(ctx context.Context) (DeviceInfoSummary, error) {
	var (
		totalDevices    int64
		activeDevices   int64
		buildingBounded int64
	)

	if err := s.db.WithContext(ctx).Table("orangepis").Count(&totalDevices).Error; err != nil {
		return DeviceInfoSummary{}, err
	}
	if err := s.db.WithContext(ctx).Table("orangepis").Where("is_active = ?", true).Count(&activeDevices).Error; err != nil {
		return DeviceInfoSummary{}, err
	}
	if err := s.db.WithContext(ctx).Table("buildings").Count(&buildingBounded).Error; err != nil {
		return DeviceInfoSummary{}, err
	}

	return DeviceInfoSummary{
		TotalDevices:    int(totalDevices),
		ActiveDevices:   int(activeDevices),
		BuildingBounded: int(buildingBounded),
		LastSync:        time.Now(),
	}, nil
}

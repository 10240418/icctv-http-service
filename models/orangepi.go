package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// mustMarshalJSON 辅助函数
func mustMarshalJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

// OrangePi OrangePi设备模型
type OrangePi struct {
	ModelFields

	// ISmartID 保留为兼容字段，值为 Buildings 中的第一个大厦。
	ISmartID                   string            `gorm:"type:varchar(100);not null;column:ismart_id;index" json:"ismartid"`
	ISmartIDs                  []string          `gorm:"-" json:"ismartids"`
	Name                       string            `gorm:"type:varchar(255);not null" json:"name"`                                  // Orangepi 名称
	ICCTVAuthServiceRemotePort int               `gorm:"not null" json:"icctv_auth_service_remote_port"`                          // 远程认证服务端口
	SSHRemotePort              int               `gorm:"not null" json:"ssh_remote_port"`                                         // SSH 远程端口
	IsActive                   bool              `gorm:"default:true" json:"is_active"`                                           // 是否在用
	UserChannels               []int             `gorm:"type:json;serializer:json" json:"user_channels"`                          // 普通用户可访问的频道列表
	AllChannels                []int             `gorm:"type:json;serializer:json" json:"all_channels"`                           // 所有可用频道列表
	ChannelRemarks             map[string]string `gorm:"type:json;serializer:json;column:channel_remarks" json:"channel_remarks"` // 频道说明
	Buildings                  []Building        `gorm:"many2many:orangepi_buildings;foreignKey:ID;joinForeignKey:OrangePiID;references:ID;joinReferences:BuildingID" json:"buildings,omitempty"`
	Building                   *Building         `gorm:"-" json:"building,omitempty"`
	BuildingChannels           []int             `gorm:"-" json:"building_channels,omitempty"`
	IsActiveSet                bool              `gorm:"-" json:"-"`
}

// OrangePiBuilding 是 OrangePi 与大厦的多对多关联。
type OrangePiBuilding struct {
	OrangePiID      int64     `gorm:"primaryKey"`
	BuildingID      int64     `gorm:"primaryKey"`
	AllowedChannels []int     `gorm:"type:json;serializer:json;column:allowed_channels" json:"allowed_channels"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (OrangePiBuilding) TableName() string {
	return "orangepi_buildings"
}

// TableName 指定表名
func (OrangePi) TableName() string {
	return "orangepis"
}

// BeforeSave GORM hook - 保存前处理
func (o *OrangePi) BeforeSave(tx *gorm.DB) error {
	// 如果 UserChannels 为 nil，初始化为空数组，避免数据库存为 null
	if o.UserChannels == nil {
		o.UserChannels = []int{}
	}
	// 如果 AllChannels 为 nil，初始化为空数组，避免数据库存为 null
	if o.AllChannels == nil {
		o.AllChannels = []int{}
	}
	if o.ChannelRemarks == nil {
		o.ChannelRemarks = map[string]string{}
	}

	return nil
}

// BeforeSave initializes association channels as an empty JSON array.
func (o *OrangePiBuilding) BeforeSave(tx *gorm.DB) error {
	if o.AllowedChannels == nil {
		o.AllowedChannels = []int{}
	}
	return nil
}

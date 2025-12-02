package models

import (
	"encoding/json"

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

	ISmartID                   string `gorm:"type:varchar(100);not null;column:ismart_id;index" json:"ismartid"` // 关联楼栋 ismartId
	Name                       string `gorm:"type:varchar(255);not null" json:"name"`                            // Orangepi 名称
	ICCTVAuthServiceRemotePort int    `gorm:"not null" json:"icctv_auth_service_remote_port"`                    // 远程认证服务端口
	SSHRemotePort              int    `gorm:"not null" json:"ssh_remote_port"`                                   // SSH 远程端口
	IsActive                   bool   `gorm:"default:true" json:"is_active"`                                     // 是否在用
	UserChannels               []int  `gorm:"type:json;serializer:json" json:"user_channels"`                    // 普通用户可访问的频道列表
	AllChannels                []int  `gorm:"type:json;serializer:json" json:"all_channels"`                     // 所有可用频道列表
	// 关联关系
	Building *Building `gorm:"foreignKey:ISmartID;references:ISmartID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"building,omitempty"` // 关联建筑
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

	return nil
}

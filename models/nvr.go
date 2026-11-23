package models

import "time"

// AdminUser 管理员账户信息
type AdminUser struct {
	Name     string `json:"name"`     // 管理员用户名
	Password string `json:"password"` // 管理员密码
}

// User 普通用户账户信息
type User struct {
	Name     string `json:"name"`     // 用户名
	Password string `json:"password"` // 密码
}

// ChannelURL RTSP频道地址
type ChannelURL struct {
	ID               string     `json:"id,omitempty"`
	SourceID         int64      `json:"source_id,omitempty"`
	SyncedSource     string     `json:"synced_source,omitempty"`
	Channel          int        `json:"channel"`          // 通道号，保留兼容旧数据
	Path             string     `json:"path,omitempty"`   // MediaMTX Path 名称
	URL              string     `json:"url"`              // RTSP 地址
	Remark           string     `json:"remark,omitempty"` // 地址说明
	Visible          *bool      `json:"visible,omitempty"`
	SyncStatus       string     `json:"sync_status,omitempty"`
	SyncError        string     `json:"sync_error,omitempty"`
	SyncedAt         *time.Time `json:"synced_at,omitempty"`
	SyncedOrangePiID int64      `json:"synced_orangepi_id,omitempty"`
}

// NVR 网络硬盘录像机模型
type NVR struct {
	ModelFields
	Binding *RTSPGroupBinding `gorm:"-" json:"binding,omitempty"`

	Name       string       `gorm:"type:varchar(255);not null" json:"name"`           // NVR 名称
	URL        string       `gorm:"type:varchar(255);not null;column:url" json:"url"` // NVR 访问地址 (IP:Port)
	BuildingID int64        `gorm:"default:0;index" json:"building_id"`               // 关联建筑ID (0表示未绑定)
	OrangePiID int64        `gorm:"default:0;index" json:"orangepi_id"`               // 新模式绑定的 OrangePi
	AdminUser  AdminUser    `gorm:"type:json;serializer:json" json:"admin_user"`      // 管理员账户(JSON存储)
	Users      []User       `gorm:"type:json;serializer:json" json:"users"`           // 普通用户列表(JSON存储)
	RTSPUrls   []ChannelURL `gorm:"type:json;serializer:json" json:"rtsp_urls"`       // RTSP 地址列表(JSON存储)
	// PendingRTSPDeletes stores paths removed from the group until the bound
	// OrangePi confirms that the corresponding MediaMTX path was deleted.
	PendingRTSPDeletes []ChannelURL `gorm:"type:json;serializer:json" json:"-"`
	// 关联关系
	Building *Building `gorm:"foreignKey:BuildingID;references:ID" json:"building,omitempty"` // 所属建筑 (使用指针以支持 NULL)
	OrangePi *OrangePi `gorm:"foreignKey:OrangePiID;references:ID" json:"orangepi,omitempty"`
}

// RTSPGroupBinding allows an explicit unbind without changing legacy partial updates.
type RTSPGroupBinding struct {
	BuildingID int64 `json:"building_id"`
	OrangePiID int64 `json:"orangepi_id"`
}

// TableName 指定表名
func (NVR) TableName() string {
	return "nvrs"
}

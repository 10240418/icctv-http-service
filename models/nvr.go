package models

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
	Channel int    `json:"channel"` // 通道号
	URL     string `json:"url"`     // RTSP 地址
}

// NVR 网络硬盘录像机模型
type NVR struct {
	ModelFields

	Name       string       `gorm:"type:varchar(255);not null" json:"name"`           // NVR 名称
	URL        string       `gorm:"type:varchar(255);not null;column:url" json:"url"` // NVR 访问地址 (IP:Port)
	BuildingID int64        `gorm:"default:0;index" json:"building_id"`               // 关联建筑ID (0表示未绑定)
	AdminUser  AdminUser    `gorm:"type:json;serializer:json" json:"admin_user"`      // 管理员账户(JSON存储)
	Users      []User       `gorm:"type:json;serializer:json" json:"users"`           // 普通用户列表(JSON存储)
	RTSPUrls   []ChannelURL `gorm:"type:json;serializer:json" json:"rtsp_urls"`       // RTSP 地址列表(JSON存储)
	// 关联关系
	Building *Building `gorm:"foreignKey:BuildingID;references:ID" json:"building,omitempty"` // 所属建筑 (使用指针以支持 NULL)
}

// TableName 指定表名
func (NVR) TableName() string {
	return "nvrs"
}

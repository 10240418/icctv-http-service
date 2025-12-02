package models

// Building 建筑信息模型
type Building struct {
	ModelFields

	ISmartID string `gorm:"type:varchar(100);not null;uniqueIndex;column:ismart_id" json:"ismartid"` // ismart 系统ID，唯一标识
	Name     string `gorm:"type:varchar(255);not null" json:"name"`                                  // 楼栋名称
	Remark   string `gorm:"type:text" json:"remark"`                                                 // 备注信息

	// 关联关系
	// OrangePi.ISmartID (外键) -> Building.ISmartID (引用)
	OrangePis []OrangePi `gorm:"foreignKey:ISmartID;references:ISmartID" json:"orangepis,omitempty"` // 关联的OrangePi设备列表
	NVRs      []NVR      `gorm:"foreignKey:BuildingID;references:ID" json:"nvrs,omitempty"`          // 关联的NVR设备列表
}

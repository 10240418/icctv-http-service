package models

// Building 建筑信息模型
type Building struct {
	ModelFields

	ISmartID string `gorm:"type:varchar(100);not null;uniqueIndex;column:ismart_id" json:"ismartid"` // ismart 系统ID，唯一标识
	Name     string `gorm:"type:varchar(255);not null" json:"name"`                                  // 楼栋名称
	Remark   string `gorm:"type:text" json:"remark"`                                                 // 备注信息

	// 关联关系
	OrangePis []OrangePi `gorm:"many2many:orangepi_buildings;foreignKey:ID;joinForeignKey:BuildingID;references:ID;joinReferences:OrangePiID" json:"orangepis,omitempty"`
	NVRs      []NVR      `gorm:"foreignKey:BuildingID;references:ID" json:"nvrs,omitempty"`
}

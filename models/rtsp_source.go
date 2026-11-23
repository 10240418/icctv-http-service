package models

// RTSPSource is the address library; paths and visibility belong to each group.
type RTSPSource struct {
	ModelFields
	URL        string    `gorm:"type:text;not null" json:"url"`
	Remark     string    `gorm:"type:text" json:"remark"`
	BuildingID int64     `gorm:"default:0;index" json:"building_id"`
	Building   *Building `gorm:"foreignKey:BuildingID;references:ID" json:"building,omitempty"`
}

type RTSPSourceUsage struct {
	GroupID    int64  `json:"group_id"`
	GroupName  string `json:"group_name"`
	Path       string `json:"path"`
	BuildingID int64  `json:"building_id"`
	OrangePiID int64  `json:"orangepi_id"`
}

type RTSPSourceView struct {
	RTSPSource
	Usages []RTSPSourceUsage `json:"usages"`
}

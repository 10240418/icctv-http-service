package databases

import (
	"icctv-http-service/models"

	"gorm.io/gorm"
)

// MigrateRTSPLibrary preserves every legacy camera as its own library entry.
// It updates JSON references without touching group timestamps or device state.
func MigrateRTSPLibrary(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var groups []models.NVR
		if err := tx.Order("id").Find(&groups).Error; err != nil {
			return err
		}
		for _, group := range groups {
			changed := false
			for i := range group.RTSPUrls {
				camera := &group.RTSPUrls[i]
				if camera.SourceID > 0 {
					continue
				}
				entry := models.RTSPSource{URL: camera.URL, Remark: camera.Remark, BuildingID: group.BuildingID}
				entry.CreatedAt = group.CreatedAt
				entry.UpdatedAt = group.UpdatedAt
				if err := tx.Create(&entry).Error; err != nil {
					return err
				}
				camera.SourceID = entry.ID
				if camera.SyncStatus == "success" && camera.SyncedOrangePiID > 0 {
					camera.SyncedSource = camera.URL
				}
				changed = true
			}
			if changed {
				if err := tx.Model(&group).Select("RTSPUrls").UpdateColumns(&group).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

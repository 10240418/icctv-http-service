package databases

import (
	"reflect"
	"testing"
	"time"

	"icctv-http-service/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRTSPLibraryMigrationPreservesCamerasAndIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NVR{}, &models.RTSPSource{}); err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 11, 10, 12, 13, 14, 0, time.UTC)
	group := models.NVR{
		ModelFields: models.ModelFields{CreatedAt: stamp, UpdatedAt: stamp},
		Name:        "Legacy", BuildingID: 3, OrangePiID: 7,
		RTSPUrls: []models.ChannelURL{
			{ID: "old-a", Path: "LegacyGate", URL: "rtsp://camera/1", Remark: "A", SyncStatus: "success", SyncedOrangePiID: 7},
			{ID: "old-b", Path: "channel12", URL: "rtsp://camera/1", Remark: "B"},
		},
		PendingRTSPDeletes: []models.ChannelURL{{Path: "channel9", URL: "rtsp://camera/9"}},
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	original := group
	if err := MigrateRTSPLibrary(db); err != nil {
		t.Fatal(err)
	}
	if err := MigrateRTSPLibrary(db); err != nil {
		t.Fatal(err)
	}
	var loaded models.NVR
	if err := db.First(&loaded, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	var entries []models.RTSPSource
	if err := db.Order("id").Find(&entries).Error; err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Remark != "A" || entries[1].Remark != "B" {
		t.Fatalf("duplicates must remain independent entries: %+v", entries)
	}
	for i := range loaded.RTSPUrls {
		if loaded.RTSPUrls[i].SourceID != entries[i].ID {
			t.Fatalf("missing source relation: %+v", loaded.RTSPUrls)
		}
		loaded.RTSPUrls[i].SourceID = 0
		loaded.RTSPUrls[i].SyncedSource = ""
	}
	if !loaded.UpdatedAt.Equal(stamp) || !entries[0].UpdatedAt.Equal(stamp) ||
		!reflect.DeepEqual(loaded.RTSPUrls, original.RTSPUrls) ||
		!reflect.DeepEqual(loaded.PendingRTSPDeletes, original.PendingRTSPDeletes) {
		t.Fatalf("migration changed legacy state: %+v", loaded)
	}
}

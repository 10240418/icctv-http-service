package services

import (
	"context"
	"strings"
	"testing"

	"icctv-http-service/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRTSPGroupTestService(t *testing.T) *NVRService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.NVR{}, &models.RTSPSource{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return NewNVRService(db)
}

func TestNVRServiceStoresRTSPPathsAsAGroup(t *testing.T) {
	service := newRTSPGroupTestService(t)
	ctx := context.Background()

	group, err := service.Create(ctx, models.NVR{
		Name: "Main entrance cameras",
		RTSPUrls: []models.ChannelURL{
			{Channel: 1, URL: "rtsp://camera.example/channel1", Remark: "Lobby"},
			{Path: "channel100", URL: "rtsps://camera.example/carpark", Remark: "Car park"},
		},
	})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if group.RTSPUrls[0].Path != "channel1" {
		t.Fatalf("legacy channel was not converted to a path: %+v", group.RTSPUrls[0])
	}
	if group.RTSPUrls[1].Path != "channel100" || group.RTSPUrls[1].Channel != 100 {
		t.Fatalf("high channel path was not preserved: %+v", group.RTSPUrls[1])
	}
	if group.RTSPUrls[0].ID == "" || group.RTSPUrls[0].Visible == nil || !*group.RTSPUrls[0].Visible {
		t.Fatalf("stable ID or default visibility missing: %+v", group.RTSPUrls[0])
	}

	loaded, err := service.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("load group: %v", err)
	}
	if loaded.RTSPUrls[0].Path != "channel1" || loaded.RTSPUrls[0].Remark != "Lobby" {
		t.Fatalf("unexpected stored group: %+v", loaded.RTSPUrls)
	}
}

func TestNVRServiceRequiresCompleteUniqueManagedBinding(t *testing.T) {
	service := newRTSPGroupTestService(t)
	ctx := context.Background()
	if err := service.db.AutoMigrate(&models.Building{}, &models.OrangePi{}); err != nil {
		t.Fatal(err)
	}
	building := models.Building{ISmartID: "building-1", Name: "Building 1"}
	if err := service.db.Create(&building).Error; err != nil {
		t.Fatal(err)
	}
	device := models.OrangePi{ISmartID: "building-1", Name: "OrangePi 1"}
	if err := service.db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	source := models.ChannelURL{Path: "channel1", URL: "rtsp://camera.example/1"}
	if _, err := service.Create(ctx, models.NVR{Name: "Incomplete", BuildingID: building.ID, RTSPUrls: []models.ChannelURL{source}}); err == nil {
		t.Fatal("expected incomplete building/device binding to fail")
	}
	if _, err := service.Create(ctx, models.NVR{Name: "Managed", BuildingID: building.ID, OrangePiID: device.ID, RTSPUrls: []models.ChannelURL{source}}); err != nil {
		t.Fatalf("create bound group: %v", err)
	}
	if _, err := service.Create(ctx, models.NVR{Name: "Duplicate", BuildingID: building.ID, OrangePiID: device.ID, RTSPUrls: []models.ChannelURL{source}}); err == nil {
		t.Fatal("expected one RTSP group per building constraint")
	}
}

func TestNVRServiceSynchronizesBoundGroupAfterSaving(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.NVR{}, &models.RTSPSource{}, &models.Building{}, &models.OrangePi{}, &models.OrangePiBuilding{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Building{ModelFields: models.ModelFields{ID: 1}, ISmartID: "building-1", Name: "Building 1"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.OrangePi{ModelFields: models.ModelFields{ID: 2}, ISmartID: "building-1", Name: "OrangePi 1"}).Error; err != nil {
		t.Fatal(err)
	}
	called := false
	service := NewNVRService(db, func(_ context.Context, group *models.NVR) error {
		called = true
		if group.ID == 0 || group.BuildingID != 1 || group.OrangePiID != 2 {
			t.Fatalf("sync received unexpected group: %+v", group)
		}
		return nil
	})
	group, err := service.Create(context.Background(), models.NVR{
		Name:       "Managed",
		BuildingID: 1,
		OrangePiID: 2,
		RTSPUrls:   []models.ChannelURL{{URL: "rtsp://camera.example/1"}},
	})
	if err != nil {
		t.Fatalf("create and sync group: %v", err)
	}
	if !called || group.RTSPUrls[0].Path != "channel1" {
		t.Fatalf("bound group was not synchronized: called=%v group=%+v", called, group)
	}
	var link models.OrangePiBuilding
	if err := db.Where("orange_pi_id = ? AND building_id = ?", 2, 1).First(&link).Error; err != nil {
		t.Fatalf("group binding did not create the building-device association: %v", err)
	}
}

func TestNextAvailableChannelPathSkipsExistingAndReservedIDs(t *testing.T) {
	path, channel := nextAvailableChannelPath(map[string]bool{
		"channel1": true,
		"channel2": true,
		"channel5": true,
	})
	if path != "channel3" || channel != 3 {
		t.Fatalf("unexpected allocation: %s (%d)", path, channel)
	}
}

func TestReservedRTSPGroupPathsIgnoresStalePermissionIDs(t *testing.T) {
	device := models.OrangePi{
		AllChannels:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22},
		UserChannels:     []int{1, 2},
		BuildingChannels: []int{1, 2},
		ChannelPaths: map[string]string{
			"channel1":  "channel1",
			"channel2":  "channel2",
			"channel22": "OldGate",
			"channel23": "channel200",
		},
	}
	reserved := reservedRTSPGroupPaths(map[string]MediaMTXPathItem{
		"channel1": {Name: "channel1"},
		"channel2": {Name: "channel2"},
	}, device)

	if reserved["channel22"] || reserved["channel23"] || reserved["channel200"] {
		t.Fatalf("stale catalogue IDs must not be reserved: %v", reserved)
	}
	if !reserved["channel1"] || !reserved["channel2"] {
		t.Fatalf("current remote paths must remain reserved: %v", reserved)
	}
	path, channel := nextAvailableChannelPath(reserved)
	if path != "channel3" || channel != 3 {
		t.Fatalf("expected channel3 after current paths, got %s (%d)", path, channel)
	}
}

func TestNVRServiceRejectsDuplicateOrInvalidRTSPPaths(t *testing.T) {
	service := newRTSPGroupTestService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, models.NVR{
		Name: "Duplicate paths",
		RTSPUrls: []models.ChannelURL{
			{Path: "channel1", URL: "rtsp://camera.example/one"},
			{Path: "channel1", URL: "rtsp://camera.example/two"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate path") {
		t.Fatalf("expected duplicate path error, got %v", err)
	}

	_, err = service.Create(ctx, models.NVR{
		Name:     "Invalid URL",
		RTSPUrls: []models.ChannelURL{{Path: "channel1", URL: "http://camera.example/one"}},
	})
	if err == nil || !strings.Contains(err.Error(), "must start with rtsp") {
		t.Fatalf("expected invalid RTSP URL error, got %v", err)
	}
}

func TestNVRServiceQueuesOnlySuccessfullyManagedRemovedPaths(t *testing.T) {
	service := newRTSPGroupTestService(t)
	ctx := context.Background()
	synced := true
	group := models.NVR{
		Name: "Managed group",
		RTSPUrls: []models.ChannelURL{
			{ID: "camera-1", Path: "channel1", URL: "rtsp://camera.example/1", SyncStatus: "success", SyncedOrangePiID: 7, Visible: &synced},
			{ID: "camera-2", Path: "channel2", URL: "rtsp://camera.example/2", SyncStatus: "error", SyncedOrangePiID: 7, Visible: &synced},
		},
	}
	if err := service.db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}

	updated, err := service.Update(ctx, group.ID, models.NVR{
		RTSPUrls: []models.ChannelURL{
			{ID: "camera-2", Path: "channel2", URL: "rtsp://camera.example/2"},
		},
	})
	if err != nil {
		t.Fatalf("update group: %v", err)
	}
	if len(updated.PendingRTSPDeletes) != 1 ||
		updated.PendingRTSPDeletes[0].Path != "channel1" ||
		updated.PendingRTSPDeletes[0].SyncedOrangePiID != 7 {
		t.Fatalf("unexpected pending deletes: %+v", updated.PendingRTSPDeletes)
	}
}

func TestPathRemapDoesNotConfirmDeletion(t *testing.T) {
	old := models.ChannelURL{ID: "camera-1", Path: "channel1", URL: "rtsp://camera/1", SyncStatus: "success", SyncedOrangePiID: 7}
	updated := models.ChannelURL{ID: "camera-1", Path: "channel3", URL: old.URL}
	if removed := removedRTSPUrls([]models.ChannelURL{old}, []models.ChannelURL{updated}); len(removed) != 0 {
		t.Fatal("remapping a retained camera must not delete its previous path")
	}
}

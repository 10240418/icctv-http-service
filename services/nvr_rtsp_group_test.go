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
	if err := db.AutoMigrate(&models.NVR{}); err != nil {
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
			{Path: "carpark", URL: "rtsps://camera.example/carpark", Remark: "Car park"},
		},
	})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if group.RTSPUrls[0].Path != "channel1" {
		t.Fatalf("legacy channel was not converted to a path: %+v", group.RTSPUrls[0])
	}
	if group.RTSPUrls[1].Path != "carpark" || group.RTSPUrls[1].Channel != 0 {
		t.Fatalf("custom path was not preserved: %+v", group.RTSPUrls[1])
	}

	loaded, err := service.GetByID(ctx, group.ID)
	if err != nil {
		t.Fatalf("load group: %v", err)
	}
	if loaded.RTSPUrls[0].Path != "channel1" || loaded.RTSPUrls[0].Remark != "Lobby" {
		t.Fatalf("unexpected stored group: %+v", loaded.RTSPUrls)
	}
}

func TestNVRServiceRejectsDuplicateOrInvalidRTSPPaths(t *testing.T) {
	service := newRTSPGroupTestService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, models.NVR{
		Name: "Duplicate paths",
		RTSPUrls: []models.ChannelURL{
			{Path: "channel1", URL: "rtsp://camera.example/one"},
			{Path: "CHANNEL1", URL: "rtsp://camera.example/two"},
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

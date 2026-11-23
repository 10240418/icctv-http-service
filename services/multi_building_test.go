package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"icctv-http-service/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMultiBuildingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.SetupJoinTable(&models.OrangePi{}, "Buildings", &models.OrangePiBuilding{}); err != nil {
		t.Fatalf("set up join table: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Building{},
		&models.OrangePi{},
		&models.OrangePiBuilding{},
		&models.PublicNetConfig{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db
}

func TestPublicTokenSupportsOneOrangePiBoundToMultipleBuildings(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	ctx := context.Background()

	buildings := []models.Building{
		{ISmartID: "building-a", Name: "Building A"},
		{ISmartID: "building-b", Name: "Building B"},
	}
	if err := db.Create(&buildings).Error; err != nil {
		t.Fatalf("create buildings: %v", err)
	}
	if err := db.Create(&models.PublicNetConfig{ExternalIP: "203.0.113.10"}).Error; err != nil {
		t.Fatalf("create public network config: %v", err)
	}

	publicNetService := NewPublicNetService(db)
	orangePiService := NewOrangePiService(db, publicNetService)
	device, err := orangePiService.Create(ctx, models.OrangePi{
		ISmartIDs:                  []string{"building-a", "building-b"},
		Name:                       "Orange Pi 1",
		ICCTVAuthServiceRemotePort: 29005,
		SSHRemotePort:              30005,
		IsActive:                   true,
		UserChannels:               []int{1, 2},
		AllChannels:                []int{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("create orangepi: %v", err)
	}

	authService := NewAuthService(db, nil, orangePiService, NewBuildingService(db))
	for _, ismartID := range []string{"building-a", "building-b"} {
		response, err := authService.GeneratePublicToken(ctx, ismartID, false)
		if err != nil {
			t.Fatalf("generate token for %s: %v", ismartID, err)
		}
		if len(response.OrangePis) != 1 {
			t.Fatalf("generate token for %s returned %d devices", ismartID, len(response.OrangePis))
		}
		if response.OrangePis[0].OrangePiID != device.ID {
			t.Fatalf("generate token for %s returned device %d", ismartID, response.OrangePis[0].OrangePiID)
		}
		if len(response.OrangePis[0].URLs) != 2 {
			t.Fatalf("generate token for %s returned %d urls", ismartID, len(response.OrangePis[0].URLs))
		}
	}
}

func TestRegisterMediaMTXChannelAddsAllPathsWithoutChangingPermissions(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	ctx := context.Background()
	device := models.OrangePi{
		ISmartID:                   "building-a",
		Name:                       "Orange Pi channel sync",
		ICCTVAuthServiceRemotePort: 29005,
		SSHRemotePort:              30005,
		AllChannels:                []int{1, 2, 8},
		UserChannels:               []int{1, 2},
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatalf("create orangepi: %v", err)
	}

	service := NewOrangePiService(db, NewPublicNetService(db))
	if err := service.registerMediaMTXChannel(ctx, device.ID, "channel9"); err != nil {
		t.Fatalf("register channel9: %v", err)
	}
	if err := service.registerMediaMTXChannel(ctx, device.ID, "channel9"); err != nil {
		t.Fatalf("register channel9 again: %v", err)
	}
	if err := service.registerMediaMTXChannel(ctx, device.ID, "lobby-camera"); err != nil {
		t.Fatalf("register non-channel path: %v", err)
	}

	var stored models.OrangePi
	if err := db.First(&stored, device.ID).Error; err != nil {
		t.Fatalf("load orangepi: %v", err)
	}
	if got, want := fmt.Sprint(stored.AllChannels), "[1 2 8]"; got != want {
		t.Fatalf("selected channels = %s, want %s", got, want)
	}
	if stored.ChannelPaths["channel9"] != "channel9" || stored.ChannelPaths["channel10"] != "lobby-camera" {
		t.Fatalf("numeric and custom paths must both be registered: %v", stored.ChannelPaths)
	}
	if got, want := fmt.Sprint(stored.UserChannels), "[1 2]"; got != want {
		t.Fatalf("user channels changed to %s, want %s", got, want)
	}
}

func TestUnbindOrangePiOnlyRemovesSelectedBuilding(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	ctx := context.Background()

	buildings := []models.Building{
		{ISmartID: "building-a", Name: "Building A"},
		{ISmartID: "building-b", Name: "Building B"},
	}
	if err := db.Create(&buildings).Error; err != nil {
		t.Fatalf("create buildings: %v", err)
	}

	orangePiService := NewOrangePiService(db, NewPublicNetService(db))
	device, err := orangePiService.Create(ctx, models.OrangePi{
		ISmartIDs:                  []string{"building-a", "building-b"},
		Name:                       "Orange Pi 1",
		ICCTVAuthServiceRemotePort: 29005,
		SSHRemotePort:              30005,
		IsActive:                   true,
	})
	if err != nil {
		t.Fatalf("create orangepi: %v", err)
	}

	buildingService := NewBuildingService(db)
	if err := buildingService.UnbindOrangePi(ctx, buildings[0].ID, device.ID); err != nil {
		t.Fatalf("unbind building-a: %v", err)
	}

	fromA, err := buildingService.GetOrangePisByBuildingID(ctx, buildings[0].ID)
	if err != nil {
		t.Fatalf("list building-a devices: %v", err)
	}
	if len(fromA) != 0 {
		t.Fatalf("building-a still has %d devices", len(fromA))
	}

	fromB, err := buildingService.GetOrangePisByBuildingID(ctx, buildings[1].ID)
	if err != nil {
		t.Fatalf("list building-b devices: %v", err)
	}
	if len(fromB) != 1 || fromB[0].ID != device.ID {
		t.Fatalf("building-b association was not preserved")
	}
}

func TestPublicTokensApplyBuildingChannels(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	ctx := context.Background()

	building := models.Building{ISmartID: "building-v2", Name: "Building V2"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatalf("create building: %v", err)
	}
	if err := db.Create(&models.PublicNetConfig{ExternalIP: "203.0.113.20"}).Error; err != nil {
		t.Fatalf("create public network config: %v", err)
	}

	orangePiService := NewOrangePiService(db, NewPublicNetService(db))
	device, err := orangePiService.Create(ctx, models.OrangePi{
		ISmartIDs:                  []string{building.ISmartID},
		Name:                       "Orange Pi V2",
		ICCTVAuthServiceRemotePort: 29003,
		SSHRemotePort:              30003,
		IsActive:                   true,
		UserChannels:               []int{1, 2},
		AllChannels:                []int{1, 2, 3},
		ChannelRemarks: map[string]string{
			"channel1": "Lobby",
			"channel2": "Lift",
		},
	})
	if err != nil {
		t.Fatalf("create orangepi: %v", err)
	}
	buildingService := NewBuildingService(db)
	if err := buildingService.UpdateOrangePiChannels(ctx, building.ID, device.ID, []int{2}); err != nil {
		t.Fatalf("set building channels: %v", err)
	}

	authService := NewAuthService(db, nil, orangePiService, buildingService)
	legacy, err := authService.GeneratePublicToken(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatalf("generate legacy token: %v", err)
	}
	if got := len(legacy.OrangePis[0].URLs); got != 1 {
		t.Fatalf("legacy endpoint should apply the building channel rule, got %d urls", got)
	}
	if !strings.Contains(legacy.OrangePis[0].URLs[0], "/channel2?") {
		t.Fatalf("legacy endpoint returned an unexpected url: %s", legacy.OrangePis[0].URLs[0])
	}
	legacyJSON, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy response: %v", err)
	}
	if strings.Contains(string(legacyJSON), "channel_remarks") || strings.Contains(string(legacyJSON), "remarks") {
		t.Fatalf("legacy response unexpectedly contains remarks: %s", legacyJSON)
	}
	permanent, err := authService.GeneratePermanentToken(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatalf("generate permanent token: %v", err)
	}
	if got := len(permanent.OrangePis[0].URLs); got != 1 {
		t.Fatalf("permanent token should apply the building channel rule, got %d urls", got)
	}
	if !strings.Contains(permanent.OrangePis[0].URLs[0], "/channel2?") {
		t.Fatalf("permanent token returned an unexpected url: %s", permanent.OrangePis[0].URLs[0])
	}

	v2, err := authService.GeneratePublicTokenV2(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatalf("generate v2 token: %v", err)
	}
	if got := len(v2.OrangePis[0].URLs); got != 1 {
		t.Fatalf("v2 endpoint should apply the building channel rule, got %d urls", got)
	}
	if !strings.Contains(v2.OrangePis[0].URLs[0], "/channel2?") {
		t.Fatalf("v2 endpoint returned an unexpected url: %s", v2.OrangePis[0].URLs[0])
	}
	if got := v2.OrangePis[0].ChannelRemarks["channel2"]; got != "Lift" {
		t.Fatalf("unexpected channel2 remark: %q", got)
	}
	if _, exists := v2.OrangePis[0].ChannelRemarks["channel1"]; exists {
		t.Fatalf("v2 response should only contain visible channel remarks")
	}
}

func TestManagedV2UsesVisibleSyncedChannelsForStaffWithoutAdminToken(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	if err := db.AutoMigrate(&models.NVR{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	building := models.Building{ISmartID: "managed-building", Name: "Managed Building"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatal(err)
	}
	device := models.OrangePi{
		ISmartID: "managed-building", Name: "Managed OrangePi", ICCTVAuthServiceRemotePort: 29005,
		UserChannels: []int{1}, AllChannels: []int{1, 2},
		ChannelRemarks: map[string]string{
			"channel3":  "历史备注",
			"channel12": "旧备注",
		},
		ChannelPaths: map[string]string{
			"channel3":  "OldGate",
			"channel12": "OldBuildingGate",
		},
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.OrangePiBuilding{OrangePiID: device.ID, BuildingID: building.ID, AllowedChannels: []int{1}}).Error; err != nil {
		t.Fatal(err)
	}
	visible := true
	hidden := false
	group := models.NVR{
		Name: "Managed group", BuildingID: building.ID, OrangePiID: device.ID,
		RTSPUrls: []models.ChannelURL{
			{ID: "camera-visible-12", Path: "channel12", URL: "rtsp://camera.example/12", Remark: "大门", Visible: &visible, SyncStatus: "success", SyncedOrangePiID: device.ID},
			{ID: "camera-visible-3", Path: "channel3", URL: "rtsp://camera.example/3", Remark: "管理处", Visible: &visible, SyncStatus: "success", SyncedOrangePiID: device.ID},
			{ID: "camera-hidden", Path: "channel4", URL: "rtsp://camera.example/4", Visible: &hidden, SyncStatus: "success", SyncedOrangePiID: device.ID},
			{ID: "camera-pending", Path: "channel5", URL: "rtsp://camera.example/5", Visible: &visible, SyncStatus: "error", SyncedOrangePiID: device.ID},
		},
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PublicNetConfig{ExternalIP: "203.0.113.10"}).Error; err != nil {
		t.Fatal(err)
	}

	authService := NewAuthService(db, nil, NewOrangePiService(db, NewPublicNetService(db)), NewBuildingService(db))
	response, err := authService.GeneratePublicTokenV2(ctx, building.ISmartID, true)
	if err != nil {
		t.Fatalf("generate managed staff token: %v", err)
	}
	if len(response.OrangePis) != 1 || len(response.OrangePis[0].URLs) != 2 {
		t.Fatalf("expected only visible synchronized channels, got %+v", response.OrangePis)
	}
	if !strings.Contains(response.OrangePis[0].URLs[0], "/channel3?") ||
		!strings.Contains(response.OrangePis[0].URLs[1], "/channel12?") {
		t.Fatalf("managed playback URLs are not numerically sorted: %v", response.OrangePis[0].URLs)
	}
	if !reflect.DeepEqual(response.OrangePis[0].ChannelRemarks, map[string]string{
		"channel3":  "管理处",
		"channel12": "大门",
	}) {
		t.Fatalf("managed remarks must match only returned RTSP paths: %v", response.OrangePis[0].ChannelRemarks)
	}
	encodedPayload := strings.Split(response.OrangePis[0].Token, ".")[0]
	payloadBytes, err := base64.URLEncoding.DecodeString(encodedPayload)
	if err != nil {
		t.Fatalf("decode token payload: %v", err)
	}
	var payload VideoTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("parse token payload: %v", err)
	}
	if payload.IsStaff || len(payload.Channels) != 2 || payload.Channels[0] != "channel3" || payload.Channels[1] != "channel12" {
		t.Fatalf("managed staff token is not channel-scoped: %+v", payload)
	}
}

func TestOrangePiManagementTokenDoesNotRequireBuildingAssociation(t *testing.T) {
	ctx := context.Background()
	db := newMultiBuildingTestDB(t)
	building := models.Building{ISmartID: "building-preview", Name: "Preview Building"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatal(err)
	}
	device := models.OrangePi{
		ISmartID:     building.ISmartID,
		Name:         "Preview OrangePi",
		AllChannels:  []int{1, 2},
		UserChannels: []int{1},
		ChannelPaths: map[string]string{"channel1": "channel1", "channel2": "Lobby"},
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	auth := NewAuthService(db, nil, NewOrangePiService(db, NewPublicNetService(db)), NewBuildingService(db))
	token, err := auth.GenerateOrangePiManagementToken(ctx, building.ISmartID, device.ID)
	if err != nil {
		t.Fatalf("generate management token without association: %v", err)
	}
	payloadBytes, err := base64.URLEncoding.DecodeString(strings.Split(token, ".")[0])
	if err != nil {
		t.Fatal(err)
	}
	var payload VideoTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.IsStaff || payload.BuildingID != building.ISmartID ||
		len(payload.Channels) != 2 || payload.Channels[0] != "channel1" || payload.Channels[1] != "Lobby" {
		t.Fatalf("unexpected management token claims: %+v", payload)
	}
	var links int64
	if err := db.Model(&models.OrangePiBuilding{}).Count(&links).Error; err != nil || links != 0 {
		t.Fatalf("token generation unexpectedly changed bindings: links=%d err=%v", links, err)
	}
}

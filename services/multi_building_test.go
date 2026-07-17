package services

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestPublicTokenV2AddsRemarksWithoutChangingLegacyResponse(t *testing.T) {
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
	if got := len(legacy.OrangePis[0].URLs); got != 2 {
		t.Fatalf("legacy endpoint should keep two user channels, got %d", got)
	}
	legacyJSON, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy response: %v", err)
	}
	if strings.Contains(string(legacyJSON), "channel_remarks") || strings.Contains(string(legacyJSON), "remarks") {
		t.Fatalf("legacy response unexpectedly contains remarks: %s", legacyJSON)
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

package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"icctv-http-service/models"
)

func TestRTSPChannelsBeyondNinePreserveExplicitVisibility(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	ctx := context.Background()
	building := models.Building{ISmartID: "rtsp-capacity-building", Name: "Capacity building"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PublicNetConfig{ExternalIP: "203.0.113.10"}).Error; err != nil {
		t.Fatal(err)
	}
	service := NewOrangePiService(db, NewPublicNetService(db))
	device, err := service.Create(ctx, models.OrangePi{
		ISmartIDs:                  []string{building.ISmartID},
		Name:                       "Capacity device",
		ICCTVAuthServiceRemotePort: 29005,
		SSHRemotePort:              30005,
		IsActive:                   true,
		AllChannels:                []int{1},
		UserChannels:               []int{1},
	})
	if err != nil {
		t.Fatal(err)
	}

	for channel := 2; channel <= 120; channel++ {
		if err := service.registerMediaMTXChannel(ctx, device.ID, fmt.Sprintf("channel%d", channel)); err != nil {
			t.Fatalf("register channel%d: %v", channel, err)
		}
	}
	var stored models.OrangePi
	if err := db.First(&stored, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored.ChannelPaths) != 120 || stored.ChannelPaths["channel120"] != "channel120" {
		t.Fatalf("expected 120 available paths, got %v", stored.ChannelPaths)
	}
	if !reflect.DeepEqual(stored.AllChannels, []int{1}) {
		t.Fatalf("discovery changed selected staff channels: %v", stored.AllChannels)
	}
	if !reflect.DeepEqual(stored.UserChannels, []int{1}) {
		t.Fatalf("registration changed user visibility: %v", stored.UserChannels)
	}
	var link models.OrangePiBuilding
	if err := db.Where("orange_pi_id = ? AND building_id = ?", device.ID, building.ID).First(&link).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(link.AllowedChannels, []int{1}) {
		t.Fatalf("registration changed building visibility: %v", link.AllowedChannels)
	}

	buildingService := NewBuildingService(db)
	authService := NewAuthService(db, nil, service, buildingService)
	before, err := authService.GeneratePublicToken(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.OrangePis) != 1 || len(before.OrangePis[0].URLs) != 1 ||
		!strings.Contains(before.OrangePis[0].URLs[0], "/channel1?") {
		t.Fatalf("registration unexpectedly expanded public URLs: %+v", before)
	}

	if _, err := service.Update(ctx, device.ID, models.OrangePi{
		UserChannels: []int{1, 100}, AllChannels: []int{1, 100},
	}); err != nil {
		t.Fatal(err)
	}
	if err := buildingService.UpdateOrangePiChannels(ctx, building.ID, device.ID, []int{100}); err != nil {
		t.Fatalf("allow channel100: %v", err)
	}
	after, err := authService.GeneratePublicToken(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.OrangePis) != 1 || len(after.OrangePis[0].URLs) != 1 ||
		!strings.Contains(after.OrangePis[0].URLs[0], "/channel100?") {
		t.Fatalf("explicitly allowed high channel is missing: %+v", after)
	}
}

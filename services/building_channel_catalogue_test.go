package services

import (
	"context"
	"reflect"
	"testing"

	"icctv-http-service/models"
)

func TestBuildingChannelsCanSelectUnselectedCataloguePaths(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	ctx := context.Background()
	buildings := []models.Building{
		{ISmartID: "catalogue-a", Name: "Building A"},
		{ISmartID: "catalogue-b", Name: "Building B"},
	}
	if err := db.Create(&buildings).Error; err != nil {
		t.Fatal(err)
	}
	service := NewOrangePiService(db, NewPublicNetService(db))
	device, err := service.Create(ctx, models.OrangePi{
		ISmartIDs: []string{"catalogue-a", "catalogue-b"}, Name: "Catalogue device",
		AllChannels: []int{1, 2, 3, 4, 5, 6, 7, 8, 9}, UserChannels: []int{1, 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	device.ChannelPaths = map[string]string{
		"channel12": "BuildingGate", "channel17": "MainGateFisheye",
		"channel20": "RoofStaircase",
	}
	if err := db.Model(device).Select("ChannelPaths").Updates(device).Error; err != nil {
		t.Fatal(err)
	}
	buildingService := NewBuildingService(db)
	before, err := buildingService.GetOrangePisByBuildingID(ctx, buildings[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := buildingService.UpdateOrangePiChannels(ctx, buildings[0].ID, device.ID, []int{17, 12, 17}); err != nil {
		t.Fatalf("known paths outside selected channels must be saveable: %v", err)
	}
	if err := buildingService.UpdateOrangePiChannels(ctx, buildings[0].ID, device.ID, []int{999}); err == nil {
		t.Fatal("unknown channel must be rejected")
	}
	if err := buildingService.UpdateOrangePiChannels(ctx, buildings[0].ID, device.ID, nil); err == nil {
		t.Fatal("empty selection must be rejected")
	}
	fromA, err := buildingService.GetOrangePisByBuildingID(ctx, buildings[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	fromB, err := buildingService.GetOrangePisByBuildingID(ctx, buildings[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromA[0].BuildingChannels, []int{17, 12}) {
		t.Fatalf("selection not preserved after rejected saves: %v", fromA[0].BuildingChannels)
	}
	if !reflect.DeepEqual(fromB[0].BuildingChannels, before[0].BuildingChannels) {
		t.Fatal("saving building A changed building B")
	}
	if !reflect.DeepEqual(fromA[0].AllChannels, device.AllChannels) ||
		!reflect.DeepEqual(fromA[0].UserChannels, device.UserChannels) {
		t.Fatal("saving building visibility changed device permissions")
	}
	// New bindings retain their existing permission defaults, even with a full
	// catalogue already present.
	third := models.Building{ISmartID: "catalogue-c", Name: "Building C"}
	if err := db.Create(&third).Error; err != nil {
		t.Fatal(err)
	}
	if err := buildingService.BindOrangePi(ctx, third.ID, device.ID); err != nil {
		t.Fatal(err)
	}
	fromC, err := buildingService.GetOrangePisByBuildingID(ctx, third.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromC[0].BuildingChannels, device.AllChannels) {
		t.Fatalf("new binding expanded visibility: %v", fromC[0].BuildingChannels)
	}
}

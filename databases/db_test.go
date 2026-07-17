package databases

import (
	"testing"

	"icctv-http-service/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateLegacyOrangePiBuildings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:legacy-migration?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.SetupJoinTable(&models.OrangePi{}, "Buildings", &models.OrangePiBuilding{}); err != nil {
		t.Fatalf("set up join table: %v", err)
	}
	if err := db.AutoMigrate(&models.Building{}, &models.OrangePi{}, &models.OrangePiBuilding{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}

	building := models.Building{ISmartID: "legacy-building", Name: "Legacy Building"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatalf("create building: %v", err)
	}
	device := models.OrangePi{
		ISmartID:                   building.ISmartID,
		Name:                       "Legacy Orange Pi",
		ICCTVAuthServiceRemotePort: 29005,
		SSHRemotePort:              30005,
	}
	if err := db.Omit("Buildings").Create(&device).Error; err != nil {
		t.Fatalf("create legacy orangepi: %v", err)
	}

	if err := migrateLegacyOrangePiBuildings(db); err != nil {
		t.Fatalf("migrate legacy association: %v", err)
	}
	if err := migrateLegacyOrangePiBuildings(db); err != nil {
		t.Fatalf("repeat legacy migration: %v", err)
	}

	var links []models.OrangePiBuilding
	if err := db.Find(&links).Error; err != nil {
		t.Fatalf("list associations: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected one association after idempotent migration, got %d", len(links))
	}
	if links[0].OrangePiID != device.ID || links[0].BuildingID != building.ID {
		t.Fatalf("unexpected association: %+v", links[0])
	}
}

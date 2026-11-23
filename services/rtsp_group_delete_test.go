package services

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"icctv-http-service/models"
)

func TestSyncRTSPGroupDeletesExclusiveManagedPath(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	if err := db.AutoMigrate(&models.NVR{}); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasColumn(&models.NVR{}, "orange_pi_id") {
		t.Fatalf("nvr schema does not contain orange_pi_id")
	}
	building := models.Building{ISmartID: "delete-building", Name: "Delete building"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatal(err)
	}
	var deleted []string
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/mediamtx/paths":
			_ = json.NewEncoder(w).Encode(MediaMTXPathsListResponse{
				Items:      []MediaMTXPathItem{{Name: "channel1", Source: "rtsp://camera.example/1"}},
				ItemsTotal: 1,
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/mediamtx/paths/channel1":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"source": "rtsp://camera.example/1"})
		case r.Method == http.MethodDelete && r.URL.Path == "/api/device/mediamtx/paths/channel1":
			deleted = append(deleted, r.URL.Path)
			_ = json.NewEncoder(w).Encode(MediaMTXPathActionResponse{
				Action: "delete", Name: "channel1", StatusCode: http.StatusOK,
			})
		default:
			t.Errorf("unexpected device request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer remote.Close()
	host, portText, _ := net.SplitHostPort(strings.TrimPrefix(remote.URL, "http://"))
	port, _ := strconv.Atoi(portText)
	if err := db.Create(&models.PublicNetConfig{ExternalIP: host}).Error; err != nil {
		t.Fatal(err)
	}
	device := models.OrangePi{
		Name:                       "Delete device",
		ISmartID:                   building.ISmartID,
		ICCTVAuthServiceRemotePort: port,
	}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	group := models.NVR{
		Name:       "Delete group",
		BuildingID: building.ID,
		OrangePiID: device.ID,
		PendingRTSPDeletes: []models.ChannelURL{{
			ID: "camera-1", Path: "channel1", URL: "rtsp://camera.example/1",
			SyncStatus: "success", SyncedOrangePiID: device.ID,
		}},
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}

	service := NewOrangePiService(db, NewPublicNetService(db))
	service.SetTokenGenerator(func(context.Context, string, int64, bool) (string, error) {
		return "test-token", nil
	})
	loaded := group
	if err := db.First(&loaded, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SyncRTSPGroup(context.Background(), &loaded); err != nil {
		t.Fatalf("sync group: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != "/api/device/mediamtx/paths/channel1" {
		t.Fatalf("expected channel1 delete, got %v", deleted)
	}
	if len(loaded.PendingRTSPDeletes) != 0 {
		t.Fatalf("completed delete remained queued: %+v", loaded.PendingRTSPDeletes)
	}
}

func TestSyncRTSPGroupKeepsPathReferencedByAnotherGroup(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	if err := db.AutoMigrate(&models.NVR{}); err != nil {
		t.Fatal(err)
	}
	building := models.Building{ISmartID: "shared-delete-building", Name: "Shared delete building"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatal(err)
	}
	var deleteCalls int
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/device/mediamtx/paths" {
			_ = json.NewEncoder(w).Encode(MediaMTXPathsListResponse{
				Items: []MediaMTXPathItem{{Name: "channel1"}}, ItemsTotal: 1,
			})
			return
		}
		if r.Method == http.MethodDelete {
			deleteCalls++
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer remote.Close()
	host, portText, _ := net.SplitHostPort(strings.TrimPrefix(remote.URL, "http://"))
	port, _ := strconv.Atoi(portText)
	if err := db.Create(&models.PublicNetConfig{ExternalIP: host}).Error; err != nil {
		t.Fatal(err)
	}
	device := models.OrangePi{Name: "Shared delete device", ISmartID: building.ISmartID, ICCTVAuthServiceRemotePort: port}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	group := models.NVR{
		Name: "Delete group", BuildingID: building.ID, OrangePiID: device.ID,
		PendingRTSPDeletes: []models.ChannelURL{{Path: "channel1", SyncStatus: "success", SyncedOrangePiID: device.ID}},
	}
	other := models.NVR{
		Name: "Other group", OrangePiID: device.ID,
		RTSPUrls: []models.ChannelURL{{Path: "channel1", URL: "rtsp://camera.example/other"}},
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	service := NewOrangePiService(db, NewPublicNetService(db))
	service.SetTokenGenerator(func(context.Context, string, int64, bool) (string, error) {
		return "test-token", nil
	})
	loaded := group
	if err := db.First(&loaded, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SyncRTSPGroup(context.Background(), &loaded); err == nil ||
		!strings.Contains(err.Error(), "another RTSP group references it") {
		t.Fatalf("expected shared path error, got %v", err)
	}
	if deleteCalls != 0 {
		t.Fatalf("shared path was deleted")
	}
	if len(loaded.PendingRTSPDeletes) != 1 {
		t.Fatalf("shared path should remain queued: %+v", loaded.PendingRTSPDeletes)
	}
}

func TestRTSPRemovalPersistsAndRetriesDeletesSafely(t *testing.T) {
	for _, mode := range []string{"success", "missing", "source-changed", "http-failure", "embedded-failure", "unknown-response"} {
		t.Run(mode, func(t *testing.T) {
			db := newMultiBuildingTestDB(t)
			if err := db.AutoMigrate(&models.NVR{}, &models.RTSPSource{}); err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			building := models.Building{Name: "Test", ISmartID: "delete-test"}
			if err := db.Create(&building).Error; err != nil {
				t.Fatal(err)
			}
			currentMode := mode
			deleteCalls := 0
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/device/mediamtx/paths":
					items := []MediaMTXPathItem{}
					if currentMode != "missing" {
						items = append(items, MediaMTXPathItem{Name: "channel1"})
					}
					_ = json.NewEncoder(w).Encode(MediaMTXPathsListResponse{Items: items, ItemsTotal: len(items)})
				case r.Method == http.MethodGet && r.URL.Path == "/api/device/mediamtx/paths/channel1":
					source := "rtsp://camera.example/1"
					if currentMode == "source-changed" {
						source = "rtsp://camera.example/replaced"
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"source": source})
				case r.Method == http.MethodDelete:
					deleteCalls++
					switch currentMode {
					case "http-failure":
						w.WriteHeader(http.StatusBadGateway)
					case "embedded-failure":
						_ = json.NewEncoder(w).Encode(MediaMTXPathActionResponse{StatusCode: http.StatusConflict})
					case "unknown-response":
						_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "unknown"})
					default:
						_ = json.NewEncoder(w).Encode(MediaMTXPathActionResponse{StatusCode: http.StatusOK})
					}
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer remote.Close()
			host, portText, _ := net.SplitHostPort(strings.TrimPrefix(remote.URL, "http://"))
			port, _ := strconv.Atoi(portText)
			if err := db.Create(&models.PublicNetConfig{ExternalIP: host}).Error; err != nil {
				t.Fatal(err)
			}
			device := models.OrangePi{Name: "Test device", ISmartID: building.ISmartID, ICCTVAuthServiceRemotePort: port}
			if err := db.Create(&device).Error; err != nil {
				t.Fatal(err)
			}
			group := models.NVR{
				Name: "Test group", BuildingID: building.ID, OrangePiID: device.ID,
				RTSPUrls: []models.ChannelURL{{
					ID: "camera-1", Path: "channel1", URL: "rtsp://camera.example/1",
					SyncStatus: "success", SyncedOrangePiID: device.ID,
				}},
			}
			if err := db.Create(&group).Error; err != nil {
				t.Fatal(err)
			}
			deviceService := NewOrangePiService(db, NewPublicNetService(db))
			deviceService.SetTokenGenerator(func(context.Context, string, int64, bool) (string, error) {
				return "test-token", nil
			})
			service := NewNVRService(db, deviceService.SyncRTSPGroup)
			_, err := service.Update(ctx, group.ID, models.NVR{RTSPUrls: []models.ChannelURL{}})
			wantSuccess := mode == "success" || mode == "missing"
			if (err == nil) != wantSuccess {
				t.Fatalf("unexpected result: %v", err)
			}
			var stored models.NVR
			if err := db.First(&stored, group.ID).Error; err != nil {
				t.Fatal(err)
			}
			if len(stored.RTSPUrls) != 0 || (len(stored.PendingRTSPDeletes) == 0) != wantSuccess {
				t.Fatalf("removal or pending deletion not persisted: %+v", stored)
			}
			if (mode == "missing" || mode == "source-changed") && deleteCalls != 0 {
				t.Fatal("unverified or missing path must not receive DELETE")
			}
			if !wantSuccess {
				// A later save retries the persisted task after the failure is resolved.
				currentMode = "success"
				if _, err := service.Update(ctx, group.ID, models.NVR{Name: "Retry"}); err != nil {
					t.Fatalf("retry failed: %v", err)
				}
				var retried models.NVR
				if err := db.First(&retried, group.ID).Error; err != nil {
					t.Fatal(err)
				}
				if len(retried.PendingRTSPDeletes) != 0 {
					t.Fatal("successful retry must clear the task")
				}
			}
		})
	}
}

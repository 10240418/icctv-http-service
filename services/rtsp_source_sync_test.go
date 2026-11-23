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

func TestLibrarySourceChangeSyncChecksOwnershipAndRetries(t *testing.T) {
	for _, mode := range []string{"success", "source-changed", "missing", "http-failure"} {
		t.Run(mode, func(t *testing.T) {
			db := newMultiBuildingTestDB(t)
			if err := db.AutoMigrate(&models.NVR{}, &models.RTSPSource{}); err != nil {
				t.Fatal(err)
			}
			currentMode := mode
			remoteSource := "rtsp://camera/old"
			patches := 0
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "GET" && r.URL.Path == "/api/device/mediamtx/paths":
					items := []MediaMTXPathItem{}
					if currentMode != "missing" {
						source := remoteSource
						if currentMode == "source-changed" {
							source = "rtsp://external/replaced"
						}
						items = append(items, MediaMTXPathItem{Name: "channel1", Source: source})
					}
					_ = json.NewEncoder(w).Encode(MediaMTXPathsListResponse{Items: items, ItemsTotal: len(items)})
				case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/channel1"):
					source := remoteSource
					if currentMode == "source-changed" {
						source = "rtsp://external/replaced"
					}
					_ = json.NewEncoder(w).Encode(map[string]string{"source": source})
				case r.Method == "PATCH" && strings.HasSuffix(r.URL.Path, "/channel1"):
					patches++
					if currentMode == "http-failure" {
						w.WriteHeader(http.StatusBadGateway)
						return
					}
					remoteSource = "rtsp://camera/new"
					_ = json.NewEncoder(w).Encode(MediaMTXPathActionResponse{StatusCode: http.StatusOK})
				default:
					t.Errorf("unexpected device operation: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer remote.Close()
			host, portText, _ := net.SplitHostPort(strings.TrimPrefix(remote.URL, "http://"))
			port, _ := strconv.Atoi(portText)
			if err := db.Create(&models.PublicNetConfig{ExternalIP: host}).Error; err != nil {
				t.Fatal(err)
			}
			building := models.Building{Name: "Test", ISmartID: "source-sync"}
			if err := db.Create(&building).Error; err != nil {
				t.Fatal(err)
			}
			device := models.OrangePi{Name: "Test", ISmartID: building.ISmartID, ICCTVAuthServiceRemotePort: port}
			if err := db.Create(&device).Error; err != nil {
				t.Fatal(err)
			}
			entry := models.RTSPSource{URL: remoteSource}
			if err := db.Create(&entry).Error; err != nil {
				t.Fatal(err)
			}
			group := models.NVR{Name: "Test", BuildingID: building.ID, OrangePiID: device.ID,
				RTSPUrls: []models.ChannelURL{{
					ID: "camera-1", SourceID: entry.ID, Path: "channel1", URL: remoteSource,
					SyncStatus: "success", SyncedOrangePiID: device.ID, SyncedSource: remoteSource,
				}},
			}
			if err := db.Create(&group).Error; err != nil {
				t.Fatal(err)
			}
			deviceService := NewOrangePiService(db, NewPublicNetService(db))
			deviceService.SetTokenGenerator(func(context.Context, string, int64, bool) (string, error) { return "qa", nil })
			groups := NewNVRService(db, deviceService.SyncRTSPGroup)
			library := NewRTSPSourceService(db, groups)
			result, err := library.Update(context.Background(), entry.ID, RTSPSourceUpdate{
				URL: "rtsp://camera/new", ExpectedVersion: entry.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
				ConfirmedGroups: []int64{group.ID},
			})
			if err != nil {
				t.Fatal(err)
			}
			if (len(result.SyncErrors) == 0) != (mode == "success") {
				t.Fatalf("sync failure reporting: %+v", result)
			}
			if (mode == "source-changed" || mode == "missing") && patches != 0 {
				t.Fatal("external or missing path must not be overwritten")
			}
			stored, err := groups.GetByID(context.Background(), group.ID)
			if err != nil || stored.RTSPUrls[0].URL != "rtsp://camera/new" {
				t.Fatalf("desired source not persisted: %+v %v", stored, err)
			}
			if mode != "success" {
				if stored.RTSPUrls[0].SyncStatus == "success" {
					t.Fatal("failed new source must not remain authorized")
				}
				currentMode = "success"
				if _, err := groups.Update(context.Background(), group.ID, models.NVR{Name: "Retry"}); err != nil {
					t.Fatal(err)
				}
			}
			stored, err = groups.GetByID(context.Background(), group.ID)
			if err != nil || stored.RTSPUrls[0].SyncStatus != "success" || stored.RTSPUrls[0].SyncedSource != "rtsp://camera/new" {
				t.Fatalf("sync or retry did not finish: %+v %v", stored, err)
			}
		})
	}
}

func TestExplicitRTSPGroupUnbindKeepsLibraryAndDeviceHistory(t *testing.T) {
	groups := newRTSPGroupTestService(t)
	ctx := context.Background()
	group := models.NVR{Name: "Bound", BuildingID: 1, OrangePiID: 7,
		RTSPUrls: []models.ChannelURL{{Path: "channel1", URL: "rtsp://camera/1", SyncStatus: "success", SyncedOrangePiID: 7}},
	}
	if err := groups.db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	updated, err := groups.Update(ctx, group.ID, models.NVR{Binding: &models.RTSPGroupBinding{}})
	if err != nil || updated.BuildingID != 0 || updated.OrangePiID != 0 || updated.RTSPUrls[0].SyncedOrangePiID != 7 {
		t.Fatalf("explicit unbind lost history or failed: %+v %v", updated, err)
	}
}

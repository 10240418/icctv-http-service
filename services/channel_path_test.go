package services

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"icctv-http-service/models"
)

func TestChannelPathValidationPrecedesDeviceAccess(t *testing.T) {
	// A nil database proves rejected requests never reach the device or database.
	service := NewOrangePiService(nil, nil)
	for _, name := range []string{"", "SO_MAIN", "channel_SO_MAIN", "Channel1", "channel0", "channel01", "channel-1", "channel1/whep", "channel1?x=y", " channel1", "channel9007199254740992"} {
		t.Run(name, func(t *testing.T) {
			config := map[string]interface{}{"source": "rtsp://camera.example/SO_MAIN"}
			if _, err := service.RemoteAddMediaMTXPath(context.Background(), 1, "token", name, config); !errors.Is(err, ErrInvalidChannelPath) {
				t.Fatalf("add did not reject %q: %v", name, err)
			}
			if _, err := service.RemoteUpdateMediaMTXPath(context.Background(), 1, "token", name, config); !errors.Is(err, ErrInvalidChannelPath) {
				t.Fatalf("update did not reject %q: %v", name, err)
			}
		})
	}
	for _, config := range []map[string]interface{}{nil, {"name": "SO_MAIN"}} {
		if _, err := service.RemoteAddMediaMTXPath(context.Background(), 1, "token", "channel120", config); !errors.Is(err, ErrInvalidChannelPath) {
			t.Fatalf("invalid config was accepted: %v", err)
		}
		if _, err := service.RemoteUpdateMediaMTXPath(context.Background(), 1, "token", "channel120", config); !errors.Is(err, ErrInvalidChannelPath) {
			t.Fatalf("invalid update config was accepted: %v", err)
		}
	}
	for _, name := range []string{"channel1", "channel10", "channel120", "channel9007199254740991"} {
		if err := ValidateChannelPath(name); err != nil {
			t.Fatalf("valid path %s rejected: %v", name, err)
		}
	}
}

func TestCanonicalPathMutationsPreserveSourceAndPermissions(t *testing.T) {
	for _, mode := range []string{"success", "http-failure", "embedded-failure"} {
		t.Run(mode, func(t *testing.T) {
			db := newMultiBuildingTestDB(t)
			source := "rtsp://camera.example/SO_MAIN?channel=1&subtype=0"
			var methods []string
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.URL.Query().Get("token") != "test-token" {
					t.Error("missing token")
				}
				var body struct {
					Name   string                 `json:"name"`
					Config map[string]interface{} `json:"config"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Config["source"] != source || body.Config["sourceOnDemand"] != false || body.Config["record"] != false {
					t.Errorf("source or fixed settings changed: %v", body)
				}
				if r.Method == http.MethodPost && (body.Name != "channel120" || r.URL.Path != "/api/device/mediamtx/paths") {
					t.Errorf("unexpected add: %s, %+v", r.URL.Path, body)
				}
				if r.Method == http.MethodPatch && r.URL.Path != "/api/device/mediamtx/paths/channel120" {
					t.Errorf("unexpected update: %s", r.URL.Path)
				}
				if mode == "http-failure" {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				status := 200
				if mode == "embedded-failure" {
					status = 409
				}
				_ = json.NewEncoder(w).Encode(MediaMTXPathActionResponse{Name: "channel120", StatusCode: status})
			}))
			defer remote.Close()
			host, portText, _ := net.SplitHostPort(strings.TrimPrefix(remote.URL, "http://"))
			port, _ := strconv.Atoi(portText)
			if err := db.Create(&models.PublicNetConfig{ExternalIP: host}).Error; err != nil {
				t.Fatal(err)
			}
			device := models.OrangePi{Name: "Device", ISmartID: "building", ICCTVAuthServiceRemotePort: port,
				AllChannels: []int{1}, UserChannels: []int{1}, ChannelPaths: map[string]string{"channel10": "SO_MAIN"},
				ChannelRemarks: map[string]string{"channel10": "Entrance"}}
			if err := db.Create(&device).Error; err != nil {
				t.Fatal(err)
			}
			link := models.OrangePiBuilding{OrangePiID: device.ID, BuildingID: 1, AllowedChannels: []int{1}}
			if err := db.Create(&link).Error; err != nil {
				t.Fatal(err)
			}
			service := NewOrangePiService(db, NewPublicNetService(db))
			for _, update := range []bool{false, true} {
				config := map[string]interface{}{"source": source, "record": true, "sourceOnDemand": true}
				var err error
				if update {
					_, err = service.RemoteUpdateMediaMTXPath(context.Background(), device.ID, "test-token", "channel120", config)
				} else {
					_, err = service.RemoteAddMediaMTXPath(context.Background(), device.ID, "test-token", "channel120", config)
				}
				if (err == nil) != (mode == "success") {
					t.Fatalf("unexpected result mode=%s update=%v: %v", mode, update, err)
				}
			}
			if !reflect.DeepEqual(methods, []string{http.MethodPost, http.MethodPatch}) {
				t.Fatalf("unexpected device mutations: %v", methods)
			}
			var stored models.OrangePi
			if err := db.First(&stored, device.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.ChannelPaths["channel10"] != "SO_MAIN" || stored.ChannelRemarks["channel10"] != "Entrance" ||
				!reflect.DeepEqual(stored.UserChannels, []int{1}) || !reflect.DeepEqual(stored.AllChannels, []int{1}) {
				t.Fatalf("original mapping or permissions changed: %+v", stored)
			}
			if (stored.ChannelPaths["channel120"] == "channel120") != (mode == "success") {
				t.Fatalf("failed create registered a path: %v", stored.ChannelPaths)
			}
			var storedLink models.OrangePiBuilding
			if err := db.First(&storedLink).Error; err != nil || !reflect.DeepEqual(storedLink.AllowedChannels, []int{1}) {
				t.Fatalf("building permissions changed: %+v %v", storedLink, err)
			}
			for _, name := range []string{"channel1", "channel10"} {
				if _, err := service.RemoteAddMediaMTXPath(context.Background(), device.ID, "test-token", name, map[string]interface{}{}); !errors.Is(err, ErrReservedChannelPath) {
					t.Fatalf("reserved ID accepted: %s %v", name, err)
				}
			}
			if len(methods) != 2 {
				t.Fatal("reserved ID reached device")
			}
		})
	}
}

func TestDevicePathDetailSupportsNativeAndWrappedConfig(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		t.Run(strconv.FormatBool(wrapped), func(t *testing.T) {
			db := newMultiBuildingTestDB(t)
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/device/mediamtx/paths/SO_MAIN" {
					t.Errorf("unexpected device access: %s %s", r.Method, r.URL.Path)
				}
				conf := map[string]interface{}{"source": "rtsp://camera.example/SO_MAIN", "record": false}
				if wrapped {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"name": "SO_MAIN", "conf": conf})
				} else {
					_ = json.NewEncoder(w).Encode(conf)
				}
			}))
			defer remote.Close()
			host, portText, _ := net.SplitHostPort(strings.TrimPrefix(remote.URL, "http://"))
			port, _ := strconv.Atoi(portText)
			if err := db.Create(&models.PublicNetConfig{ExternalIP: host}).Error; err != nil {
				t.Fatal(err)
			}
			device := models.OrangePi{Name: "Device", ISmartID: "building", ICCTVAuthServiceRemotePort: port}
			if err := db.Create(&device).Error; err != nil {
				t.Fatal(err)
			}
			result, err := NewOrangePiService(db, NewPublicNetService(db)).RemoteGetMediaMTXPath(context.Background(), device.ID, "test-token", "SO_MAIN")
			if err != nil || result.Name != "SO_MAIN" || result.Conf["source"] != "rtsp://camera.example/SO_MAIN" {
				t.Fatalf("detail lost original source: %+v %v", result, err)
			}
		})
	}
}

func TestGroupWritesRequireCanonicalPathsButLegacyReadsRemainAvailable(t *testing.T) {
	service := newRTSPGroupTestService(t)
	ctx := context.Background()
	group := models.NVR{Name: "Legacy", RTSPUrls: []models.ChannelURL{{Path: "SO_MAIN", URL: "rtsp://camera.example/SO_MAIN"}}}
	if err := service.db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	loaded, err := service.GetByID(ctx, group.ID)
	if err != nil || loaded.RTSPUrls[0].Path != "SO_MAIN" {
		t.Fatalf("legacy read failed: %+v %v", loaded, err)
	}
	invalid := []models.ChannelURL{{Path: "channel_SO_MAIN", URL: "rtsp://camera.example/SO_MAIN"}}
	if _, err := service.Update(ctx, group.ID, models.NVR{RTSPUrls: invalid}); !errors.Is(err, ErrInvalidChannelPath) {
		t.Fatalf("group update accepted invalid path: %v", err)
	}
	if _, err := service.UpdateRTSPUrls(ctx, group.ID, invalid); !errors.Is(err, ErrInvalidChannelPath) {
		t.Fatalf("URL update accepted invalid path: %v", err)
	}
	if _, err := service.AddRTSPUrl(ctx, group.ID, invalid[0]); !errors.Is(err, ErrInvalidChannelPath) {
		t.Fatalf("URL add accepted invalid path: %v", err)
	}
	valid := []models.ChannelURL{{Path: "channel120", Channel: 1, URL: "rtsp://camera.example/SO_MAIN", Remark: "SO_MAIN"}}
	updated, err := service.Update(ctx, group.ID, models.NVR{RTSPUrls: valid})
	if err != nil || updated.RTSPUrls[0].Channel != 120 || updated.RTSPUrls[0].URL != valid[0].URL {
		t.Fatalf("canonical group save lost source or channel: %+v %v", updated, err)
	}
}

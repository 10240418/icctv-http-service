package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"icctv-http-service/models"
)

func TestNamedPathIDsStayStableAndDoNotGrantVisibility(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	ctx := context.Background()
	service := NewOrangePiService(db, NewPublicNetService(db))
	building := models.Building{ISmartID: "named-building", Name: "Named building"}
	if err := db.Create(&building).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.PublicNetConfig{ExternalIP: "203.0.113.10"}).Error; err != nil {
		t.Fatal(err)
	}
	device, err := service.Create(ctx, models.OrangePi{
		ISmartIDs: []string{building.ISmartID}, Name: "Named paths",
		ICCTVAuthServiceRemotePort: 29005, SSHRemotePort: 30005,
		AllChannels: []int{1, 2, 3, 4, 5, 6, 7, 8, 9}, UserChannels: []int{1},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Include a legacy unrestricted building rule; discovery must not expand its
	// effective visibility either.
	if err := db.Model(&models.OrangePiBuilding{}).Where("orange_pi_id = ?", device.ID).
		Update("allowed_channels", "[]").Error; err != nil {
		t.Fatal(err)
	}
	auth := NewAuthService(db, nil, service, NewBuildingService(db))
	before, err := auth.GeneratePublicToken(ctx, building.ISmartID, true)
	if err != nil {
		t.Fatal(err)
	}
	items := []MediaMTXPathItem{{Name: "MainGateFisheye"}, {Name: "BuildingGate"}, {Name: "channel01"}}
	for i := 0; i < 120; i++ {
		items = append(items, MediaMTXPathItem{Name: fmt.Sprintf("camera-%03d", i)})
	}
	items = append(items, MediaMTXPathItem{Name: "all_others"}, MediaMTXPathItem{Name: "~^camera.*"}, MediaMTXPathItem{})
	if err := service.registerMediaMTXPaths(ctx, device.ID, items); err != nil {
		t.Fatal(err)
	}
	var stored models.OrangePi
	if err := db.First(&stored, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	originalPaths := stored.ChannelPaths
	if len(originalPaths) != 132 || originalPaths["channel10"] != "BuildingGate" {
		t.Fatalf("unexpected named catalogue: %v", originalPaths)
	}
	after, err := auth.GeneratePublicToken(ctx, building.ISmartID, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.OrangePis[0].URLs) != len(after.OrangePis[0].URLs) {
		t.Fatalf("discovery changed staff visibility")
	}
	for i, value := range before.OrangePis[0].URLs {
		beforeURL, _ := url.Parse(value)
		afterURL, _ := url.Parse(after.OrangePis[0].URLs[i])
		if beforeURL.Path != afterURL.Path {
			t.Fatalf("discovery changed staff URL path: %s -> %s", beforeURL.Path, afterURL.Path)
		}
	}
	// A new real channel10 must not steal the ID already used by BuildingGate.
	items = append(items, MediaMTXPathItem{Name: "channel10"})
	if err := service.registerMediaMTXPaths(ctx, device.ID, items); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&stored, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	for key, path := range originalPaths {
		if stored.ChannelPaths[key] != path {
			t.Fatalf("remapped %s from %s to %s", key, path, stored.ChannelPaths[key])
		}
	}
	if stored.ChannelPaths["channel133"] != "channel10" {
		t.Fatalf("new numeric path lost after ID collision: %v", stored.ChannelPaths)
	}
	if err := service.registerMediaMTXPaths(ctx, device.ID, []MediaMTXPathItem{{Name: "BuildingGate"}}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&stored, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored.ChannelPaths) != 133 || !reflect.DeepEqual(stored.UserChannels, []int{1}) {
		t.Fatalf("resync lost paths or changed permissions: %+v", stored)
	}

	if _, err := service.Update(ctx, device.ID, models.OrangePi{
		AllChannels: []int{1, 10}, UserChannels: []int{1, 10},
		ChannelRemarks: map[string]string{"channel10": "Gate camera"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := NewBuildingService(db).UpdateOrangePiChannels(ctx, building.ID, device.ID, []int{10}); err != nil {
		t.Fatal(err)
	}
	public, err := auth.GeneratePublicToken(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatal(err)
	}
	permanent, err := auth.GeneratePermanentToken(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatal(err)
	}
	v2, err := auth.GeneratePublicTokenV2(ctx, building.ISmartID, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range []OrangePiURLs{public.OrangePis[0], permanent.OrangePis[0], {
		Token: v2.OrangePis[0].Token, URLs: v2.OrangePis[0].URLs,
	}} {
		if len(entry.URLs) != 1 {
			t.Fatalf("unexpected URLs: %v", entry.URLs)
		}
		parsed, err := url.Parse(entry.URLs[0])
		if err != nil || parsed.Path != "/BuildingGate" || parsed.Query().Get("token") != entry.Token {
			t.Fatalf("URL does not use real path: %v", entry.URLs)
		}
		payloadBytes, err := base64.URLEncoding.DecodeString(strings.Split(entry.Token, ".")[0])
		if err != nil {
			t.Fatal(err)
		}
		var payload VideoTokenPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(payload.Channels, []string{"BuildingGate"}) {
			t.Fatalf("token does not use exact path permissions: %+v", payload)
		}
	}
	if !reflect.DeepEqual(v2.OrangePis[0].ChannelRemarks, map[string]string{"BuildingGate": "Gate camera"}) {
		t.Fatalf("remarks do not match real URL path: %v", v2.OrangePis[0].ChannelRemarks)
	}
}

func TestSyncMediaMTXChannelsReadsAllPagesAndFailsAtomically(t *testing.T) {
	for _, mode := range []string{"success", "native-pagination", "http-failure", "incomplete", "duplicate", "count-changed"} {
		t.Run(mode, func(t *testing.T) {
			db := newMultiBuildingTestDB(t)
			var requested []int
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/device/mediamtx/paths" {
					t.Errorf("unexpected device request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if r.URL.Query().Get("token") != "test-token" {
					t.Errorf("missing device token")
				}
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				requested = append(requested, page)
				if page == 1 && mode == "http-failure" {
					w.WriteHeader(http.StatusBadGateway)
					return
				}
				items := make([]MediaMTXPathItem, 0)
				for i := page * 50; i < (page+1)*50 && i < 51; i++ {
					items = append(items, MediaMTXPathItem{Name: fmt.Sprintf("Lobby-%d", i)})
				}
				total := 51
				if page == 1 {
					switch mode {
					case "incomplete":
						items = nil
					case "duplicate":
						items = []MediaMTXPathItem{{Name: "Lobby-0"}}
					case "count-changed":
						total = 52
					}
				}
				if mode == "native-pagination" {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"items": items, "itemCount": total, "pageCount": 2,
					})
				} else {
					_ = json.NewEncoder(w).Encode(MediaMTXPathsListResponse{Items: items, ItemsTotal: total})
				}
			}))
			defer remote.Close()
			host, portText, _ := net.SplitHostPort(strings.TrimPrefix(remote.URL, "http://"))
			port, _ := strconv.Atoi(portText)
			if err := db.Create(&models.PublicNetConfig{ExternalIP: host}).Error; err != nil {
				t.Fatal(err)
			}
			device := models.OrangePi{Name: "Paged device", ISmartID: "legacy",
				ICCTVAuthServiceRemotePort: port, AllChannels: []int{1}, UserChannels: []int{1}}
			if err := db.Create(&device).Error; err != nil {
				t.Fatal(err)
			}
			service := NewOrangePiService(db, NewPublicNetService(db))
			result, err := service.SyncMediaMTXChannels(context.Background(), device.ID, "test-token")
			success := mode == "success" || mode == "native-pagination"
			if !success {
				if err == nil {
					t.Fatal("partial device read must fail")
				}
			} else if err != nil || len(result.ChannelPaths) != 52 {
				t.Fatalf("all paths were not discovered: %+v, %v", result, err)
			}
			if !reflect.DeepEqual(requested, []int{0, 1}) {
				t.Fatalf("missing pages: %v", requested)
			}
			var stored models.OrangePi
			if err := db.First(&stored, device.ID).Error; err != nil {
				t.Fatal(err)
			}
			if !success && len(stored.ChannelPaths) != 0 {
				t.Fatalf("partial read committed catalogue: %v", stored.ChannelPaths)
			}
			if !reflect.DeepEqual(stored.AllChannels, []int{1}) || !reflect.DeepEqual(stored.UserChannels, []int{1}) {
				t.Fatalf("sync changed permission selections: %+v", stored)
			}
		})
	}
}

func TestStreamURLPreservesSpecialPathAndTokenCharacters(t *testing.T) {
	value := streamURL("203.0.113.10", 29005, "Gate #1&camera", "token=+&")
	parsed, err := url.Parse(value)
	if err != nil || parsed.Path != "/Gate #1&camera" || parsed.Query().Get("token") != "token=+&" {
		t.Fatalf("unsafe stream URL: %q, %v", value, err)
	}
}

func TestCatalogueRecognizesExactNonstandardNames(t *testing.T) {
	db := newMultiBuildingTestDB(t)
	device := models.OrangePi{Name: "Exact paths", ISmartID: "legacy", AllChannels: []int{1}}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	service := NewOrangePiService(db, NewPublicNetService(db))
	names := []string{"channel01", "channel9007199254740993", "Lobby/Floor1", "Gate #1", "Camera_\u5927\u5802"}
	items := []MediaMTXPathItem{{Name: "all"}, {Name: "all_others"}, {Name: "~^camera"}}
	for _, name := range names {
		items = append(items, MediaMTXPathItem{Name: name})
	}
	if err := service.registerMediaMTXPaths(context.Background(), device.ID, items); err != nil {
		t.Fatal(err)
	}
	var stored models.OrangePi
	if err := db.First(&stored, device.ID).Error; err != nil {
		t.Fatal(err)
	}
	found := make(map[string]bool)
	for key, name := range stored.ChannelPaths {
		if numericChannel(key) == 0 {
			t.Fatalf("catalogue ID is not a safe positive integer: %s", key)
		}
		found[name] = true
	}
	for _, name := range names {
		if !found[name] {
			t.Fatalf("exact Path name not recognized: %s", name)
		}
	}
	if len(stored.ChannelPaths) != len(names)+1 {
		t.Fatalf("templates should not become channel options: %v", stored.ChannelPaths)
	}
}

func TestMediaMTXPathsPaginationCompatibility(t *testing.T) {
	for _, body := range []string{
		`{"items":[{"name":"BuildingGate"}],"itemCount":20,"pageCount":1}`,
		`{"items":[{"name":"BuildingGate"}],"itemsTotal":20,"itemsPage":0}`,
	} {
		result, err := decodeMediaMTXPaths(strings.NewReader(body), 2)
		if err != nil || result.ItemsTotal != 20 || result.ItemsPage != 2 || result.Items[0].Name != "BuildingGate" {
			t.Fatalf("pagination was not normalized: %+v, %v", result, err)
		}
	}
	for _, body := range []string{`{"items":[]}`, `{"itemCount":-1}`, `{"itemsTotal":"20"}`, `invalid`} {
		if _, err := decodeMediaMTXPaths(strings.NewReader(body), 0); err == nil {
			t.Fatalf("invalid metadata must not be silently accepted: %s", body)
		}
	}
}

func TestMediaMTXPathReadyStatePreservesMissingValue(t *testing.T) {
	var paths []MediaMTXPathItem
	if err := json.Unmarshal([]byte(`[
		{"name":"idle"},
		{"name":"active","ready":true},
		{"name":"inactive","ready":false}
	]`), &paths); err != nil {
		t.Fatal(err)
	}
	if paths[0].Ready != nil {
		t.Fatalf("missing ready state should remain unknown, got %v", *paths[0].Ready)
	}
	if paths[1].Ready == nil || !*paths[1].Ready {
		t.Fatalf("ready=true should be preserved: %+v", paths[1])
	}
	if paths[2].Ready == nil || *paths[2].Ready {
		t.Fatalf("ready=false should be preserved: %+v", paths[2])
	}
}

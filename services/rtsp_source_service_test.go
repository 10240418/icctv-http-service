package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"icctv-http-service/models"
)

func TestRTSPSourceLibraryCRUDAndReferenceProtection(t *testing.T) {
	groups := newRTSPGroupTestService(t)
	library := NewRTSPSourceService(groups.db, groups)
	ctx := context.Background()
	entry, err := library.Create(ctx, models.RTSPSource{URL: "rtsp://camera.example/1", Remark: "Lobby"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := groups.Create(ctx, models.NVR{Name: "First", RTSPUrls: []models.ChannelURL{
		{Path: "channel1", SourceID: entry.ID, URL: "rtsp://stale-or-forged/1"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if group.RTSPUrls[0].URL != entry.URL {
		t.Fatalf("group did not resolve canonical address: %+v", group)
	}
	list, err := library.List(ctx)
	if err != nil || len(list) != 1 || len(list[0].Usages) != 1 {
		t.Fatalf("missing library usage: %+v %v", list, err)
	}
	if err := library.Delete(ctx, entry.ID); !errors.Is(err, ErrRTSPSourceInUse) {
		t.Fatalf("referenced address deleted: %v", err)
	}
	if _, err := groups.UpdateRTSPUrls(ctx, group.ID, []models.ChannelURL{}); err != nil {
		t.Fatal(err)
	}
	list, err = library.List(ctx)
	if err != nil || len(list) != 1 || len(list[0].Usages) != 0 {
		t.Fatalf("unlink deleted source or retained usage: %+v %v", list, err)
	}
	if err := library.Delete(ctx, entry.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.Update(ctx, group.ID, models.NVR{RTSPUrls: []models.ChannelURL{{Path: "channel1", SourceID: entry.ID}}}); err == nil {
		t.Fatal("deleted source must not be selectable")
	}
}

func TestRTSPSourceEditRequiresConfirmationAndUpdatesAllSnapshots(t *testing.T) {
	groups := newRTSPGroupTestService(t)
	library := NewRTSPSourceService(groups.db, groups)
	ctx := context.Background()
	entry, err := library.Create(ctx, models.RTSPSource{URL: "rtsp://camera/old"})
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for _, name := range []string{"A", "B"} {
		group, err := groups.Create(ctx, models.NVR{Name: name, RTSPUrls: []models.ChannelURL{
			{Path: "channel1", SourceID: entry.ID},
		}})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, group.ID)
	}
	payload := RTSPSourceUpdate{
		URL: "rtsp://camera/new", ExpectedVersion: entry.UpdatedAt.Format(time.RFC3339Nano),
		ConfirmedGroups: ids[:1],
	}
	if _, err := library.Update(ctx, entry.ID, payload); !errors.Is(err, ErrRTSPSourceConflict) {
		t.Fatalf("missing affected-group confirmation accepted: %v", err)
	}
	payload.ConfirmedGroups = ids
	result, err := library.Update(ctx, entry.ID, payload)
	if err != nil {
		t.Fatal(err)
	}
	if result.Source.URL != payload.URL {
		t.Fatal("library URL did not update")
	}
	for _, id := range ids {
		group, err := groups.GetByID(ctx, id)
		if err != nil || group.RTSPUrls[0].URL != payload.URL || group.RTSPUrls[0].Path != "channel1" {
			t.Fatalf("snapshot did not update: %+v %v", group, err)
		}
		// A stale form must resolve the current canonical source instead of
		// restoring the URL it originally loaded.
		group.RTSPUrls[0].URL = "rtsp://camera/old"
		updated, err := groups.Update(ctx, id, models.NVR{RTSPUrls: group.RTSPUrls})
		if err != nil || updated.RTSPUrls[0].URL != payload.URL {
			t.Fatalf("stale save restored previous URL: %+v %v", updated, err)
		}
	}
	if _, err := library.Update(ctx, entry.ID, payload); !errors.Is(err, ErrRTSPSourceConflict) {
		t.Fatalf("stale edit accepted: %v", err)
	}
}

func TestRTSPSourceRejectsInvalidURLsAndPreservesLibraryOnGroupDeletion(t *testing.T) {
	groups := newRTSPGroupTestService(t)
	library := NewRTSPSourceService(groups.db, groups)
	ctx := context.Background()
	for _, value := range []string{"https://camera/1", "rtsp://", "rtsp:///missing-host", ""} {
		if _, err := library.Create(ctx, models.RTSPSource{URL: value}); err == nil {
			t.Fatalf("invalid URL accepted: %s", value)
		}
	}
	group, err := groups.Create(ctx, models.NVR{Name: "Legacy API", RTSPUrls: []models.ChannelURL{
		{Path: "channel1", URL: "rtsp://camera/1"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := groups.Update(ctx, group.ID, models.NVR{RTSPUrls: group.RTSPUrls}); err != nil {
		t.Fatal(err)
	}
	if err := groups.Delete(ctx, group.ID); err != nil {
		t.Fatal(err)
	}
	list, err := library.List(ctx)
	if err != nil || len(list) != 1 || len(list[0].Usages) != 0 {
		t.Fatalf("group deletion lost address library: %+v %v", list, err)
	}
}

func TestRTSPSourceReplacementKeepsLastManagedSourceForRetryAndDeletion(t *testing.T) {
	old := models.ChannelURL{
		ID: "camera-1", SourceID: 1, Path: "channel1", URL: "rtsp://camera/old",
		SyncStatus: "success", SyncedOrangePiID: 7,
	}
	updated := []models.ChannelURL{{ID: old.ID, SourceID: 2, Path: old.Path, URL: "rtsp://camera/new"}}
	mergeRTSPSyncState([]models.ChannelURL{old}, updated)
	if updated[0].SyncStatus != "pending" || updated[0].SyncedSource != old.URL || updated[0].SyncedOrangePiID != 7 {
		t.Fatalf("replacement lost ownership or remained authorized: %+v", updated)
	}
	removed := removedRTSPUrls(updated, nil)
	if len(removed) != 1 || removed[0].URL != old.URL || removed[0].SyncStatus != "success" {
		t.Fatalf("removal must verify last successfully managed source: %+v", removed)
	}
}

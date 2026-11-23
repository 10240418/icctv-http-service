package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"icctv-http-service/models"

	"gorm.io/gorm"
)

// Group snapshots and library edits share a lock, including device sync, so a
// stale in-flight save cannot restore an old URL after a library update.
var rtspWriteMu sync.Mutex

var ErrRTSPSourceInUse = errors.New("RTSP 地址仍被組使用，請先從組中移除或更換")
var ErrRTSPSourceConflict = errors.New("RTSP 地址或引用已改變，請重新整理後再試")

type RTSPSourceService struct {
	db     *gorm.DB
	groups *NVRService
}

func NewRTSPSourceService(db *gorm.DB, groups *NVRService) *RTSPSourceService {
	return &RTSPSourceService{db: db, groups: groups}
}

func (s *RTSPSourceService) List(ctx context.Context) ([]models.RTSPSourceView, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	var entries []models.RTSPSource
	if err := s.db.WithContext(ctx).Preload("Building").Order("id").Find(&entries).Error; err != nil {
		return nil, err
	}
	var groups []models.NVR
	if err := s.db.WithContext(ctx).Order("id").Find(&groups).Error; err != nil {
		return nil, err
	}
	result := make([]models.RTSPSourceView, 0, len(entries))
	for _, entry := range entries {
		usages := make([]models.RTSPSourceUsage, 0)
		for _, group := range groups {
			for _, camera := range group.RTSPUrls {
				if camera.SourceID == entry.ID {
					usages = append(usages, models.RTSPSourceUsage{
						GroupID: group.ID, GroupName: group.Name, Path: camera.Path,
						BuildingID: group.BuildingID, OrangePiID: group.OrangePiID,
					})
				}
			}
		}
		result = append(result, models.RTSPSourceView{RTSPSource: entry, Usages: usages})
	}
	return result, nil
}

func validateRTSPSource(db *gorm.DB, entry *models.RTSPSource) error {
	entry.URL = strings.TrimSpace(entry.URL)
	entry.Remark = strings.TrimSpace(entry.Remark)
	parsed, err := url.Parse(entry.URL)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "rtsp" && parsed.Scheme != "rtsps") {
		return errors.New("請輸入包含主機的 rtsp:// 或 rtsps:// 地址")
	}
	if entry.BuildingID < 0 {
		return errors.New("invalid building_id")
	}
	if entry.BuildingID > 0 {
		var building models.Building
		if err := db.First(&building, entry.BuildingID).Error; err != nil {
			return fmt.Errorf("building not found: %w", err)
		}
	}
	return nil
}

func (s *RTSPSourceService) Create(ctx context.Context, entry models.RTSPSource) (*models.RTSPSource, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	entry.ModelFields = models.ModelFields{}
	entry.Building = nil
	db := s.db.WithContext(ctx)
	if err := validateRTSPSource(db, &entry); err != nil {
		return nil, err
	}
	if err := db.Create(&entry).Error; err != nil {
		return nil, err
	}
	// MySQL rounds timestamps to the column precision; return the persisted
	// value so clients can use it immediately for optimistic concurrency.
	return &entry, db.First(&entry, entry.ID).Error
}

type RTSPSourceUpdate struct {
	URL             string  `json:"url"`
	Remark          string  `json:"remark"`
	BuildingID      int64   `json:"building_id"`
	ExpectedVersion string  `json:"expected_version"`
	ConfirmedGroups []int64 `json:"confirmed_groups"`
}

type RTSPSourceUpdateResult struct {
	Source     models.RTSPSource `json:"source"`
	SyncErrors []string          `json:"sync_errors"`
}

func (s *RTSPSourceService) Update(ctx context.Context, id int64, payload RTSPSourceUpdate) (*RTSPSourceUpdateResult, error) {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	db := s.db.WithContext(ctx)
	var item models.RTSPSource
	var affected []models.NVR
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		if payload.ExpectedVersion != item.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00") {
			return ErrRTSPSourceConflict
		}
		oldURL := item.URL
		item.URL, item.Remark, item.BuildingID = payload.URL, payload.Remark, payload.BuildingID
		if err := validateRTSPSource(tx, &item); err != nil {
			return err
		}
		if item.URL != oldURL {
			var groups []models.NVR
			if err := tx.Order("id").Find(&groups).Error; err != nil {
				return err
			}
			confirmed := make(map[int64]bool)
			for _, groupID := range payload.ConfirmedGroups {
				confirmed[groupID] = true
			}
			for _, group := range groups {
				changed := false
				for i := range group.RTSPUrls {
					camera := &group.RTSPUrls[i]
					if camera.SourceID == id {
						if !confirmed[group.ID] {
							return ErrRTSPSourceConflict
						}
						if camera.SyncStatus == "success" && camera.SyncedOrangePiID > 0 && camera.SyncedSource == "" {
							camera.SyncedSource = camera.URL
						}
						camera.URL = item.URL
						camera.SyncStatus = "pending"
						camera.SyncError = ""
						changed = true
					}
				}
				if changed {
					if err := tx.Save(&group).Error; err != nil {
						return err
					}
					affected = append(affected, group)
				}
			}
		}
		return tx.Save(&item).Error
	})
	if err != nil {
		return nil, err
	}
	if err := db.First(&item, id).Error; err != nil {
		return nil, err
	}
	result := &RTSPSourceUpdateResult{Source: item, SyncErrors: make([]string, 0)}
	for i := range affected {
		group := &affected[i]
		if err := s.groups.syncGroup(ctx, group); err != nil {
			// The URL is already committed; report sync failures without hiding it.
			result.SyncErrors = append(result.SyncErrors, fmt.Sprintf("組 %s：%v", group.Name, err))
		}
	}
	return result, nil
}

func (s *RTSPSourceService) Delete(ctx context.Context, id int64) error {
	rtspWriteMu.Lock()
	defer rtspWriteMu.Unlock()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var entry models.RTSPSource
		if err := tx.First(&entry, id).Error; err != nil {
			return err
		}
		var groups []models.NVR
		if err := tx.Find(&groups).Error; err != nil {
			return err
		}
		for _, group := range groups {
			for _, camera := range group.RTSPUrls {
				if camera.SourceID == id {
					return ErrRTSPSourceInUse
				}
			}
		}
		return tx.Delete(&entry).Error
	})
}

// resolveLibrarySources keeps snapshots compatible with existing device/auth
// code. Legacy API writes create an entry rather than orphaning an address.
func resolveLibrarySources(tx *gorm.DB, values []models.ChannelURL) error {
	for i := range values {
		camera := &values[i]
		var entry models.RTSPSource
		if camera.SourceID > 0 {
			if err := tx.First(&entry, camera.SourceID).Error; err != nil {
				return fmt.Errorf("RTSP 清單地址 #%d 不存在，請重新整理: %w", camera.SourceID, err)
			}
		} else {
			entry = models.RTSPSource{URL: camera.URL, Remark: camera.Remark}
			if err := tx.Create(&entry).Error; err != nil {
				return err
			}
			camera.SourceID = entry.ID
		}
		camera.URL = entry.URL
	}
	return nil
}

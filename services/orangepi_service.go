package services

// OrangePiService Methods:
//0. NewOrangePiService(db *gorm.DB) -> 初始化设备服务
//1. List(ctx context.Context, ismartId string) -> 按ismartId筛选设备
//2. Create(ctx context.Context, payload models.OrangePi) -> 创建设备
//3. Update(ctx context.Context, id int64, payload models.OrangePi) -> 更新设备
//4. Delete(ctx context.Context, id int64) -> 删除设备

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"icctv-http-service/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrangePiServiceInterface 定义设备业务能力
type OrangePiServiceInterface interface {
	List(ctx context.Context, ismartId string) ([]models.OrangePi, error)                                    //1.查询设备
	Create(ctx context.Context, payload models.OrangePi) (*models.OrangePi, error)                           //2.创建设备
	Update(ctx context.Context, id int64, payload models.OrangePi) (*models.OrangePi, error)                 //3.更新设备
	Delete(ctx context.Context, id int64) error                                                              //4.删除设备
	RemoteUpdatePorts(ctx context.Context, id int64, sshPort int, authPort int) (*RemoteUpdateResult, error) //5.远程更新端口
	RemoteGetInfo(ctx context.Context, id int64, token string) (*RemoteDeviceInfo, error)                    //6.远程获取设备信息
	RemoteHealthCheck(ctx context.Context, id int64) (*RemoteHealthStatus, error)                            //7.远程健康检查
	// MediaMTX 远程管理
	RemoteListMediaMTXPaths(ctx context.Context, id int64, token string, page int, itemsPerPage int) (*MediaMTXPathsListResponse, error) //9.远程列出paths
	RemoteListMediaMTXPathsForBuilding(ctx context.Context, id int64, ismartID string, page, itemsPerPage int) (*MediaMTXPathsListResponse, error)
	SyncMediaMTXChannels(ctx context.Context, id int64, token string) (*models.OrangePi, error)
	RemoteGetMediaMTXPath(ctx context.Context, id int64, token string, name string) (*MediaMTXPathDetailResponse, error)                                   //10.远程查询单个path
	RemoteAddMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error)    //11.远程新增path
	RemoteUpdateMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error) //12.远程更新path
	RemoteDeleteMediaMTXPath(ctx context.Context, id int64, token string, name string) (*MediaMTXPathActionResponse, error)                                //13.远程删除path
	SyncRTSPGroup(ctx context.Context, group *models.NVR) error
}

// RemoteUpdateResult 远程更新结果
type RemoteUpdateResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Restarted bool   `json:"restarted"`
}

// RemoteDeviceInfo 远程设备信息
type RemoteDeviceInfo struct {
	DeviceID           string   `json:"device_id"`
	MediaMTXVersion    string   `json:"mediamtx_version"`
	FRPCServer         string   `json:"frpc_server"`
	FRPCAuthRemotePort int      `json:"frpc_auth_remote_port"`
	FRPCSSHRemotePort  int      `json:"frpc_ssh_remote_port"`
	FRPCAuthProxyName  string   `json:"frpc_auth_proxy_name"`
	FRPCSSHProxyName   string   `json:"frpc_ssh_proxy_name"`
	AvailableChannels  []string `json:"available_channels"`
	Status             string   `json:"status"`
}

// RemoteHealthStatus 远程健康状态
type RemoteHealthStatus struct {
	Status         string          `json:"status"`
	Service        string          `json:"service"`
	DockerServices map[string]bool `json:"docker_services"`
	MediaMTXStatus string          `json:"mediamtx_status"`
	FRPCStatus     string          `json:"frpc_status"`
	// 新增字段：设备活跃状态是否被更新
	IsActiveUpdated bool `json:"is_active_updated,omitempty"`
	NewIsActive     bool `json:"new_is_active,omitempty"`
}

// MediaMTXPathItem MediaMTX path 项
type MediaMTXPathItem struct {
	Name      string `json:"name"`
	Ready     *bool  `json:"ready,omitempty"`
	ConfName  string `json:"confName"`
	ReadyTime string `json:"readyTime"`
	Source    string `json:"source"` // RTSP URL
}

// MediaMTXPathsListResponse MediaMTX paths 列表响应
type MediaMTXPathsListResponse struct {
	Items      []MediaMTXPathItem `json:"items"`
	ItemsPage  int                `json:"itemsPage"`
	ItemsTotal int                `json:"itemsTotal"`
}

// MediaMTXPathDetailResponse MediaMTX path 详情响应
type MediaMTXPathDetailResponse struct {
	Name string                 `json:"name"`
	Conf map[string]interface{} `json:"conf"`
}

// MediaMTXPathActionResponse MediaMTX path 操作响应
type MediaMTXPathActionResponse struct {
	Action           string      `json:"action"`
	Name             string      `json:"name"`
	StatusCode       int         `json:"status_code"`
	MediaMTXResponse interface{} `json:"mediamtx_response"`
}

// TokenGeneratorFunc 定义生成 Token 的函数类型
// 参数: ismartID, isStaff
// 返回: token 字符串
type TokenGeneratorFunc func(ctx context.Context, ismartID string, orangePiID int64, isStaff bool) (string, error)

// OrangePiService 设备业务逻辑
type OrangePiService struct {
	db                 *gorm.DB
	publicNetService   *PublicNetService
	generateStaffToken TokenGeneratorFunc
}

type rtspGroupSyncContextKey struct{}

// 0. NewOrangePiService 构造函数
func NewOrangePiService(db *gorm.DB, publicNetService *PublicNetService) *OrangePiService {
	return &OrangePiService{
		db:               db,
		publicNetService: publicNetService,
	}
}

// SetTokenGenerator 设置 Token 生成器（用于避免循环依赖）
func (s *OrangePiService) SetTokenGenerator(fn TokenGeneratorFunc) {
	s.generateStaffToken = fn
}

// SyncRTSPGroup applies persisted RTSP sources to the bound device. Paths that
// were successfully managed by this group are deleted only after the group
// update has explicitly removed them and no other group references the path.
func (s *OrangePiService) SyncRTSPGroup(ctx context.Context, group *models.NVR) error {
	if group == nil || group.BuildingID <= 0 || group.OrangePiID <= 0 {
		return errors.New("rtsp group must be bound to a building and OrangePi")
	}
	var building models.Building
	if err := s.db.WithContext(ctx).First(&building, group.BuildingID).Error; err != nil {
		return err
	}
	if s.generateStaffToken == nil {
		return errors.New("OrangePi token generator is not configured")
	}
	token, err := s.generateStaffToken(ctx, building.ISmartID, group.OrangePiID, true)
	if err != nil {
		return err
	}
	const pageSize = 100
	first, err := s.RemoteListMediaMTXPaths(ctx, group.OrangePiID, token, 0, pageSize)
	if err != nil {
		return err
	}
	allPaths := append([]MediaMTXPathItem{}, first.Items...)
	totalPages := (first.ItemsTotal + pageSize - 1) / pageSize
	for page := 1; page < totalPages; page++ {
		result, pageErr := s.RemoteListMediaMTXPaths(ctx, group.OrangePiID, token, page, pageSize)
		if pageErr != nil {
			return pageErr
		}
		if result.ItemsTotal != first.ItemsTotal {
			return errors.New("device paths changed during synchronization")
		}
		allPaths = append(allPaths, result.Items...)
	}
	remote := make(map[string]MediaMTXPathItem, len(allPaths))
	for _, path := range allPaths {
		if _, duplicate := remote[path.Name]; duplicate {
			return errors.New("duplicate device paths during synchronization")
		}
		remote[path.Name] = path
	}
	if len(remote) != first.ItemsTotal {
		return errors.New("incomplete device paths during synchronization")
	}
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, group.OrangePiID).Error; err != nil {
		return err
	}
	reserved := reservedRTSPGroupPaths(remote, device)
	syncCtx := context.WithValue(ctx, rtspGroupSyncContextKey{}, true)
	now := time.Now()
	var firstErr error
	remainingDeletes := make([]models.ChannelURL, 0, len(group.PendingRTSPDeletes))
	for _, pending := range group.PendingRTSPDeletes {
		path := strings.TrimSpace(pending.Path)
		if path == "" || pending.SyncStatus != "success" || pending.SyncedOrangePiID != group.OrangePiID {
			remainingDeletes = append(remainingDeletes, pending)
			if firstErr == nil {
				firstErr = fmt.Errorf("cannot verify ownership of pending OrangePi Path %q", path)
			}
			continue
		}
		// Re-adding a camera cancels its pending deletion.
		readded := false
		for _, camera := range group.RTSPUrls {
			if camera.Path == path {
				readded = true
				break
			}
		}
		if readded {
			continue
		}
		referenced, referenceErr := s.rtspPathReferencedByOtherGroup(ctx, group.ID, group.OrangePiID, path)
		if referenceErr != nil {
			remainingDeletes = append(remainingDeletes, pending)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to verify OrangePi Path %q references: %w", path, referenceErr)
			}
			continue
		}
		if referenced {
			remainingDeletes = append(remainingDeletes, pending)
			if firstErr == nil {
				firstErr = fmt.Errorf("cannot delete OrangePi Path %q: another RTSP group references it", path)
			}
			continue
		}
		if _, exists := remote[path]; !exists {
			continue
		}
		detail, detailErr := s.RemoteGetMediaMTXPath(ctx, group.OrangePiID, token, path)
		if detailErr != nil || pending.URL == "" || detail.Conf["source"] != pending.URL {
			remainingDeletes = append(remainingDeletes, pending)
			if firstErr == nil {
				firstErr = fmt.Errorf("cannot delete OrangePi Path %q: source changed or configuration could not be verified", path)
			}
			continue
		}
		if _, opErr := s.RemoteDeleteMediaMTXPath(syncCtx, group.OrangePiID, token, path); opErr != nil {
			remainingDeletes = append(remainingDeletes, pending)
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to delete OrangePi Path %q: %w", path, opErr)
			}
			continue
		}
		delete(remote, path)
		delete(reserved, path)
	}
	group.PendingRTSPDeletes = remainingDeletes
	for i := range group.RTSPUrls {
		item := &group.RTSPUrls[i]
		wasSynced := item.SyncedOrangePiID == group.OrangePiID &&
			(item.SyncStatus == "success" || item.SyncedSource != "")
		item.SyncStatus = "syncing"
		item.SyncError = ""
		config := map[string]interface{}{"source": item.URL}
		var opErr error
		existing, exists := remote[item.Path]
		switch {
		case wasSynced && !exists:
			// A previously managed path disappearing is an external change. Do not
			// recreate it automatically, because the path may have been removed
			// intentionally or reassigned on the device.
			opErr = fmt.Errorf("managed path %q is missing on OrangePi", item.Path)
		case wasSynced && exists:
			if item.SyncedSource != "" && existing.Source != item.SyncedSource && existing.Source != item.URL {
				detail, detailErr := s.RemoteGetMediaMTXPath(ctx, group.OrangePiID, token, item.Path)
				if detailErr != nil || (detail.Conf["source"] != item.SyncedSource && detail.Conf["source"] != item.URL) {
					opErr = fmt.Errorf("managed path %q source changed on OrangePi", item.Path)
					break
				}
			}
			if existing.Source != "" && existing.Source == item.URL {
				item.SyncStatus = "success"
				item.SyncedAt = &now
				item.SyncedOrangePiID = group.OrangePiID
				item.SyncedSource = item.URL
				continue
			}
			referenced, referenceErr := s.rtspPathReferencedByOtherGroup(ctx, group.ID, group.OrangePiID, item.Path)
			if referenceErr != nil || referenced {
				opErr = fmt.Errorf("cannot update path %q: ownership is shared or could not be verified", item.Path)
				break
			}
			_, opErr = s.RemoteUpdateMediaMTXPath(syncCtx, group.OrangePiID, token, item.Path, config)
		case exists || reserved[item.Path]:
			// Never silently move an existing database binding to another
			// channel. The administrator must resolve the conflict explicitly.
			opErr = fmt.Errorf("path %q is already reserved on OrangePi", item.Path)
		default:
			_, opErr = s.RemoteAddMediaMTXPath(syncCtx, group.OrangePiID, token, item.Path, config)
		}
		if opErr != nil {
			item.SyncStatus = "error"
			item.SyncError = opErr.Error()
			if firstErr == nil {
				firstErr = opErr
			}
			continue
		}
		item.SyncStatus = "success"
		item.SyncedAt = &now
		item.SyncedOrangePiID = group.OrangePiID
		item.SyncedSource = item.URL
		reserved[item.Path] = true
		remote[item.Path] = MediaMTXPathItem{Name: item.Path, Source: item.URL}
	}
	return firstErr
}

func (s *OrangePiService) rtspPathReferencedByOtherGroup(ctx context.Context, groupID, orangePiID int64, path string) (bool, error) {
	var groups []models.NVR
	if err := s.db.WithContext(ctx).
		Where("id <> ?", groupID).
		Find(&groups).Error; err != nil {
		return false, err
	}
	for _, group := range groups {
		for _, item := range group.RTSPUrls {
			if (group.OrangePiID == orangePiID || item.SyncedOrangePiID == orangePiID) &&
				strings.EqualFold(strings.TrimSpace(item.Path), path) {
				return true, nil
			}
		}
		for _, item := range group.PendingRTSPDeletes {
			if item.SyncedOrangePiID == orangePiID && strings.EqualFold(strings.TrimSpace(item.Path), path) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (s *OrangePiService) RemoteListMediaMTXPathsForBuilding(ctx context.Context, id int64, ismartID string, page, itemsPerPage int) (*MediaMTXPathsListResponse, error) {
	if s.generateStaffToken == nil {
		return nil, errors.New("OrangePi token generator is not configured")
	}
	token, err := s.generateStaffToken(ctx, ismartID, id, true)
	if err != nil {
		return nil, err
	}
	return s.RemoteListMediaMTXPaths(ctx, id, token, page, itemsPerPage)
}

// reservedRTSPGroupPaths returns names that can actually conflict with a
// persisted RTSP group. The remote MediaMTX catalogue is authoritative here;
// legacy permission arrays and stale channel catalogue entries are not device
// Paths and must not force allocations such as channel23 on a device that only
// contains channel1 and channel2.
func reservedRTSPGroupPaths(remote map[string]MediaMTXPathItem, device models.OrangePi) map[string]bool {
	reserved := make(map[string]bool, len(remote)+len(device.ChannelPaths))
	for name := range remote {
		reserved[name] = true
	}
	for key, path := range device.ChannelPaths {
		if _, exists := remote[path]; exists {
			reserved[key] = true
			reserved[path] = true
		}
		if _, exists := remote[key]; exists {
			reserved[key] = true
			reserved[path] = true
		}
	}
	return reserved
}

func nextAvailableChannelPath(reserved map[string]bool) (string, int) {
	for next := 1; ; next++ {
		candidate := fmt.Sprintf("channel%d", next)
		if !reserved[candidate] {
			return candidate, next
		}
	}
}

// 1. List 查询设备列表
func (s *OrangePiService) List(ctx context.Context, ismartId string) ([]models.OrangePi, error) {
	var devices []models.OrangePi
	tx := s.db.WithContext(ctx).Model(&models.OrangePi{})
	if ismartId != "" {
		tx = tx.
			Joins("JOIN orangepi_buildings ON orangepi_buildings.orange_pi_id = orangepis.id").
			Joins("JOIN buildings ON buildings.id = orangepi_buildings.building_id AND buildings.deleted_at IS NULL").
			Where("buildings.ismart_id = ?", strings.TrimSpace(ismartId)).
			Distinct("orangepis.*")
	}
	if err := tx.Preload("Buildings", func(db *gorm.DB) *gorm.DB {
		return db.Order("buildings.id ASC")
	}).Find(&devices).Error; err != nil {
		return nil, err
	}
	hydrateOrangePis(devices)
	return devices, nil
}

// 2. Create 创建设备
func (s *OrangePiService) Create(ctx context.Context, payload models.OrangePi) (*models.OrangePi, error) {
	// 确保 UserChannels 初始化为空数组而不是 nil
	if payload.UserChannels == nil {
		payload.UserChannels = []int{}
	}
	// 确保 AllChannels 初始化为空数组而不是 nil
	if payload.AllChannels == nil {
		payload.AllChannels = []int{}
	}
	payload.ChannelRemarks = normalizeChannelRemarks(payload.ChannelRemarks)

	ismartIDs := payload.ISmartIDs
	if ismartIDs == nil && payload.ISmartID != "" {
		ismartIDs = []string{payload.ISmartID}
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		buildings, err := loadBuildingsByISmartIDs(tx, ismartIDs)
		if err != nil {
			return err
		}
		payload.ISmartID = buildings[0].ISmartID
		payload.Buildings = nil
		if err := tx.Create(&payload).Error; err != nil {
			return err
		}
		return syncOrangePiBuildings(tx, payload.ID, buildings, defaultBuildingChannels(payload))
	})
	if err != nil {
		return nil, err
	}
	payload.ISmartIDs = normalizeISmartIDs(ismartIDs)
	return &payload, nil
}

// 3. Update 更新设备
func (s *OrangePiService) Update(ctx context.Context, id int64, payload models.OrangePi) (*models.OrangePi, error) {
	var device models.OrangePi
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&device, id).Error; err != nil {
			return err
		}

		if payload.Name != "" {
			device.Name = payload.Name
		}
		if payload.ICCTVAuthServiceRemotePort != 0 {
			device.ICCTVAuthServiceRemotePort = payload.ICCTVAuthServiceRemotePort
		}
		if payload.SSHRemotePort != 0 {
			device.SSHRemotePort = payload.SSHRemotePort
		}
		if payload.IsActiveSet {
			device.IsActive = payload.IsActive
		}
		if payload.UserChannels != nil {
			device.UserChannels = payload.UserChannels
		}
		if payload.AllChannels != nil {
			device.AllChannels = payload.AllChannels
		}
		if payload.ChannelRemarks != nil {
			device.ChannelRemarks = normalizeChannelRemarks(payload.ChannelRemarks)
		}

		if payload.ISmartIDs != nil || payload.ISmartID != "" {
			ismartIDs := payload.ISmartIDs
			if ismartIDs == nil {
				ismartIDs = []string{payload.ISmartID}
			}
			buildings, err := loadBuildingsByISmartIDs(tx, ismartIDs)
			if err != nil {
				return err
			}
			device.ISmartID = buildings[0].ISmartID
			if err := syncOrangePiBuildings(tx, device.ID, buildings, defaultBuildingChannels(device)); err != nil {
				return err
			}
		}

		return tx.Save(&device).Error
	})
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Preload("Buildings").First(&device, id).Error; err != nil {
		return nil, err
	}
	hydrateOrangePi(&device)
	return &device, nil
}

// 4. Delete 删除设备
func (s *OrangePiService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("orange_pi_id = ?", id).Delete(&models.OrangePiBuilding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.OrangePi{}, id).Error
	})
}

func defaultBuildingChannels(device models.OrangePi) []int {
	if len(device.AllChannels) > 0 {
		return normalizeChannels(device.AllChannels)
	}
	return normalizeChannels(device.UserChannels)
}

func normalizeChannels(values []int) []int {
	result := make([]int, 0, len(values))
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *OrangePiService) registerMediaMTXChannel(ctx context.Context, orangePiID int64, pathName string) error {
	return s.registerMediaMTXPaths(ctx, orangePiID, []MediaMTXPathItem{{Name: pathName}})
}

func normalizeChannelRemarks(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		result[key] = strings.TrimSpace(value)
	}
	return result
}

func restrictChannels(base, allowed []int) []int {
	base = normalizeChannels(base)
	allowed = normalizeChannels(allowed)
	if len(allowed) == 0 {
		return base
	}
	allowedSet := make(map[int]struct{}, len(allowed))
	for _, channel := range allowed {
		allowedSet[channel] = struct{}{}
	}
	result := make([]int, 0, len(base))
	for _, channel := range base {
		if _, ok := allowedSet[channel]; ok {
			result = append(result, channel)
		}
	}
	return result
}

func syncOrangePiBuildings(tx *gorm.DB, orangePiID int64, buildings []models.Building, defaultChannels []int) error {
	var existing []models.OrangePiBuilding
	if err := tx.Where("orange_pi_id = ?", orangePiID).Find(&existing).Error; err != nil {
		return err
	}
	existingByBuilding := make(map[int64]models.OrangePiBuilding, len(existing))
	wanted := make(map[int64]struct{}, len(buildings))
	for _, link := range existing {
		existingByBuilding[link.BuildingID] = link
	}
	for _, building := range buildings {
		wanted[building.ID] = struct{}{}
		if _, ok := existingByBuilding[building.ID]; ok {
			continue
		}
		link := models.OrangePiBuilding{
			OrangePiID:      orangePiID,
			BuildingID:      building.ID,
			AllowedChannels: append([]int(nil), defaultChannels...),
		}
		if err := tx.Create(&link).Error; err != nil {
			return err
		}
	}
	for _, link := range existing {
		if _, ok := wanted[link.BuildingID]; !ok {
			if err := tx.Where("orange_pi_id = ? AND building_id = ?", orangePiID, link.BuildingID).Delete(&models.OrangePiBuilding{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeISmartIDs(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func loadBuildingsByISmartIDs(tx *gorm.DB, values []string) ([]models.Building, error) {
	ismartIDs := normalizeISmartIDs(values)
	if len(ismartIDs) == 0 {
		return nil, errors.New("at least one ismartid is required")
	}

	var found []models.Building
	if err := tx.Where("ismart_id IN ?", ismartIDs).Find(&found).Error; err != nil {
		return nil, err
	}
	byISmartID := make(map[string]models.Building, len(found))
	for _, building := range found {
		byISmartID[building.ISmartID] = building
	}

	buildings := make([]models.Building, 0, len(ismartIDs))
	for _, ismartID := range ismartIDs {
		building, exists := byISmartID[ismartID]
		if !exists {
			return nil, fmt.Errorf("building not found: %s", ismartID)
		}
		buildings = append(buildings, building)
	}
	return buildings, nil
}

func hydrateOrangePis(devices []models.OrangePi) {
	for i := range devices {
		hydrateOrangePi(&devices[i])
	}
}

func hydrateOrangePi(device *models.OrangePi) {
	device.ISmartIDs = make([]string, 0, len(device.Buildings))
	for _, building := range device.Buildings {
		device.ISmartIDs = append(device.ISmartIDs, building.ISmartID)
	}
	if len(device.Buildings) > 0 {
		device.ISmartID = device.Buildings[0].ISmartID
		device.Building = &device.Buildings[0]
	}
}

func (s *OrangePiService) primaryISmartID(ctx context.Context, device *models.OrangePi) (string, error) {
	if err := s.db.WithContext(ctx).Preload("Buildings", func(db *gorm.DB) *gorm.DB {
		return db.Order("buildings.id ASC")
	}).First(device, device.ID).Error; err != nil {
		return "", err
	}
	hydrateOrangePi(device)
	if len(device.ISmartIDs) == 0 {
		return "", errors.New("orangepi is not bound to any building")
	}
	return device.ISmartIDs[0], nil
}

// 5. RemoteUpdatePorts 远程更新端口
func (s *OrangePiService) RemoteUpdatePorts(ctx context.Context, id int64, sshPort int, authPort int) (*RemoteUpdateResult, error) {
	// 0. 校验端口是否相同
	if sshPort == authPort {
		return nil, errors.New("ssh_remote_port and icctv_auth_service_remote_port cannot be the same")
	}

	// 1. 校验端口是否已被其他设备占用
	var count int64
	err := s.db.WithContext(ctx).Model(&models.OrangePi{}).
		Where("id <> ?", id). // 排除当前设备
		Where("(ssh_remote_port = ? OR icctv_auth_service_remote_port = ? OR ssh_remote_port = ? OR icctv_auth_service_remote_port = ?)",
			sshPort, sshPort, authPort, authPort).
		Count(&count).Error

	if err != nil {
		return nil, fmt.Errorf("failed to check port availability: %w", err)
	}

	if count > 0 {
		return nil, errors.New("one or both ports are already in use by another device")
	}

	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 生成 staff token（需要 is_staff=true 权限才能修改端口）
	if s.generateStaffToken == nil {
		return nil, errors.New("token generator not configured")
	}
	ismartID, err := s.primaryISmartID(ctx, &device)
	if err != nil {
		return nil, err
	}
	token, err := s.generateStaffToken(ctx, ismartID, device.ID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to generate staff token: %w", err)
	}

	// 构建远程URL（携带 token）
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/frpc/ports?token=%s",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, token)

	// 构建请求体
	requestBody := map[string]int{
		"orangepi_ssh_remote_port":        sshPort,
		"icctv_orangepi_auth_remote_port": authPort,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// 发送HTTP请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(remoteURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	var result RemoteUpdateResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 如果远程更新成功，更新本地数据库
	if result.Success {
		device.SSHRemotePort = sshPort
		device.ICCTVAuthServiceRemotePort = authPort
		if err := s.db.WithContext(ctx).Model(&device).
			Select("SSHRemotePort", "ICCTVAuthServiceRemotePort").Updates(&device).Error; err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// 6. RemoteGetInfo 远程获取设备信息
func (s *OrangePiService) RemoteGetInfo(ctx context.Context, id int64, token string) (*RemoteDeviceInfo, error) {
	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 构建远程URL (需要携带token)
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/info?token=%s",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, token)

	// 发送HTTP请求
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	var deviceInfo RemoteDeviceInfo
	if err := json.NewDecoder(resp.Body).Decode(&deviceInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &deviceInfo, nil
}

// 7. RemoteHealthCheck 远程健康检查
// 如果健康检查失败，会自动更新数据库中的 is_active 状态
func (s *OrangePiService) RemoteHealthCheck(ctx context.Context, id int64) (*RemoteHealthStatus, error) {
	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 构建远程URL
	remoteURL := fmt.Sprintf("http://%s:%d/health",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort)

	// 发送HTTP请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(remoteURL)

	// 健康检查失败
	if err != nil {
		// 如果设备之前是活跃的，更新为不活跃
		if device.IsActive {
			device.IsActive = false
			if updateErr := s.db.WithContext(ctx).Model(&device).Select("IsActive").Updates(&device).Error; updateErr != nil {
				return nil, fmt.Errorf("failed to connect to remote device and failed to update status: connect=%w, update=%v", err, updateErr)
			}
			return &RemoteHealthStatus{
				Status:          "unhealthy",
				Service:         "unknown",
				IsActiveUpdated: true,
				NewIsActive:     false,
			}, nil
		}
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码，非200也视为不健康
	if resp.StatusCode != http.StatusOK {
		// 如果设备之前是活跃的，更新为不活跃
		if device.IsActive {
			device.IsActive = false
			if updateErr := s.db.WithContext(ctx).Model(&device).Select("IsActive").Updates(&device).Error; updateErr != nil {
				return nil, fmt.Errorf("health check returned non-200 and failed to update status: %w", updateErr)
			}
			return &RemoteHealthStatus{
				Status:          "unhealthy",
				Service:         "unknown",
				IsActiveUpdated: true,
				NewIsActive:     false,
			}, nil
		}
		return &RemoteHealthStatus{
			Status:  "unhealthy",
			Service: "unknown",
		}, nil
	}

	var healthStatus RemoteHealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&healthStatus); err != nil {
		// 解码失败也视为不健康
		healthStatus = RemoteHealthStatus{
			Status:  "unhealthy",
			Service: "unknown",
		}
	}

	// 根据健康状态判断是否需要更新数据库
	// healthy/degraded 视为活跃，unhealthy 视为不活跃
	newIsActive := healthStatus.Status == "healthy" || healthStatus.Status == "degraded"

	// 只有状态变化时才更新数据库
	if device.IsActive != newIsActive {
		device.IsActive = newIsActive
		if updateErr := s.db.WithContext(ctx).Model(&device).Select("IsActive").Updates(&device).Error; updateErr != nil {
			return nil, fmt.Errorf("health check succeeded but failed to update status: %w", updateErr)
		}
		healthStatus.IsActiveUpdated = true
		healthStatus.NewIsActive = newIsActive
	}

	return &healthStatus, nil
}

// 9. RemoteListMediaMTXPaths 远程列出 MediaMTX paths
func (s *OrangePiService) RemoteListMediaMTXPaths(ctx context.Context, id int64, token string, page int, itemsPerPage int) (*MediaMTXPathsListResponse, error) {
	result, err := s.fetchMediaMTXPaths(ctx, id, token, page, itemsPerPage)
	if err != nil {
		return nil, err
	}
	if err := s.registerMediaMTXPaths(ctx, id, result.Items); err != nil {
		return nil, fmt.Errorf("failed to register MediaMTX channels: %w", err)
	}
	return result, nil
}

func (s *OrangePiService) fetchMediaMTXPaths(ctx context.Context, id int64, token string, page int, itemsPerPage int) (*MediaMTXPathsListResponse, error) {
	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 构建远程URL
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/mediamtx/paths?token=%s&page=%d&items_per_page=%d",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, token, page, itemsPerPage)

	// 发送HTTP请求
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote device returned status: %d", resp.StatusCode)
	}

	return decodeMediaMTXPaths(resp.Body, page)
}

// 10. RemoteGetMediaMTXPath 远程查询单个 MediaMTX path
func (s *OrangePiService) RemoteGetMediaMTXPath(ctx context.Context, id int64, token string, name string) (*MediaMTXPathDetailResponse, error) {
	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 构建远程URL
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/mediamtx/paths/%s?token=%s",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, url.PathEscape(name), url.QueryEscape(token))

	// 发送HTTP请求
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote device returned status: %d", resp.StatusCode)
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if raw == nil {
		return nil, errors.New("remote device returned empty path configuration")
	}
	conf, nested := raw["conf"].(map[string]interface{})
	if !nested {
		delete(raw, "name")
		conf = raw
	}
	return &MediaMTXPathDetailResponse{Name: name, Conf: conf}, nil
}

// 11. RemoteAddMediaMTXPath 远程新增 MediaMTX path
func (s *OrangePiService) RemoteAddMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error) {
	if err := validateChannelPathConfig(name, config); err != nil {
		return nil, err
	}
	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}
	if _, syncingRTSPGroup := ctx.Value(rtspGroupSyncContextKey{}).(bool); !syncingRTSPGroup {
		// Keep the legacy manual-add guard. RTSP group synchronization has
		// already checked the complete live remote catalogue and must not be
		// blocked by stale permission/catalogue IDs.
		if _, reserved := device.ChannelPaths[name]; reserved {
			return nil, ErrReservedChannelPath
		}
		for _, path := range device.ChannelPaths {
			if path == name {
				return nil, ErrReservedChannelPath
			}
		}
		for _, channel := range append(append([]int{}, device.AllChannels...), device.UserChannels...) {
			if channel == numericChannel(name) {
				return nil, ErrReservedChannelPath
			}
		}
		var links []models.OrangePiBuilding
		if err := s.db.WithContext(ctx).Where("orange_pi_id = ?", id).Find(&links).Error; err != nil {
			return nil, err
		}
		for _, link := range links {
			for _, channel := range link.AllowedChannels {
				if channel == numericChannel(name) {
					return nil, ErrReservedChannelPath
				}
			}
		}
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 构建远程URL
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/mediamtx/paths?token=%s",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, url.QueryEscape(token))

	// 强制设置固定配置
	config["sourceOnDemand"] = false // 持续拉流
	config["record"] = false         // 不录像

	// 构建请求体
	requestBody := map[string]interface{}{
		"name":   name,
		"config": config,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// 发送HTTP请求
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, remoteURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("remote device returned status: %d", resp.StatusCode)
	}
	var result MediaMTXPathActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("remote device failed to add MediaMTX path: status %d", result.StatusCode)
	}
	if err := s.registerMediaMTXChannel(ctx, device.ID, name); err != nil {
		return nil, fmt.Errorf("MediaMTX path was created but its channel could not be registered: %w", err)
	}

	return &result, nil
}

// 12. RemoteUpdateMediaMTXPath 远程更新 MediaMTX path
func (s *OrangePiService) RemoteUpdateMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error) {
	if err := validateChannelPathConfig(name, config); err != nil {
		return nil, err
	}
	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 构建远程URL
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/mediamtx/paths/%s?token=%s",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, name, url.QueryEscape(token))

	// 强制设置固定配置
	config["sourceOnDemand"] = false // 持续拉流
	config["record"] = false         // 不录像

	// 构建请求体
	requestBody := map[string]interface{}{
		"config": config,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// 创建 PATCH 请求
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "PATCH", remoteURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("remote device returned status: %d", resp.StatusCode)
	}
	var result MediaMTXPathActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("remote device failed to update MediaMTX path: status %d", result.StatusCode)
	}

	return &result, nil
}

// 13. RemoteDeleteMediaMTXPath 远程删除 MediaMTX path
func (s *OrangePiService) RemoteDeleteMediaMTXPath(ctx context.Context, id int64, token string, name string) (*MediaMTXPathActionResponse, error) {
	// 获取设备信息
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 获取公网配置
	publicNetConfig, err := s.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		return nil, errors.New("public network configuration not found")
	}

	// 构建远程URL
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/mediamtx/paths/%s?token=%s",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, url.PathEscape(name), url.QueryEscape(token))

	// 创建 DELETE 请求
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "DELETE", remoteURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("remote device path %q was not found", name)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("remote device returned status: %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNoContent {
		return &MediaMTXPathActionResponse{
			Action:     "delete",
			Name:       name,
			StatusCode: resp.StatusCode,
		}, nil
	}

	var result MediaMTXPathActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("remote device failed to delete MediaMTX path: status %d", result.StatusCode)
	}

	return &result, nil
}

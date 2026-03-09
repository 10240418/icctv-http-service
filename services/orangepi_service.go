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
	"time"

	"icctv-http-service/models"

	"gorm.io/gorm"
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
	RemoteListMediaMTXPaths(ctx context.Context, id int64, token string, page int, itemsPerPage int) (*MediaMTXPathsListResponse, error)                   //9.远程列出paths
	RemoteGetMediaMTXPath(ctx context.Context, id int64, token string, name string) (*MediaMTXPathDetailResponse, error)                                   //10.远程查询单个path
	RemoteAddMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error)    //11.远程新增path
	RemoteUpdateMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error) //12.远程更新path
	RemoteDeleteMediaMTXPath(ctx context.Context, id int64, token string, name string) (*MediaMTXPathActionResponse, error)                                //13.远程删除path
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
	Ready     bool   `json:"ready"`
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
type TokenGeneratorFunc func(ctx context.Context, ismartID string, isStaff bool) (string, error)

// OrangePiService 设备业务逻辑
type OrangePiService struct {
	db                 *gorm.DB
	publicNetService   *PublicNetService
	generateStaffToken TokenGeneratorFunc
}

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

// 1. List 查询设备列表
func (s *OrangePiService) List(ctx context.Context, ismartId string) ([]models.OrangePi, error) {
	var devices []models.OrangePi
	tx := s.db.WithContext(ctx).Model(&models.OrangePi{})
	if ismartId != "" {
		tx = tx.Where("ismart_id = ?", ismartId)
	}
	if err := tx.Preload("Building").Find(&devices).Error; err != nil {
		return nil, err
	}
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

	if err := s.db.WithContext(ctx).Create(&payload).Error; err != nil {
		return nil, err
	}
	return &payload, nil
}

// 3. Update 更新设备
func (s *OrangePiService) Update(ctx context.Context, id int64, payload models.OrangePi) (*models.OrangePi, error) {
	var device models.OrangePi
	if err := s.db.WithContext(ctx).First(&device, id).Error; err != nil {
		return nil, err
	}

	// 更新字段
	if payload.ISmartID != "" {
		device.ISmartID = payload.ISmartID
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
	device.IsActive = payload.IsActive
	// 更新 UserChannels（如果提供了）
	if payload.UserChannels != nil {
		device.UserChannels = payload.UserChannels
	}
	// 更新 AllChannels（如果提供了）
	if payload.AllChannels != nil {
		device.AllChannels = payload.AllChannels
	}

	if err := s.db.WithContext(ctx).Save(&device).Error; err != nil {
		return nil, err
	}
	return &device, nil
}

// 4. Delete 删除设备
func (s *OrangePiService) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&models.OrangePi{}, id).Error
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
	token, err := s.generateStaffToken(ctx, device.ISmartID, true)
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
		if err := s.db.WithContext(ctx).Save(&device).Error; err != nil {
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
			if updateErr := s.db.WithContext(ctx).Save(&device).Error; updateErr != nil {
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
			if updateErr := s.db.WithContext(ctx).Save(&device).Error; updateErr != nil {
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
		if updateErr := s.db.WithContext(ctx).Save(&device).Error; updateErr != nil {
			return nil, fmt.Errorf("health check succeeded but failed to update status: %w", updateErr)
		}
		healthStatus.IsActiveUpdated = true
		healthStatus.NewIsActive = newIsActive
	}

	return &healthStatus, nil
}

// 9. RemoteListMediaMTXPaths 远程列出 MediaMTX paths
func (s *OrangePiService) RemoteListMediaMTXPaths(ctx context.Context, id int64, token string, page int, itemsPerPage int) (*MediaMTXPathsListResponse, error) {
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
	resp, err := client.Get(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote device returned status: %d", resp.StatusCode)
	}

	var result MediaMTXPathsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
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
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, name, token)

	// 发送HTTP请求
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote device returned status: %d", resp.StatusCode)
	}

	var result MediaMTXPathDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// 11. RemoteAddMediaMTXPath 远程新增 MediaMTX path
func (s *OrangePiService) RemoteAddMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error) {
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
	remoteURL := fmt.Sprintf("http://%s:%d/api/device/mediamtx/paths?token=%s",
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, token)

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
	resp, err := client.Post(remoteURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote device: %w", err)
	}
	defer resp.Body.Close()

	var result MediaMTXPathActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// 12. RemoteUpdateMediaMTXPath 远程更新 MediaMTX path
func (s *OrangePiService) RemoteUpdateMediaMTXPath(ctx context.Context, id int64, token string, name string, config map[string]interface{}) (*MediaMTXPathActionResponse, error) {
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
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, name, token)

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

	var result MediaMTXPathActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
		publicNetConfig.ExternalIP, device.ICCTVAuthServiceRemotePort, name, token)

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

	var result MediaMTXPathActionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

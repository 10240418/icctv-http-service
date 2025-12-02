package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"icctv-http-service/models"

	"gorm.io/gorm"
)

// HealthChecker 后台健康检查服务
type HealthChecker struct {
	db               *gorm.DB
	publicNetService *PublicNetService
	interval         time.Duration
	stopChan         chan struct{}
	wg               sync.WaitGroup
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(db *gorm.DB, publicNetService *PublicNetService, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		db:               db,
		publicNetService: publicNetService,
		interval:         interval,
		stopChan:         make(chan struct{}),
	}
}

// Start 启动后台健康检查
func (h *HealthChecker) Start() {
	h.wg.Add(1)
	go h.run()
	log.Printf("✓ Health checker started, interval: %v", h.interval)
}

// Stop 停止健康检查
func (h *HealthChecker) Stop() {
	close(h.stopChan)
	h.wg.Wait()
	log.Println("✓ Health checker stopped")
}

// run 运行健康检查循环
func (h *HealthChecker) run() {
	defer h.wg.Done()

	// 启动时先执行一次检查
	h.checkAllDevices()

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkAllDevices()
		case <-h.stopChan:
			return
		}
	}
}

// checkAllDevices 检查所有 OrangePi 设备的健康状态
func (h *HealthChecker) checkAllDevices() {
	ctx := context.Background()

	// 获取所有设备
	var devices []models.OrangePi
	if err := h.db.WithContext(ctx).Find(&devices).Error; err != nil {
		log.Printf("[HealthChecker] Failed to fetch devices: %v", err)
		return
	}

	if len(devices) == 0 {
		log.Println("[HealthChecker] No devices to check")
		return
	}

	log.Printf("[HealthChecker] Checking %d devices...", len(devices))

	// 获取公网配置
	publicNetConfig, err := h.publicNetService.Get(ctx)
	if err != nil || publicNetConfig == nil {
		log.Printf("[HealthChecker] Failed to get public network config: %v", err)
		return
	}

	// 并发检查所有设备
	var wg sync.WaitGroup
	results := make(chan healthCheckResult, len(devices))

	for _, device := range devices {
		wg.Add(1)
		go func(d models.OrangePi) {
			defer wg.Done()
			isHealthy := h.checkDevice(ctx, d, publicNetConfig.ExternalIP)
			results <- healthCheckResult{
				deviceID:   d.ID,
				deviceName: d.Name,
				isHealthy:  isHealthy,
				wasActive:  d.IsActive,
			}
		}(device)
	}

	// 等待所有检查完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 处理结果并更新数据库
	healthyCount := 0
	unhealthyCount := 0
	changedCount := 0

	for result := range results {
		if result.isHealthy {
			healthyCount++
		} else {
			unhealthyCount++
		}

		// 只有状态变化时才更新数据库
		if result.isHealthy != result.wasActive {
			changedCount++
			if err := h.db.WithContext(ctx).Model(&models.OrangePi{}).
				Where("id = ?", result.deviceID).
				Update("is_active", result.isHealthy).Error; err != nil {
				log.Printf("[HealthChecker] Failed to update device %s (ID: %d): %v",
					result.deviceName, result.deviceID, err)
			} else {
				status := "active"
				if !result.isHealthy {
					status = "inactive"
				}
				log.Printf("[HealthChecker] Device %s (ID: %d) status changed to %s",
					result.deviceName, result.deviceID, status)
			}
		}
	}

	log.Printf("[HealthChecker] Check completed: %d healthy, %d unhealthy, %d changed",
		healthyCount, unhealthyCount, changedCount)
}

// healthCheckResult 健康检查结果
type healthCheckResult struct {
	deviceID   int64
	deviceName string
	isHealthy  bool
	wasActive  bool
}

// checkDevice 检查单个设备的健康状态
func (h *HealthChecker) checkDevice(ctx context.Context, device models.OrangePi, externalIP string) bool {
	// 构建健康检查 URL
	url := fmt.Sprintf("http://%s:%d/health", externalIP, device.ICCTVAuthServiceRemotePort)

	// 创建带超时的 HTTP 客户端
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// 只要能正常响应且状态码为 200，就认为是健康的
	return resp.StatusCode == http.StatusOK
}

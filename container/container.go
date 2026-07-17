package container

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"icctv-http-service/controllers"
	"icctv-http-service/databases"
	"icctv-http-service/middlewares"
	"icctv-http-service/routes"
	"icctv-http-service/services"

	"gorm.io/gorm"
)

// ServiceSet 聚合所有业务服务
type ServiceSet struct {
	Admin     *services.AdminService
	Auth      *services.AuthService
	OrangePi  *services.OrangePiService
	Building  *services.BuildingService
	Device    *services.DeviceService
	PublicNet *services.PublicNetService
	NVR       *services.NVRService
}

// Container 提供项目运行所需的依赖
type Container struct {
	DB            *gorm.DB
	Services      ServiceSet
	Controllers   routes.ControllerSet
	Middlewares   routes.MiddlewareSet
	HealthChecker *services.HealthChecker
}

// Build 初始化所有依赖
func Build() (*Container, error) {
	db, err := databases.Init()
	if err != nil {
		return nil, err
	}

	serviceSet := ServiceSet{
		Admin:     services.NewAdminService(db),
		Building:  services.NewBuildingService(db),
		Device:    services.NewDeviceService(db),
		PublicNet: services.NewPublicNetService(db),
		NVR:       services.NewNVRService(db),
	}
	serviceSet.OrangePi = services.NewOrangePiService(db, serviceSet.PublicNet)
	serviceSet.Auth = services.NewAuthService(db, serviceSet.Admin, serviceSet.OrangePi, serviceSet.Building)

	// 设置 OrangePi 服务的 Token 生成器（解决循环依赖）
	serviceSet.OrangePi.SetTokenGenerator(func(ctx context.Context, ismartID string, orangePiID int64, isStaff bool) (string, error) {
		resp, err := serviceSet.Auth.GeneratePublicToken(ctx, ismartID, isStaff)
		if err != nil {
			return "", err
		}
		for _, orangePi := range resp.OrangePis {
			if orangePi.OrangePiID == orangePiID {
				return orangePi.Token, nil
			}
		}
		return "", errors.New("no orangepi found")
	})

	ctrlSet := routes.ControllerSet{
		Auth:      controllers.NewAuthController(serviceSet.Auth),
		Admin:     controllers.NewAdminController(serviceSet.Admin),
		OrangePi:  controllers.NewOrangePiController(serviceSet.OrangePi),
		Building:  controllers.NewBuildingController(serviceSet.Building),
		Device:    controllers.NewDeviceController(serviceSet.Device, serviceSet.OrangePi),
		PublicNet: controllers.NewPublicNetController(serviceSet.PublicNet),
		NVR:       controllers.NewNVRController(serviceSet.NVR),
	}

	middlewareSet := routes.MiddlewareSet{
		Auth: middlewares.NewAuthMiddleware(serviceSet.Auth),
	}

	// 初始化健康检查器
	// 默认每30分钟检查一次，可通过 HEALTH_CHECK_INTERVAL 环境变量配置（单位：分钟）
	healthCheckInterval := 30 * time.Minute
	if intervalStr := os.Getenv("HEALTH_CHECK_INTERVAL"); intervalStr != "" {
		if minutes, err := strconv.Atoi(intervalStr); err == nil && minutes > 0 {
			healthCheckInterval = time.Duration(minutes) * time.Minute
		}
	}
	healthChecker := services.NewHealthChecker(db, serviceSet.PublicNet, healthCheckInterval)

	return &Container{
		DB:            db,
		Services:      serviceSet,
		Controllers:   ctrlSet,
		Middlewares:   middlewareSet,
		HealthChecker: healthChecker,
	}, nil
}

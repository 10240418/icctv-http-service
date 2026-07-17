package services

// AuthService Methods:
//0. NewAuthService(adminService *AdminService) -> 初始化认证服务依赖
//1. PublicToken() -> 返回公开访问 Token
//2. Login(ctx context.Context, username, password string) -> 校验管理员并签发 JWT
//3. ValidateToken(tokenStr string) -> 解析并验证 JWT

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"icctv-http-service/models"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// AuthServiceInterface 定义认证业务能力
type AuthServiceInterface interface {
	GeneratePublicToken(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenResponse, error)     //1.生成公开访问Token(24小时有效)
	GeneratePublicTokenV2(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenV2Response, error) //2.生成带频道备注的公开访问Token
	GeneratePermanentToken(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenResponse, error)  //2.生成永久访问Token
	Login(ctx context.Context, username, password string) (*AuthToken, error)                                 //3.校验账号密码并签发JWT
	ValidateToken(tokenStr string) (*AdminClaims, error)                                                      //4.验证JWT并返回Claims
}

// AuthService 认证相关逻辑
type AuthService struct {
	db                  *gorm.DB
	adminService        *AdminService
	orangePiService     *OrangePiService
	buildingService     *BuildingService
	videoTokenSecretKey string
	jwtSecret           []byte
	tokenTTL            time.Duration
}

// AuthToken JWT 返回体
type AuthToken struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// AdminClaims JWT Claims
type AdminClaims struct {
	AdminID  int64  `json:"adminId"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// VideoTokenPayload 视频Token的Payload结构
type VideoTokenPayload struct {
	Channels   []string `json:"channels"`
	BuildingID string   `json:"building_id"`
	IsStaff    bool     `json:"is_staff"`
	Exp        int64    `json:"exp"`
	Iat        int64    `json:"iat"`
}

// OrangePiURLs OrangePi的URL列表
type OrangePiURLs struct {
	OrangePiID   int64    `json:"orangepi_id"`
	OrangePiName string   `json:"orangepi_name"`
	IsActive     bool     `json:"is_active"`
	Token        string   `json:"token"`
	URLs         []string `json:"urls"`
}

// PublicTokenResponse 公开Token响应
type PublicTokenResponse struct {
	OrangePis []OrangePiURLs `json:"orangepis"`
}

// OrangePiURLsV2 是带有频道备注的公开接口响应项。
type OrangePiURLsV2 struct {
	OrangePiID     int64             `json:"orangepi_id"`
	OrangePiName   string            `json:"orangepi_name"`
	IsActive       bool              `json:"is_active"`
	ChannelRemarks map[string]string `json:"channel_remarks"`
	Token          string            `json:"token"`
	URLs           []string          `json:"urls"`
}

// PublicTokenV2Response 公开接口 V2 响应。
type PublicTokenV2Response struct {
	OrangePis []OrangePiURLsV2 `json:"orangepis"`
}

// 0. NewAuthService 构造函数，加载基础配置
func NewAuthService(db *gorm.DB, adminService *AdminService, orangePiService *OrangePiService, buildingService *BuildingService) *AuthService {
	// 读取JWT秘钥
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "icctv-secret"
	}

	// 读取视频Token秘钥
	videoSecret := os.Getenv("VIDEO_TOKEN_SECRET_KEY")
	if videoSecret == "" {
		videoSecret = "&gfbrffuk8_$f(ip1&7@_jt#7d#ocn&xdq7^)ctx8k$^a%k(7e"
	}

	// 读取JWT TTL
	ttlMinutes := 1440 // 24小时
	if val := os.Getenv("JWT_TTL_MINUTES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			ttlMinutes = parsed
		}
	}

	return &AuthService{
		db:                  db,
		adminService:        adminService,
		orangePiService:     orangePiService,
		buildingService:     buildingService,
		videoTokenSecretKey: videoSecret,
		jwtSecret:           []byte(secret),
		tokenTTL:            time.Duration(ttlMinutes) * time.Minute,
	}
}

// 1. GeneratePublicToken 生成公开访问 Token 和 URLs
// 参数：ismartID - 建筑ISmartID, isStaff - 是否员工
// 逻辑：
//   - 每个 OrangePi 独立生成 Token
//   - is_staff=true: 返回该 OrangePi 的 AllChannels（如果存在）或 UserChannels
//   - is_staff=false: 只返回该 OrangePi 的 UserChannels
//   - 无论 is_active 状态如何，都会返回 OrangePi 信息和 URLs
func (s *AuthService) GeneratePublicToken(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenResponse, error) {
	// 检查建筑是否存在
	var building models.Building
	if err := s.db.WithContext(ctx).Where("ismart_id = ?", ismartID).First(&building).Error; err != nil {
		return nil, fmt.Errorf("building not found: %s", ismartID)
	}

	// 获取该建筑关联的所有 OrangePi 设备（包括非活跃的）
	var orangePis []models.OrangePi
	if err := s.db.WithContext(ctx).
		Joins("JOIN orangepi_buildings ON orangepi_buildings.orange_pi_id = orangepis.id").
		Where("orangepi_buildings.building_id = ?", building.ID).
		Order("orangepis.id ASC").
		Find(&orangePis).Error; err != nil {
		return nil, fmt.Errorf("failed to query orangepi devices: %w", err)
	}

	if len(orangePis) == 0 {
		return nil, fmt.Errorf("no orangepi devices found for building: %s", ismartID)
	}
	// 获取公网配置
	var publicNetConfig models.PublicNetConfig
	if err := s.db.WithContext(ctx).First(&publicNetConfig).Error; err != nil {
		return nil, fmt.Errorf("public network configuration not found")
	}

	// 按 OrangePi 分组构建结果列表
	orangePiURLsList := make([]OrangePiURLs, 0)

	for _, opi := range orangePis {
		// 确定该 OrangePi 可访问的 channels
		var channelsToUse []int
		if isStaff {
			// 员工模式：优先使用 AllChannels，如果为空则使用 UserChannels
			channelsToUse = opi.AllChannels
			if len(channelsToUse) == 0 {
				channelsToUse = opi.UserChannels
			}
		} else {
			// 普通用户模式：只能访问 UserChannels
			channelsToUse = opi.UserChannels
		}
		// 构建 channel 名称列表（用于生成 token）
		channelNames := make([]string, 0, len(channelsToUse))
		for _, ch := range channelsToUse {
			channelNames = append(channelNames, fmt.Sprintf("channel%d", ch))
		}

		// 为该 OrangePi 生成独立的 Token
		var token string
		var err error
		if len(channelNames) == 0 {
			// 如果没有 channels，生成一个空 channels 的 token
			token, err = s.generateHMACToken(ismartID, []string{}, isStaff)
		} else {
			token, err = s.generateHMACToken(ismartID, channelNames, isStaff)
		}
		if err != nil {
			return nil, err
		}

		// 构建 URLs
		urls := make([]string, 0, len(channelsToUse))
		for _, ch := range channelsToUse {
			url := fmt.Sprintf("http://%s:%d/channel%d?token=%s",
				publicNetConfig.ExternalIP,
				opi.ICCTVAuthServiceRemotePort,
				ch,
				token)
			urls = append(urls, url)
		}

		orangePiURLs := OrangePiURLs{
			OrangePiID:   opi.ID,
			OrangePiName: opi.Name,
			IsActive:     opi.IsActive,
			Token:        token,
			URLs:         urls,
		}

		orangePiURLsList = append(orangePiURLsList, orangePiURLs)
	}

	return &PublicTokenResponse{
		OrangePis: orangePiURLsList,
	}, nil
}

// GeneratePublicTokenV2 生成带频道备注和大厦频道权限的公开访问 Token。
// 旧版 /api/auth/public 不读取大厦级频道规则，也不返回频道备注，以保证兼容性。
func (s *AuthService) GeneratePublicTokenV2(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenV2Response, error) {
	var building models.Building
	if err := s.db.WithContext(ctx).Where("ismart_id = ?", ismartID).First(&building).Error; err != nil {
		return nil, fmt.Errorf("building not found: %s", ismartID)
	}

	var orangePis []models.OrangePi
	if err := s.db.WithContext(ctx).
		Joins("JOIN orangepi_buildings ON orangepi_buildings.orange_pi_id = orangepis.id").
		Where("orangepi_buildings.building_id = ?", building.ID).
		Order("orangepis.id ASC").
		Find(&orangePis).Error; err != nil {
		return nil, fmt.Errorf("failed to query orangepi devices: %w", err)
	}
	if len(orangePis) == 0 {
		return nil, fmt.Errorf("no orangepi devices found for building: %s", ismartID)
	}

	channelRules, err := s.loadBuildingChannelRules(ctx, building.ID)
	if err != nil {
		return nil, err
	}
	var publicNetConfig models.PublicNetConfig
	if err := s.db.WithContext(ctx).First(&publicNetConfig).Error; err != nil {
		return nil, fmt.Errorf("public network configuration not found")
	}

	result := make([]OrangePiURLsV2, 0, len(orangePis))
	for _, opi := range orangePis {
		var channelsToUse []int
		if isStaff {
			channelsToUse = opi.AllChannels
			if len(channelsToUse) == 0 {
				channelsToUse = opi.UserChannels
			}
		} else {
			channelsToUse = opi.UserChannels
		}
		channelsToUse = restrictChannels(channelsToUse, channelRules[opi.ID])

		channelNames := make([]string, 0, len(channelsToUse))
		for _, channel := range channelsToUse {
			channelNames = append(channelNames, fmt.Sprintf("channel%d", channel))
		}
		token, err := s.generateHMACToken(ismartID, channelNames, isStaff)
		if err != nil {
			return nil, err
		}

		urls := make([]string, 0, len(channelsToUse))
		for _, channel := range channelsToUse {
			urls = append(urls, fmt.Sprintf("http://%s:%d/channel%d?token=%s",
				publicNetConfig.ExternalIP, opi.ICCTVAuthServiceRemotePort, channel, token))
		}
		result = append(result, OrangePiURLsV2{
			OrangePiID:     opi.ID,
			OrangePiName:   opi.Name,
			IsActive:       opi.IsActive,
			ChannelRemarks: channelRemarks(opi.ChannelRemarks, channelsToUse),
			Token:          token,
			URLs:           urls,
		})
	}

	return &PublicTokenV2Response{OrangePis: result}, nil
}

// GeneratePermanentToken 生成永久访问 Token 和 URLs
// 与 GeneratePublicToken 逻辑相同，但 Token 永不过期
func (s *AuthService) GeneratePermanentToken(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenResponse, error) {
	// 检查建筑是否存在
	var building models.Building
	if err := s.db.WithContext(ctx).Where("ismart_id = ?", ismartID).First(&building).Error; err != nil {
		return nil, fmt.Errorf("building not found: %s", ismartID)
	}

	// 获取该建筑关联的所有 OrangePi 设备
	var orangePis []models.OrangePi
	if err := s.db.WithContext(ctx).
		Joins("JOIN orangepi_buildings ON orangepi_buildings.orange_pi_id = orangepis.id").
		Where("orangepi_buildings.building_id = ?", building.ID).
		Order("orangepis.id ASC").
		Find(&orangePis).Error; err != nil {
		return nil, fmt.Errorf("failed to query orangepi devices: %w", err)
	}

	if len(orangePis) == 0 {
		return nil, fmt.Errorf("no orangepi devices found for building: %s", ismartID)
	}
	// 获取公网配置
	var publicNetConfig models.PublicNetConfig
	if err := s.db.WithContext(ctx).First(&publicNetConfig).Error; err != nil {
		return nil, fmt.Errorf("public network configuration not found")
	}

	// 按 OrangePi 分组构建结果列表
	orangePiURLsList := make([]OrangePiURLs, 0)

	for _, opi := range orangePis {
		var channelsToUse []int
		if isStaff {
			channelsToUse = opi.AllChannels
			if len(channelsToUse) == 0 {
				channelsToUse = opi.UserChannels
			}
		} else {
			channelsToUse = opi.UserChannels
		}
		channelNames := make([]string, 0, len(channelsToUse))
		for _, ch := range channelsToUse {
			channelNames = append(channelNames, fmt.Sprintf("channel%d", ch))
		}

		// 生成永久 Token（expiry = 0 表示永不过期）
		var token string
		var err error
		if len(channelNames) == 0 {
			token, err = s.generateHMACTokenWithExpiry(ismartID, []string{}, isStaff, 0)
		} else {
			token, err = s.generateHMACTokenWithExpiry(ismartID, channelNames, isStaff, 0)
		}
		if err != nil {
			return nil, err
		}

		urls := make([]string, 0, len(channelsToUse))
		for _, ch := range channelsToUse {
			url := fmt.Sprintf("http://%s:%d/channel%d?token=%s",
				publicNetConfig.ExternalIP,
				opi.ICCTVAuthServiceRemotePort,
				ch,
				token)
			urls = append(urls, url)
		}

		orangePiURLs := OrangePiURLs{
			OrangePiID:   opi.ID,
			OrangePiName: opi.Name,
			IsActive:     opi.IsActive,
			Token:        token,
			URLs:         urls,
		}

		orangePiURLsList = append(orangePiURLsList, orangePiURLs)
	}

	return &PublicTokenResponse{
		OrangePis: orangePiURLsList,
	}, nil
}

func (s *AuthService) loadBuildingChannelRules(ctx context.Context, buildingID int64) (map[int64][]int, error) {
	var links []models.OrangePiBuilding
	if err := s.db.WithContext(ctx).Where("building_id = ?", buildingID).Find(&links).Error; err != nil {
		return nil, fmt.Errorf("failed to query building channel rules: %w", err)
	}
	result := make(map[int64][]int, len(links))
	for _, link := range links {
		result[link.OrangePiID] = append([]int(nil), link.AllowedChannels...)
	}
	return result, nil
}

func channelRemarks(values map[string]string, channels []int) map[string]string {
	result := make(map[string]string, len(channels))
	for _, channel := range channels {
		key := fmt.Sprintf("channel%d", channel)
		result[key] = strings.TrimSpace(values[key])
	}
	return result
}

// generateHMACToken 生成 HMAC-SHA256 签名的视频 Token（24小时有效）
// 格式：base64(payload).signature
func (s *AuthService) generateHMACToken(buildingID string, channels []string, isStaff bool) (string, error) {
	return s.generateHMACTokenWithExpiry(buildingID, channels, isStaff, 86400)
}

// generateHMACTokenWithExpiry 生成 HMAC-SHA256 签名的视频 Token（可指定过期时间）
// expiry: 过期秒数，0 表示永不过期
func (s *AuthService) generateHMACTokenWithExpiry(buildingID string, channels []string, isStaff bool, expirySeconds int64) (string, error) {
	// 创建 Payload
	now := time.Now().Unix()
	var expiry int64
	if expirySeconds > 0 {
		expiry = now + expirySeconds
	} else {
		// 永不过期：设置为 100 年后
		expiry = now + 100*365*24*3600
	}

	payload := VideoTokenPayload{
		Channels:   channels,
		BuildingID: buildingID,
		IsStaff:    isStaff,
		Exp:        expiry,
		Iat:        now,
	}

	// JSON 序列化（键排序保证一致性）
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Base64 编码 payload
	payloadB64 := base64.URLEncoding.EncodeToString(payloadBytes)

	// 生成 HMAC-SHA256 签名
	h := hmac.New(sha256.New, []byte(s.videoTokenSecretKey))
	h.Write(payloadBytes)
	signature := fmt.Sprintf("%x", h.Sum(nil))

	// 组合 token: payload.signature
	token := fmt.Sprintf("%s.%s", payloadB64, signature)
	return token, nil
}

// 2. Login 验证管理员并生成 JWT
func (s *AuthService) Login(ctx context.Context, username, password string) (*AuthToken, error) {
	admin, err := s.adminService.VerifyCredential(ctx, username, password)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(s.tokenTTL)
	claims := AdminClaims{
		AdminID:  admin.ID,
		Username: admin.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.FormatInt(admin.ID, 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &AuthToken{
		AccessToken: tokenStr,
		ExpiresAt:   expiresAt,
	}, nil
}

// 3. ValidateToken 校验 JWT
func (s *AuthService) ValidateToken(tokenStr string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&AdminClaims{},
		func(t *jwt.Token) (interface{}, error) {
			return s.jwtSecret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

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
	"time"

	"icctv-http-service/models"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// AuthServiceInterface 定义认证业务能力
type AuthServiceInterface interface {
	GeneratePublicToken(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenResponse, error) //1.生成公开访问Token
	Login(ctx context.Context, username, password string) (*AuthToken, error)                             //2.校验账号密码并签发JWT
	ValidateToken(tokenStr string) (*AdminClaims, error)                                                  //3.验证JWT并返回Claims
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
	URLs         []string `json:"urls"`
}

// PublicTokenResponse 公开Token响应
type PublicTokenResponse struct {
	Token      string         `json:"token"`
	OrangePis  []OrangePiURLs `json:"orangepis"`
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
	ttlMinutes := 120
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
//   - is_staff=true: 返回该建筑下所有OrangePi的AllChannels（如果存在）或UserChannels中的channels
//   - is_staff=false: 只返回每个OrangePi的UserChannels中指定的channels
func (s *AuthService) GeneratePublicToken(ctx context.Context, ismartID string, isStaff bool) (*PublicTokenResponse, error) {
	// 检查建筑是否存在
	var building models.Building
	if err := s.db.WithContext(ctx).Where("ismart_id = ?", ismartID).First(&building).Error; err != nil {
		return nil, fmt.Errorf("building not found: %s", ismartID)
	}

	// 获取该建筑关联的所有 OrangePi 设备
	var orangePis []models.OrangePi
	if err := s.db.WithContext(ctx).Where("ismart_id = ? AND is_active = ?", ismartID, true).Find(&orangePis).Error; err != nil {
		return nil, fmt.Errorf("failed to query orangepi devices: %w", err)
	}

	if len(orangePis) == 0 {
		return nil, fmt.Errorf("no active orangepi devices found for building: %s", ismartID)
	}

	// 获取公网配置
	var publicNetConfig models.PublicNetConfig
	if err := s.db.WithContext(ctx).First(&publicNetConfig).Error; err != nil {
		return nil, fmt.Errorf("public network configuration not found")
	}

	// 收集所有可访问的 channels（用于生成 token）
	var allChannels []string
	channelSet := make(map[string]bool) // 去重

	// 按 OrangePi 分组构建 URL 列表
	orangePiURLsList := make([]OrangePiURLs, 0)

	for _, opi := range orangePis {
		orangePiURLs := OrangePiURLs{
			OrangePiID:   opi.ID,
			OrangePiName: opi.Name,
			URLs:         make([]string, 0),
		}

		// 确定该 OrangePi 下哪些 channels 可访问
		accessibleChannels := make(map[int]bool)

		if isStaff {
			// 员工模式：优先使用 AllChannels，如果为空则使用 UserChannels
			channelsToUse := opi.AllChannels
			if len(channelsToUse) == 0 {
				channelsToUse = opi.UserChannels
			}
			// 如果仍然为空，跳过这个 OrangePi
			if len(channelsToUse) == 0 {
				continue
			}
			for _, ch := range channelsToUse {
				accessibleChannels[ch] = true
				channelName := fmt.Sprintf("channel%d", ch)
				if !channelSet[channelName] {
					channelSet[channelName] = true
					allChannels = append(allChannels, channelName)
				}
			}
		} else {
			// 普通用户模式：只能访问 UserChannels 中指定的 channels
			// 如果 UserChannels 为空，跳过这个 OrangePi
			if len(opi.UserChannels) == 0 {
				continue
			}
			for _, ch := range opi.UserChannels {
				accessibleChannels[ch] = true
				channelName := fmt.Sprintf("channel%d", ch)
				if !channelSet[channelName] {
					channelSet[channelName] = true
					allChannels = append(allChannels, channelName)
				}
			}
		}

		// 如果该 OrangePi 有可访问的 channels，则生成 token（稍后统一生成）
		// 先构建该 OrangePi 的 URL 列表（使用占位符）
		for ch := range accessibleChannels {
			// 暂时不添加 token，等统一生成后再添加
			orangePiURLs.URLs = append(orangePiURLs.URLs, fmt.Sprintf("%d", ch))
		}

		// 只添加有可访问 channels 的 OrangePi
		if len(orangePiURLs.URLs) > 0 {
			orangePiURLsList = append(orangePiURLsList, orangePiURLs)
		}
	}

	if len(allChannels) == 0 {
		return nil, fmt.Errorf("no accessible channels found")
	}

	// 生成视频 Token (HMAC-SHA256 签名格式)
	token, err := s.generateHMACToken(ismartID, allChannels, isStaff)
	if err != nil {
		return nil, err
	}

	// 现在用真实的 URL 替换占位符
	for i := range orangePiURLsList {
		// 根据 OrangePiID 找到对应的 OrangePi
		var currentOpi models.OrangePi
		for _, opi := range orangePis {
			if opi.ID == orangePiURLsList[i].OrangePiID {
				currentOpi = opi
				break
			}
		}

		realURLs := make([]string, 0)
		for _, channelStr := range orangePiURLsList[i].URLs {
			// channelStr 是 channel 号（如 "1", "2"）
			url := fmt.Sprintf("http://%s:%d/channel%s?token=%s",
				publicNetConfig.ExternalIP,
				currentOpi.ICCTVAuthServiceRemotePort,
				channelStr,
				token)
			realURLs = append(realURLs, url)
		}
		orangePiURLsList[i].URLs = realURLs
	}

	return &PublicTokenResponse{
		Token:     token,
		OrangePis: orangePiURLsList,
	}, nil
}

// generateHMACToken 生成 HMAC-SHA256 签名的视频 Token
// 格式：base64(payload).signature
func (s *AuthService) generateHMACToken(buildingID string, channels []string, isStaff bool) (string, error) {
	// 创建 Payload
	now := time.Now().Unix()
	expiry := now + 86400 // 24小时有效期

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

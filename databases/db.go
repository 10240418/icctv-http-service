package databases

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"icctv-http-service/models"

	"golang.org/x/crypto/bcrypt"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var (
	db   *gorm.DB
	once sync.Once
)

// Init 初始化数据库连接（默认 sqlite，可通过 DB_DRIVER=mysql 切换到 MySQL）
func Init() (*gorm.DB, error) {
	var initErr error

	once.Do(func() {
		driver := os.Getenv("DB_DRIVER")
		if driver == "" {
			driver = "sqlite"
		}

		cfg := &gorm.Config{
			Logger:                                   logger.Default.LogMode(logger.Silent),
			DisableForeignKeyConstraintWhenMigrating: true,
		}

		var (
			conn *gorm.DB
			err  error
		)

		switch driver {
		case "mysql":
			host := getenvDefault("DB_HOST", "mysql")
			port := getenvDefault("DB_PORT", "3306")
			user := getenvDefault("DB_USER", "icctv")
			pass := getenvDefault("DB_PASS", "icctv_pass")
			name := getenvDefault("DB_NAME", "icctv")

			dsn := fmt.Sprintf(
				"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				user,
				pass,
				host,
				port,
				name,
			)

			conn, err = gorm.Open(gmysql.Open(dsn), cfg)

			if err == nil {
				if fixErr := fixLegacyMySQLIndexes(conn); fixErr != nil {
					log.Printf("Warning: failed to fix legacy MySQL indexes: %v", fixErr)
				}
			}
		default:
			dsn := getenvDefault("DATABASE_PATH", "icctv.db")
			conn, err = gorm.Open(sqlite.Open(dsn), cfg)
		}

		if err != nil {
			initErr = err
			return
		}

		if err := conn.SetupJoinTable(&models.OrangePi{}, "Buildings", &models.OrangePiBuilding{}); err != nil {
			initErr = err
			return
		}

		// 按照依赖顺序迁移表：先建没有外键的表，再建有外键依赖的表
		if err := conn.AutoMigrate(
			&models.Adminer{},
			&models.PublicNetConfig{},
			&models.Building{}, // 必须在 OrangePi 前面，因为 OrangePi 有外键关联
			&models.OrangePi{},
			&models.OrangePiBuilding{},
			&models.NVR{}, // NVR 依赖 Building
		); err != nil {
			initErr = err
			return
		}

		if err := migrateLegacyOrangePiBuildings(conn); err != nil {
			initErr = err
			return
		}

		// 初始化默认数据
		if err := initDefaultData(conn); err != nil {
			log.Printf("Warning: failed to initialize default data: %v", err)
			// 不阻止启动，只记录警告
		}

		db = conn
	})

	return db, initErr
}

// migrateLegacyOrangePiBuildings 将旧版 orangepis.ismart_id 数据幂等迁移到关联表。
func migrateLegacyOrangePiBuildings(db *gorm.DB) error {
	var orangePis []models.OrangePi
	if err := db.Where("ismart_id <> ?", "unbound_temp").Find(&orangePis).Error; err != nil {
		return err
	}

	for _, orangePi := range orangePis {
		var building models.Building
		if err := db.Where("ismart_id = ?", orangePi.ISmartID).First(&building).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}

		link := models.OrangePiBuilding{OrangePiID: orangePi.ID, BuildingID: building.ID}
		if err := db.Where("orange_pi_id = ? AND building_id = ?", orangePi.ID, building.ID).First(&link).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			link.AllowedChannels = legacyOrangePiChannels(orangePi)
			if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&link).Error; err != nil {
				return err
			}
		} else if len(link.AllowedChannels) == 0 {
			link.AllowedChannels = legacyOrangePiChannels(orangePi)
			if err := db.Save(&link).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func legacyOrangePiChannels(orangePi models.OrangePi) []int {
	if len(orangePi.AllChannels) > 0 {
		return append([]int(nil), orangePi.AllChannels...)
	}
	return append([]int(nil), orangePi.UserChannels...)
}

// MustDB 返回数据库实例（若未初始化会 Panic）
func MustDB() *gorm.DB {
	if db != nil {
		return db
	}

	conn, err := Init()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	return conn
}

func getenvDefault(key, defVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defVal
}

// initDefaultData 初始化默认数据（如默认管理员账户）
func initDefaultData(db *gorm.DB) error {
	// 检查是否已有管理员账户
	var adminCount int64
	if err := db.Model(&models.Adminer{}).Count(&adminCount).Error; err != nil {
		return fmt.Errorf("failed to check admin count: %w", err)
	}

	// 如果没有管理员，创建默认管理员账户
	if adminCount == 0 {
		// 密码: 123456 的 bcrypt 哈希值
		// 使用 bcrypt.DefaultCost 生成，与 AdminService 保持一致
		passwordHash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}

		defaultAdmin := &models.Adminer{
			Username:     "admin",
			PasswordHash: string(passwordHash),
		}

		if err := db.Create(defaultAdmin).Error; err != nil {
			return fmt.Errorf("failed to create default admin: %w", err)
		}

		log.Println("✓ Default admin account created (username: admin, password: 123456)")
	}

	return nil
}

// fixLegacyMySQLIndexes 清理旧版遗留索引，避免 AutoMigrate 冲突
func fixLegacyMySQLIndexes(db *gorm.DB) error {
	const query = `
SELECT NON_UNIQUE AS non_unique
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
LIMIT 1`

	var idxInfo struct {
		NonUnique int `gorm:"column:non_unique"`
	}

	err := db.Raw(query, "buildings", "idx_buildings_i_smart_id").Scan(&idxInfo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	// 旧索引为非唯一时删除，交由 GORM 重新创建唯一索引
	if idxInfo.NonUnique == 1 {
		if err := db.Exec("ALTER TABLE `buildings` DROP INDEX `idx_buildings_i_smart_id`").Error; err != nil {
			return err
		}
		log.Println("✓ Dropped legacy non-unique index idx_buildings_i_smart_id")
	}

	return nil
}

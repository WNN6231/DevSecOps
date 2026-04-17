package store

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"devsecops-platform/pkg/common"
)

var DB *gorm.DB

func InitDB(cfg common.DatabaseConfig) error {
	database, err := gorm.Open(mysql.Open(buildDSN(cfg)), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open mysql connection: %w", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}

	configurePool(sqlDB, cfg)

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}

	if err := database.AutoMigrate(&ScanJob{}, &ScanResult{}); err != nil {
		return fmt.Errorf("auto migrate tables: %w", err)
	}

	DB = database
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func buildDSN(cfg common.DatabaseConfig) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.Charset,
		cfg.ParseTime,
		cfg.Loc,
	)
}

func configurePool(sqlDB *sql.DB, cfg common.DatabaseConfig) {
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSec) * time.Second)
}

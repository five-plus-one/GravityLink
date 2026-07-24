package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"gravitylink/backend/internal/model"
)

func ConnectMySQL(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func EnsureSchema(db *gorm.DB) error {
	models := []any{
		&model.User{},
		&model.AuthSession{},
		&model.InstallationState{},
		&model.AuditLog{},
		&model.Domain{},
		&model.Link{},
		&model.ChannelConfig{},
		&model.RoutingStrategy{},
		&model.RoutingTarget{},
		&model.LandingPage{},
		&model.AccessLog{},
		&model.StatDaily{},
		&model.StatHourly{},
		&model.StatGeo{},
		&model.StatDevice{},
		&model.SystemConfig{},
	}
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS access_logs_archive LIKE access_logs").Error; err != nil {
		return err
	}
	return nil
}

func ConnectRedis(addr string, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

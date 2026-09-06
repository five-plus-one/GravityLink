package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gravitylink/backend/internal/model"
)

func ConnectMySQL(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
		}),
	})
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
		&model.SystemConfig{},
	}
	if !db.Migrator().HasTable(&model.Link{}) {
		models = append(models,
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
		)
	}
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS access_logs_archive LIKE access_logs").Error; err != nil {
		return err
	}
	return VerifySchema(db)
}

// VerifySchema 校验已存在表的关键列定义与 GORM 模型一致。
// AutoMigrate 不会修改已有列的约束，历史库可能残留旧结构（例如 users.email NOT NULL），
// 这里显式报错并给出修复提示，避免运行期才暴露。
func VerifySchema(db *gorm.DB) error {
	// 仅校验模型中定义为可空的列；NOT NULL 列由 AutoMigrate 正常处理
	nullableColumns := []struct {
		model  any
		table  string
		column string
	}{
		{&model.User{}, "users", "email"},
	}
	for _, check := range nullableColumns {
		columnTypes, err := db.Migrator().ColumnTypes(check.model)
		if err != nil {
			return fmt.Errorf("inspect column %s.%s: %w", check.table, check.column, err)
		}
		for _, ct := range columnTypes {
			if ct.Name() != check.column {
				continue
			}
			nullable, _ := ct.Nullable()
			if !nullable {
				return fmt.Errorf(
					"数据库表结构与当前版本不一致：列 %s.%s 应允许为空。请执行 `ALTER TABLE %s MODIFY %s VARCHAR(128) NULL` 修复后重试",
					check.table, check.column, check.table, check.column,
				)
			}
		}
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

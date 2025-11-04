package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"code-runner/env"
	"code-runner/internal/database/models"
)

// DB holds the database connection
var DB *gorm.DB

// InitDB initializes the database connection
func InitDB(config *env.DatabaseConfig) error {
	// Build DSN with explicit sslmode parameter
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s timezone=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Name,
		config.Timezone,
		config.SSLMode,
	)

	// Alternative URL format (commented out, use if needed):
	// dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s&timezone=%s",
	// 	config.User,
	// 	config.Password,
	// 	config.Host,
	// 	config.Port,
	// 	config.Name,
	// 	config.SSLMode,
	// 	config.Timezone,
	// )

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("✅ Database connected successfully to %s:%s/%s", config.Host, config.Port, config.Name)

	// Auto-migrate the schema
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// autoMigrate runs auto-migration for all models
func autoMigrate() error {
	models := []interface{}{
		&models.Execution{},
		&models.GeneratedTestCode{},
	}

	for _, model := range models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}

	log.Printf("✅ Database migration completed successfully")
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

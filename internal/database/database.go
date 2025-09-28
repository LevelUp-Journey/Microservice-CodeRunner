package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"code-runner/internal/database/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	database *Database
)

// GetDatabaseConfig reads database configuration from environment variables
func GetDatabaseConfig() *DatabaseConfig {
	port, _ := strconv.Atoi(getEnvOrDefault("DB_PORT", "5432"))

	return &DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     port,
		User:     getEnvOrDefault("DB_USER", "admin"),
		Password: getEnvOrDefault("DB_PASSWORD", "admin"),
		DBName:   getEnvOrDefault("DB_NAME", "coderunner_microservice"),
		SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		TimeZone: getEnvOrDefault("DB_TIMEZONE", "UTC"),
	}
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// BuildDSN builds PostgreSQL DSN from config
func (config *DatabaseConfig) BuildDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		config.Host,
		config.User,
		config.Password,
		config.DBName,
		config.Port,
		config.SSLMode,
		config.TimeZone,
	)
}

// Initialize initializes database connection
func Initialize() (*Database, error) {
	config := GetDatabaseConfig()
	dsn := config.BuildDSN()

	log.Printf("🔗 Connecting to database: %s@%s:%d/%s",
		config.User, config.Host, config.Port, config.DBName)

	// Configure GORM logger
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,         // Disable color
		},
	)

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Connection pool settings
	maxOpenConns, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_OPEN_CONNS", "25"))
	maxIdleConns, _ := strconv.Atoi(getEnvOrDefault("DB_MAX_IDLE_CONNS", "10"))
	maxLifetime, _ := strconv.Atoi(getEnvOrDefault("DB_CONN_MAX_LIFETIME", "3600"))

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(maxLifetime) * time.Second)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database = &Database{DB: db}

	log.Println("✅ Database connection established successfully")

	// Run migrations
	if err := database.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	if database == nil {
		log.Fatal("Database not initialized. Call Initialize() first.")
	}
	return database.DB
}

// GetInstance returns the database instance
func GetInstance() *Database {
	if database == nil {
		log.Fatal("Database not initialized. Call Initialize() first.")
	}
	return database
}

// AutoMigrate runs database migrations
func (d *Database) AutoMigrate() error {
	log.Println("🔄 Running database migrations...")

	// Enable UUID extension
	if err := d.DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		log.Printf("⚠️  Warning: Could not create uuid-ossp extension: %v", err)
	}

	// Run auto migrations for all models
	err := d.DB.AutoMigrate(
		&models.Execution{},
		&models.ExecutionStep{},
		&models.ExecutionLog{},
		&models.TestResult{},
	)

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("✅ Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health checks database health
func (d *Database) Health() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// GetStats returns database connection statistics
func (d *Database) GetStats() map[string]interface{} {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return map[string]interface{}{
			"error": "failed to get underlying sql.DB",
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}

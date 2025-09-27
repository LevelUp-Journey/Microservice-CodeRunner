package database

import (
	"gorm.io/gorm"
)

// Database wraps the GORM database instance with additional functionality
type Database struct {
	DB *gorm.DB
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	TimeZone string
}

// ConnectionStats represents database connection statistics
type ConnectionStats struct {
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
	WaitCount          int64
	WaitDuration       string
	MaxIdleClosed      int64
	MaxLifetimeClosed  int64
}

package variables

// Config holds all application configuration
type Config struct {
	Port       string
	GRPCPort   string
	APIVersion string
	AppName    string
	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBTimeZone string
}

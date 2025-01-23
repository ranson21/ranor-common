package config

import (
	"fmt"
	"os"
	"time"
)

type Environment string

const (
	Local       Environment = "local"
	Development Environment = "development"
	Production  Environment = "production"
)

type DatabaseConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	DBName       string
	Schema       string
	SSLMode      string
	MaxConns     int32
	MinConns     int32
	MaxIdleTime  time.Duration
	MaxLifetime  time.Duration
	UseIAMAuth   bool
	InstanceName string // For Cloud SQL
}

func NewDatabaseConfig(env Environment, service string) DatabaseConfig {
	switch env {
	case Local:
		// Use local postgres
		return DatabaseConfig{
			Host:     "localhost",
			Port:     "5433", // Local postgres port
			User:     "postgres",
			Password: "postgres",
			DBName:   "ranor",
			Schema:   service,
			SSLMode:  "disable",
		}
	case Development, Production:
		// Use Cloud SQL
		dbUser := os.Getenv(fmt.Sprintf("%s_DB_USER", service))
		if dbUser == "" {
			dbUser = "postgres" // Default Cloud SQL user
		}

		config := DatabaseConfig{
			Host:         "localhost", // Cloud SQL Proxy always runs locally
			Port:         "5432",
			User:         dbUser,
			Password:     os.Getenv(fmt.Sprintf("%s_DB_PASSWORD", service)),
			DBName:       fmt.Sprintf("ranor_%s", service),
			SSLMode:      "disable",         // Proxy handles encryption
			UseIAMAuth:   env == Production, // Use IAM auth in production
			InstanceName: os.Getenv("INSTANCE_CONNECTION_NAME"),
		}

		if env == Development {
			config.Schema = service // Use schemas in dev
		}

		return config
	default:
		panic("unknown environment")
	}
}

// ConnectionString returns the appropriate connection string for the environment
func (c DatabaseConfig) ConnectionString() string {
	if c.UseIAMAuth {
		// For production with IAM authentication
		return fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.DBName, c.SSLMode,
		)
	}

	// For local/development with password authentication
	base := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)

	if c.Schema != "" {
		base += fmt.Sprintf(" search_path=%s", c.Schema)
	}

	return base
}

package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
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

func NewDatabaseConfig(env Environment, service string) *DatabaseConfig {
	// MaxConns: Fetch from ENV, default to 50
	maxConns := 50
	if val := os.Getenv("PG_MAX_CONNS"); val != "" {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			maxConns = parsedVal
		} else {
			log.Printf("Invalid value for PG_MAX_CONNS, falling back to default: %d", maxConns)
		}
	}

	// MinConns: Fetch from ENV, default to 5
	minConns := 5
	if val := os.Getenv("PG_MIN_CONNS"); val != "" {
		if parsedVal, err := strconv.Atoi(val); err == nil {
			minConns = parsedVal
		} else {
			log.Printf("Invalid value for PG_MIN_CONNS, falling back to default: %d", minConns)
		}
	}

	// MaxConnIdleTime: Fetch from ENV, default to 5 minutes
	maxConnIdleTime := 5 * time.Minute
	if val := os.Getenv("PG_MAX_IDLE_TIME"); val != "" {
		if parsedVal, err := time.ParseDuration(val); err == nil {
			maxConnIdleTime = parsedVal
		} else {
			log.Printf("Invalid value for PG_MAX_IDLE_TIME, falling back to default: %v", maxConnIdleTime)
		}
	}

	// MaxConnLifetime: Fetch from ENV, default to 1 hour
	maxConnLifetime := 1 * time.Hour
	if val := os.Getenv("PG_MAX_LIFETIME"); val != "" {
		if parsedVal, err := time.ParseDuration(val); err == nil {
			maxConnLifetime = parsedVal
		} else {
			log.Printf("Invalid value for PG_MAX_LIFETIME, falling back to default: %v", maxConnLifetime)
		}
	}

	switch env {
	case Local:
		// Use local postgres
		return &DatabaseConfig{
			Host:        "localhost",
			Port:        "5432", // Local postgres port
			User:        "postgres",
			Password:    "postgres",
			DBName:      "ranor",
			Schema:      service,
			SSLMode:     "disable",
			MaxConns:    int32(maxConns),
			MinConns:    int32(minConns),
			MaxIdleTime: maxConnIdleTime,
			MaxLifetime: maxConnLifetime,
		}
	case Development, Production:
		// Use Cloud SQL
		dbUser := os.Getenv(fmt.Sprintf("%s_DB_USER", service))
		if dbUser == "" {
			dbUser = "postgres" // Default Cloud SQL user
		}

		config := &DatabaseConfig{
			Host:         "localhost", // Cloud SQL Proxy always runs locally
			Port:         "5432",
			User:         dbUser,
			Password:     os.Getenv(fmt.Sprintf("%s_DB_PASSWORD", service)),
			DBName:       fmt.Sprintf("ranor_%s", service),
			SSLMode:      "disable",         // Proxy handles encryption
			UseIAMAuth:   env == Production, // Use IAM auth in production
			InstanceName: os.Getenv("INSTANCE_CONNECTION_NAME"),
			MaxConns:     int32(maxConns),
			MinConns:     int32(minConns),
			MaxIdleTime:  maxConnIdleTime,
			MaxLifetime:  maxConnLifetime,
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

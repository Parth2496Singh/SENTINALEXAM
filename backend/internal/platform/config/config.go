package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents all application runtime configuration.
type Config struct {
	Environment string
	LogLevel    string

	// HTTP Server
	HTTPPort     int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Database
	DatabaseHost            string
	DatabasePort            int
	DatabaseUser            string
	DatabasePassword        string
	DatabaseName            string
	DatabaseSSLMode         string
	DatabaseMaxOpenConns    int
	DatabaseMaxIdleConns    int
	DatabaseConnMaxLifetime time.Duration

	// NATS JetStream
	NATSURL           string
	NATSClusterID     string
	NATSMaxReconnects int
	NATSReconnectWait time.Duration

	// Temporal
	TemporalHostPort  string
	TemporalNamespace string

	// Keycloak IAM
	KeycloakURL          string
	KeycloakRealm        string
	KeycloakClientID     string
	KeycloakClientSecret string
	KeycloakIssuer       string
	KeycloakAudience     string

	// Observability
	MetricsPort int
	OTLPEndpoint string
	ServiceName  string
}

// LoadFromEnv initializes the Config from environment variables with safe defaults.
func LoadFromEnv() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		HTTPPort:     getEnvInt("HTTP_PORT", 8080),
		ReadTimeout:  getEnvDuration("READ_TIMEOUT", 15*time.Second),
		WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:  getEnvDuration("IDLE_TIMEOUT", 60*time.Second),

		DatabaseHost:            getEnv("DATABASE_HOST", "localhost"),
		DatabasePort:            getEnvInt("DATABASE_PORT", 5432),
		DatabaseUser:            getEnv("DATABASE_USER", "sentinel"),
		DatabasePassword:        getEnv("DATABASE_PASSWORD", "sentinel_dev_password"),
		DatabaseName:            getEnv("DATABASE_NAME", "sentinelexam"),
		DatabaseSSLMode:         getEnv("DATABASE_SSLMODE", "disable"),
		DatabaseMaxOpenConns:    getEnvInt("DATABASE_MAX_OPEN_CONNS", 25),
		DatabaseMaxIdleConns:    getEnvInt("DATABASE_MAX_IDLE_CONNS", 10),
		DatabaseConnMaxLifetime: getEnvDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),

		NATSURL:           getEnv("NATS_URL", "nats://localhost:4222"),
		NATSClusterID:     getEnv("NATS_CLUSTER_ID", "sentinel-cluster"),
		NATSMaxReconnects: getEnvInt("NATS_MAX_RECONNECTS", 10),
		NATSReconnectWait: getEnvDuration("NATS_RECONNECT_WAIT", 2*time.Second),

		TemporalHostPort:  getEnv("TEMPORAL_HOST_PORT", "localhost:7233"),
		TemporalNamespace: getEnv("TEMPORAL_NAMESPACE", "default"),

		KeycloakURL:          getEnv("KEYCLOAK_URL", "http://localhost:8081"),
		KeycloakRealm:        getEnv("KEYCLOAK_REALM", "sentinel"),
		KeycloakClientID:     getEnv("KEYCLOAK_CLIENT_ID", "sentinel-api"),
		KeycloakClientSecret: getEnv("KEYCLOAK_CLIENT_SECRET", "sentinel-client-secret"),
		KeycloakIssuer:       getEnv("KEYCLOAK_ISSUER", "http://localhost:8081/realms/sentinel"),
		KeycloakAudience:     getEnv("KEYCLOAK_AUDIENCE", "sentinel-api"),

		MetricsPort:  getEnvInt("METRICS_PORT", 9090),
		OTLPEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317"),
		ServiceName:  getEnv("OTEL_SERVICE_NAME", "sentinel-backend"),
	}
}

// DatabaseDSN builds the PostgreSQL connection string.
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DatabaseUser,
		c.DatabasePassword,
		c.DatabaseHost,
		c.DatabasePort,
		c.DatabaseName,
		c.DatabaseSSLMode,
	)
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

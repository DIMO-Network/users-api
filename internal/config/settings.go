package config

import (
	"github.com/DIMO-Network/shared/db"
)

// Settings contains the application config
type Settings struct {
	Port     string      `yaml:"PORT"`
	GRPCPort string      `yaml:"GRPC_PORT"`
	LogLevel string      `yaml:"LOG_LEVEL"`
	DB       db.Settings `yaml:"DB"`
	// DBUser               string `yaml:"DB_USER"`
	// DBPassword           string `yaml:"DB_PASSWORD"`
	// DBPort               string `yaml:"DB_PORT"`
	// DBHost               string `yaml:"DB_HOST"`
	// DBName               string `yaml:"DB_NAME"`
	// DBMaxOpenConnections int    `yaml:"DB_MAX_OPEN_CONNECTIONS"`
	// DBMaxIdleConnections int    `yaml:"DB_MAX_IDLE_CONNECTIONS"`
	ServiceName        string `yaml:"SERVICE_NAME"`
	EmailHost          string `yaml:"EMAIL_HOST"`
	EmailPort          string `yaml:"EMAIL_PORT"`
	EmailUsername      string `yaml:"EMAIL_USERNAME"`
	EmailPassword      string `yaml:"EMAIL_PASSWORD"`
	EmailFrom          string `yaml:"EMAIL_FROM"`
	JWTKeySetURL       string `yaml:"JWT_KEY_SET_URL"`
	KafkaBrokers       string `yaml:"KAFKA_BROKERS"`
	EventsTopic        string `yaml:"EVENTS_TOPIC"`
	MonitoringPort     string `yaml:"MON_PORT"`
	DevicesAPIGRPCAddr string `yaml:"DEVICES_API_GRPC_ADDR"`
}

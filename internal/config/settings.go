package config

import (
	"github.com/DIMO-Network/shared/db"
)

// Settings contains the application config
type Settings struct {
	Environment        string      `yaml:"ENVIRONMENT"`
	Port               string      `yaml:"PORT"`
	GRPCPort           string      `yaml:"GRPC_PORT"`
	LogLevel           string      `yaml:"LOG_LEVEL"`
	DB                 db.Settings `yaml:"DB"`
	ServiceName        string      `yaml:"SERVICE_NAME"`
	JWTKeySetURL       string      `yaml:"JWT_KEY_SET_URL"`
	MonitoringPort     string      `yaml:"MON_PORT"`
	DevicesAPIGRPCAddr string      `yaml:"DEVICES_API_GRPC_ADDR"`

	VehicleNFTAddr string `yaml:"VEHICLE_NFT_ADDR"`
	ADNFTAddr      string `yaml:"AD_NFT_ADDR"`
	TokenAddr      string `yaml:"TOKEN_ADDR"`

	MainRPCURL string `yaml:"MAIN_RPC_URL"`
}

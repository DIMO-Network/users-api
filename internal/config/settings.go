package config

import "fmt"

// Settings contains the application config
type Settings struct {
	Port                 string `yaml:"PORT"`
	LogLevel             string `yaml:"LOG_LEVEL"`
	DBUser               string `yaml:"DB_USER"`
	DBPassword           string `yaml:"DB_PASSWORD"`
	DBPort               string `yaml:"DB_PORT"`
	DBHost               string `yaml:"DB_HOST"`
	DBName               string `yaml:"DB_NAME"`
	DBMaxOpenConnections int    `yaml:"DB_MAX_OPEN_CONNECTIONS"`
	DBMaxIdleConnections int    `yaml:"DB_MAX_IDLE_CONNECTIONS"`
	ServiceName          string `yaml:"SERVICE_NAME"`
	EmailHost            string `yaml:"EMAIL_HOST"`
	EmailPort            string `yaml:"EMAIL_PORT"`
	EmailUsername        string `yaml:"EMAIL_USERNAME"`
	EmailPassword        string `yaml:"EMAIL_PASSWORD"`
	EmailFrom            string `yaml:"EMAIL_FROM"`
	JWTKeySetURL         string `yaml:"JWT_KEY_SET_URL"`
	AdminPassword        string `yaml:"ADMIN_PASSWORD"`
	CustomerIOSiteID     string `yaml:"CUSTOMER_IO_SITE_ID"`
	CustomerIOApiKey     string `yaml:"CUSTOMER_IO_API_KEY"`
}

// GetWriterDSN builds the connection string to the db writer - for now same as reader
func (app *Settings) GetWriterDSN(withSearchPath bool) string {
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		app.DBUser,
		app.DBPassword,
		app.DBName,
		app.DBHost,
		app.DBPort,
	)
	if withSearchPath {
		dsn = fmt.Sprintf("%s search_path=%s", dsn, app.DBName) // assumption is schema has same name as dbname
	}
	return dsn
}

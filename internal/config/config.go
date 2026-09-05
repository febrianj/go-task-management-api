package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
	BcryptCost  int
	LogLevel    string
}

func Load() (*Config, error) {
	c := &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTL:      getEnvDuration("JWTTTL", 24*time.Hour),
		BcryptCost:  getEnvInt("BCRYPT_COST", 12),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}

	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) validate() error {
	var errs []error
	if c.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.JWTSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET is required"))
	} else if len(c.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 32 characters"))
	}
	if c.BcryptCost < 10 || c.BcryptCost > 14 {
		errs = append(errs, errors.New("BCRYPT_COST must be between 10 and 14"))
	}

	return errors.Join(errs...)
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return d
}

var _ = fmt.Sprintf

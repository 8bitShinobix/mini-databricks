package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl              string
	RedisURL           string
	KafkaBrokers       string
	MinioEndpoint      string
	MinioAccessKey     string
	MinioSecretKey     string
	MinioBucket        string
	APIPort            string
	JWTSecret          string
	Env                string
	TaskTimeoutSeconds int
	PythonPath         string
	TaskRunnerPath     string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("no .env file found, reading from environment")
	}

	taskTimeout := 3600
	if v := os.Getenv("TASK_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			taskTimeout = parsed
		}
	}

	return &Config{
		DBUrl:              os.Getenv("DB_URL"),
		RedisURL:           os.Getenv("REDIS_URL"),
		KafkaBrokers:       os.Getenv("KAFKA_BROKERS"),
		MinioEndpoint:      os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey:     os.Getenv("MINIO_ACCESS_KEY"),
		MinioSecretKey:     os.Getenv("MINIO_SECRET_KEY"),
		MinioBucket:        os.Getenv("MINIO_BUCKET"),
		APIPort:            os.Getenv("API_PORT"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		Env:                os.Getenv("ENV"),
		TaskTimeoutSeconds: taskTimeout,
		PythonPath:         os.Getenv("PYTHON_PATH"),
		TaskRunnerPath:     os.Getenv("TASK_RUNNER_PATH"),
	}
}

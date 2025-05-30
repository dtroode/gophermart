package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	RunAddr      string `env:"RUN_ADDRESS"`
	DatabaseDSN  string `env:"DATABASE_URI"`
	AccrualAddr  string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LogLevel     string `env:"LOG_LEVEL"`
	JWTSecretKey string `env:"JWT_SECRET_KEY"`

	ConcurrencyLimit int `env:"CONCURRENCY_LIMIT"`
	QueueSize        int `env:"QUEUE_SIZE"`

	ArgonSalt    string `env:"ARGON_SALT"`
	ArgonTime    int    `env:"ARGON_TIME"`
	ArgonMemory  int    `env:"ARGON_MEMORY"`
	ArgonThreads int    `env:"ARGON_THREADS"`
	ArgonKeyLen  int    `env:"ARGON_KEY_LEN"`
}

func Initialize() (*Config, error) {
	config := new(Config)

	flag.StringVar(&config.RunAddr, "a", ":8089", "(address and) port to run server")
	flag.StringVar(&config.DatabaseDSN, "d", "", "string for connecting to postgres")
	flag.StringVar(&config.AccrualAddr, "r", "", "accrual system address")
	flag.StringVar(&config.LogLevel, "l", "DEBUG", "log level")
	flag.StringVar(&config.JWTSecretKey, "j", "secret", "jwt secret key")

	flag.IntVar(&config.ConcurrencyLimit, "cl", 5, "number of workers in pool")
	flag.IntVar(&config.QueueSize, "qs", 0, "length of queue of jobs")

	flag.StringVar(&config.ArgonSalt, "as", "saltsalt", "argon salt")
	flag.IntVar(&config.ArgonTime, "atime", 1, "argon time parameter")
	flag.IntVar(&config.ArgonMemory, "amem", 47104, "argon memory parameter")
	flag.IntVar(&config.ArgonThreads, "athreads", 1, "argon threads parameter")
	flag.IntVar(&config.ArgonKeyLen, "akeylen", 32, "argon key length parameter")

	flag.Parse()

	err := env.Parse(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

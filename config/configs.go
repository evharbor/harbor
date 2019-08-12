package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

// DBConfig database config struct
type DBConfig struct {
	Alias    string `mapstructure:"alias"`
	Engine   string `mapstructure:"engine"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Charset  string `mapstructure:"charset"`
}

// Config struct
type Config struct {
	Debug     bool       `mapstructure:"debug"`
	SecretKey string     `mapstructure:"secret_key"`
	Databases []DBConfig `mapstructure:"databases"` //database configs
	BaseDir   string
}

var configs Config

// GetConfigs return config struct instance inited with values read from config  file
func GetConfigs() *Config {
	return &configs
}

// LoadConfigFile load config file
func LoadConfigFile(basDir string) {
	path := filepath.Join(basDir, "config")
	v := viper.New()
	v.SetConfigName("config") // name of config file (without extension)
	v.AddConfigPath(path)     // optionally look for config in the working directory
	v.SetConfigType("json")
	// Find and read the config file
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %s ", err))
	}
	if err := v.Unmarshal(&configs); err != nil {
		panic(fmt.Errorf("fatal error config file: %s ", err))
	}
	configs.BaseDir = basDir
}

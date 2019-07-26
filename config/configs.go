package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// DBConfig database config struct
type DBConfig struct {
	Alias    string `json:"alias"`
	Engine   string `json:"engine"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	Charset  string `json:"charset"`
}

// Config struct
type Config struct {
	Debug     bool       `json:"debug"`
	Databases []DBConfig `json:"databases"` //database configs
}

var configs Config

// GetConfigs return config struct instance inited with values read from config  file
func GetConfigs() *Config {
	return &configs
}

func init() {
	v := viper.New()
	v.SetConfigName("config")   // name of config file (without extension)
	v.AddConfigPath("./config") // optionally look for config in the working directory
	v.SetConfigType("json")
	// Find and read the config file
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %s ", err))
	}
	if err := v.Unmarshal(&configs); err != nil {
		panic(fmt.Errorf("fatal error config file: %s ", err))
	}
	// var dbs []DBConfig
	// if err := v.UnmarshalKey("databases", &dbs); err != nil {
	// 	panic(fmt.Errorf("fatal error config file: %s ", err))
	// }
	// configs.Databases = dbs
}

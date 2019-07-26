package database

import (
	"fmt"
	"harbor/config"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var dbConnMap map[string]*gorm.DB

func init() {
	dbConnMap = make(map[string]*gorm.DB)
	configs := config.GetConfigs()
	debug := configs.Debug
	dbs := configs.Databases
	for _, db := range dbs {
		engine := db.Engine
		var url string
		if engine == "mysql" {
			url = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", db.User, db.Password, db.Host, db.Port, db.Name, db.Charset)
		}
		dbConn, errConn := gorm.Open(engine, url)
		if errConn != nil {
			panic("连接数据库失败:" + errConn.Error())
		}
		dbConn.LogMode(debug)
		dbConnMap[db.Alias] = dbConn
	}
}

// GetDBDefault get database connect
func GetDBDefault() *gorm.DB {
	return dbConnMap["default"]
}

// GetDB get database connect
func GetDB(alias string) *gorm.DB {
	if alias == "default" {
		return GetDBDefault()
	}
	db, ok := dbConnMap[alias]
	if !ok {
		panic(fmt.Sprintf("DB with alias ‘%s’ does not exist", alias))
	}
	return db
}

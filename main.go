package main

import (
	"github.com/spf13/viper"
	"goChat/models"
	"goChat/utils"
	"log"
	"time"
)

import (
	"goChat/router"
)

func main() {
	if err := utils.InitConfig(); err != nil {
		log.Fatal(err)
	}
	if err := utils.InitMysql(); err != nil {
		log.Fatal(err)
	}
	if err := utils.InitRedis(); err != nil {
		log.Fatal(err)
	}
	if err := models.Migrate(); err != nil {
		log.Fatal(err)
	}
	models.StartMessageTransport()

	InitTimer()

	r := router.Router()
	if err := r.Run(viper.GetString("server.port")); err != nil {
		log.Fatal(err)
	}
}

//func main() {
//	utils.InitConfig()
//	// Migrate the schema
//	utils.InitMysql()
//	utils.InitRedis()
//
//	utils.Db.AutoMigrate(&models.UserBasic{})
//	// Migrate the schema
//	utils.Db.AutoMigrate(&models.Message{})
//	utils.Db.AutoMigrate(&models.Contact{})
//	utils.Db.AutoMigrate(&models.GroupBasic{})
//	utils.Db.AutoMigrate(&models.Community{})
//
//}

// 初始化定时器
func InitTimer() {
	utils.Timer(time.Duration(viper.GetInt("timeout.DelayHeartbeat"))*time.Second, time.Duration(viper.GetInt("timeout.HeartbeatHz"))*time.Second, models.CleanConnection, "")
}

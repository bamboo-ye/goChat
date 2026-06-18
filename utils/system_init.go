package utils

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"strings"
	"time"
)

var (
	Db  *gorm.DB
	Red *redis.Client
)

func InitConfig() error {
	viper.SetConfigName("app")
	viper.AddConfigPath("config")
	viper.SetEnvPrefix("GOCHAT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("server.port", ":8081")
	viper.SetDefault("redis.addr", "127.0.0.1:6379")
	viper.SetDefault("redis.DB", 0)
	viper.SetDefault("udp.enabled", false)
	viper.SetDefault("ai.provider", "openrouter")
	viper.SetDefault("ai.baseURL", "https://openrouter.ai/api/v1")
	viper.SetDefault("ai.model", "openai/gpt-oss-120b:free")
	viper.SetDefault("ai.httpReferer", "")
	viper.SetDefault("ai.appTitle", "goChat")
	viper.SetDefault("ai.timeoutSeconds", 60)
	viper.SetDefault("ai.historyLimit", 20)
	viper.SetDefault("ai.maxInputCharacters", 4000)
	viper.SetDefault("ai.maxOutputTokens", 800)
	viper.SetDefault("ai.temperature", 0.8)
	viper.SetDefault("ai.dailyMessageLimit", 100)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	return nil
}

func InitMysql() error {
	//自定义日志模板，打印SQL语句
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second, //慢SQL阈值
			LogLevel:      logger.Info, //级别
			Colorful:      true,        //彩色
		},
	)

	dsn := viper.GetString("mysql.dsn")
	if dsn == "" {
		return fmt.Errorf("mysql.dsn is required; set GOCHAT_MYSQL_DSN")
	}
	var err error
	Db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: newLogger})
	if err != nil {
		return fmt.Errorf("connect mysql: %w", err)
	}
	sqlDB, err := Db.DB()
	if err != nil {
		return fmt.Errorf("get mysql connection: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping mysql: %w", err)
	}
	return nil
}

func InitRedis() error {
	Red = redis.NewClient(&redis.Options{
		Addr:         viper.GetString("redis.addr"),
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.DB"),
		PoolSize:     viper.GetInt("redis.poolSize"),
		MinIdleConns: viper.GetInt("redis.minIdleConn"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := Red.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

const (
	PublishKey = "websocket"
)

// Publish 发布消息到Redis
func Publish(ctx context.Context, channel string, msg string) error {
	fmt.Println("Publish 。。。。", msg)
	err := Red.Publish(ctx, channel, msg).Err()
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// Subscribe 订阅Redis 接收消息
func Subscribe(ctx context.Context, channel string) (string, error) {
	sub := Red.Subscribe(ctx, channel)
	fmt.Println("Subscribe 。。。。", ctx)
	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	fmt.Println("Subscribe 。。。。", msg.Payload)
	return msg.Payload, err
}

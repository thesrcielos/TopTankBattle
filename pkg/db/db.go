package db

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var Rdb *redis.Client

func Init() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}
	redisDBConnection()
}

func redisDBConnection() {
	ctx := context.Background()
	useTLS := os.Getenv("REDIS_TLS") == "true"

	var tlsConfig *tls.Config
	if useTLS {
		tlsConfig = &tls.Config{}
	}
	dbNum, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		log.Fatalf("error converting REDIS_DB to int: %v", err)
		dbNum = 0
	}
	Rdb = redis.NewClient(&redis.Options{
		Addr:      os.Getenv("REDIS_ADDR"),
		Username:  os.Getenv("REDIS_USERNAME"),
		Password:  os.Getenv("REDIS_PASSWORD"),
		DB:        dbNum,
		TLSConfig: tlsConfig,
	})

	pong, err := Rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	fmt.Println("Redis connected:", pong)

}

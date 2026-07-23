package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(viper *viper.Viper, log *logrus.Logger) *gorm.DB {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = viper.GetString("database.host")
	}

	username := os.Getenv("DB_USER")
	if username == "" {
		username = viper.GetString("database.username")
	}

	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = viper.GetString("database.password")
	}

	databaseName := os.Getenv("DB_NAME")
	if databaseName == "" {
		databaseName = viper.GetString("database.name")
	}

	portStr := os.Getenv("DB_PORT")
	port := viper.GetInt("database.port")
	if portStr != "" {
		port, _ = strconv.Atoi(portStr)
	}
	idleConnection := viper.GetInt("database.pool.idle")
	maxConnection := viper.GetInt("database.pool.max")
	maxLifeTimeConnection := viper.GetInt("database.pool.lifetime")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, host, port, databaseName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.New(&logrusWriter{Logger: log}, logger.Config{
			SlowThreshold:             time.Second * 5,
			Colorful:                  false,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			LogLevel:                  logger.Info,
		}),
	})

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	connection, err := db.DB()
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	log.Info("Database connected successfully")

	connection.SetMaxIdleConns(idleConnection)
	connection.SetMaxOpenConns(maxConnection)
	connection.SetConnMaxLifetime(time.Second * time.Duration(maxLifeTimeConnection))

	return db
}

type logrusWriter struct {
	Logger *logrus.Logger
}

func (l *logrusWriter) Printf(message string, args ...interface{}) {
	l.Logger.Tracef(message, args...)
}
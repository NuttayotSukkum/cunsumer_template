package main

import (
	"context"
	"fmt"
	"github.com/NuttayotSukkum/batch_consumer/configs"
	"github.com/NuttayotSukkum/batch_consumer/configs/db_config"
	"github.com/NuttayotSukkum/batch_consumer/internals/handlers/rest"
	"github.com/NuttayotSukkum/batch_consumer/internals/pkg/constants"
	"github.com/NuttayotSukkum/batch_consumer/internals/pkg/utils"
	repo "github.com/NuttayotSukkum/batch_consumer/internals/repositories/db"
	"github.com/NuttayotSukkum/batch_consumer/internals/services/clients"
	"github.com/NuttayotSukkum/batch_consumer/internals/services/preprocess"
	"github.com/labstack/echo/v4"
	logger "github.com/labstack/gommon/log"
	"gorm.io/gorm"
	"log"
	"time"
)

func main() {
	ctx := context.Background()
	cfg := configs.InitConfig(ctx)
	logger.Errorf("Config: %+v", cfg)
	location, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Fatalf("Failed to load timezone Asia/Bangkok: %v", err)
	}
	time.Local = location
	dbConnection := initDb(ctx, cfg)
	sqlDb, err := dbConnection.DB()
	if err != nil {
		logger.Errorf("%v Failed to get sql.DB from gorm.DB: %s", ctx, err)
	}

	batchHeaderDBRepo := repo.NewBatchHeaderRepository(dbConnection)
	S3Client, _ := clients.NewS3Client(ctx, cfg.Secrets.AWSSecret.S3.Region, cfg.Secrets.AWSSecret.S3.S3Bucket, cfg.Secrets.AWSSecret.AccessKey, cfg.Secrets.AWSSecret.SecretKey)
	preProcessSvc := preprocess.NewPreProcessService(batchHeaderDBRepo, *S3Client)
	e := rest.InitRouter(ctx, &preProcessSvc)
	defer func() {
		if err := sqlDb.Close(); err != nil {
			log.Fatal(ctx, "Failed to close sql DB:%s", err)
		}
		logger.Info(ctx, "database is closed successfully.")
		utils.DeleteDirectory(constants.DirPath)
	}()

	execute(ctx, cfg, e)
}

func execute(ctx context.Context, cfg *configs.Config, e *echo.Echo) {
	svPort := fmt.Sprintf(":%v", cfg.App.Port)
	if err := e.Start(svPort); err != nil {
		logger.Fatalf("%v:Shutdown the server port:%v", ctx, svPort)
	}

}

func initDb(ctx context.Context, cfg *configs.Config) *gorm.DB {
	host := cfg.Secrets.Host
	port := cfg.Secrets.Port
	username := cfg.Secrets.Username
	password := cfg.Secrets.Password
	dbName := cfg.Secrets.DBName
	return db_config.ACNDBConnector(ctx, host, port, username, password, dbName)
}
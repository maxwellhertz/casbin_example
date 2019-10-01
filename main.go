package main

import (
	"casbin_example/component"
	"casbin_example/handler"
	"casbin_example/middleware"
	"fmt"
	gormadapter "github.com/casbin/gorm-adapter/v2"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
)

var (
	router *gin.Engine
)

func init() {
	// Initialize  casbin adapter
	adapter, err := gormadapter.NewAdapterByDB(component.DB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize casbin adapter: %v", err))
	}

	// Initialize gin router
	router = gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig)) // CORS configuraion
	router.POST("/user/login", handler.Login)
	resource := router.Group("/api")
	resource.Use(middleware.Authenticate())
	{
		resource.GET("/resource", middleware.Authorize("resource", "read", adapter), handler.ReadResource)
		resource.POST("/resource", middleware.Authorize("resource", "write", adapter), handler.WriteResource)
	}
}

func main() {
	defer component.DB.Close()

	err := router.Run(":8081")
	if err != nil {
		panic(fmt.Sprintf("failed to start gin engin: %v", err))
	}
	log.Println("application is now running...")
}

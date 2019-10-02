package main

import (
	"casbin_example/component"
	"casbin_example/handler"
	"casbin_example/middleware"
	"fmt"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
)

var (
	router *gin.Engine
)

func init() {
	// Initialize  casbin adapter
	adapter := fileadapter.NewAdapter("config/basic_policy.csv")

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
	defer func() {
		err := component.DB.Close()
		if err != nil {
			log.Println(fmt.Errorf("failed to close DB connection: %w", err))
		}
	}()

	err := router.Run(":8081")
	if err != nil {
		log.Fatalln(fmt.Errorf("faild to start Gin application: %w", err))
	}
}

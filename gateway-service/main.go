package main

import (
	"gateway-service/handlers"
	"gateway-service/internal/consul"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func main() {
	// this should be the first thing to load all the environment data
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("error loading .env file")
	}

	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "80"
	}
	router := gin.New()

	client, err := consul.CreateConnection()
	if err != nil {
		panic(err)
	}
	h := handlers.NewHandler(client)
	router.Any("/*path", h.APIGateway)

	err = router.Run(":" + appPort)
	if err != nil {
		log.Panic(err)
	}

}

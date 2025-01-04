package handlers

import (
	"github.com/gin-gonic/gin"
	"os"
)

func API() *gin.Engine {
	r := gin.New()
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	prefix := os.Getenv("SERVICE_ENDPOINT_PREFIX")
	if prefix == "" {
		panic("SERVICE_ENDPOINT_PREFIX is not set")
	}
	v1 := r.Group(prefix)
	{
		v1.Use(gin.Logger(), gin.Recovery())
		v1.POST("/signup", Signup)
	}
	return r
}

package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"os"
	"user-service/internal/stores/kafka"
	"user-service/internal/users"
	"user-service/middleware"
)

type Handler struct {
	u        *users.Conf
	validate *validator.Validate
	k        *kafka.Conf
}

func NewHandler(u *users.Conf, k *kafka.Conf) *Handler {
	return &Handler{
		u:        u,
		k:        k,
		validate: validator.New(),
	}
}

func API(u *users.Conf, k *kafka.Conf) *gin.Engine {
	r := gin.New()
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	h := NewHandler(u, k)

	prefix := os.Getenv("SERVICE_ENDPOINT_PREFIX")
	if prefix == "" {
		panic("SERVICE_ENDPOINT_PREFIX is not set")
	}
	v1 := r.Group(prefix)
	v1.Use(middleware.Logger())
	{
		v1.Use(gin.Logger(), gin.Recovery())
		v1.POST("/signup", h.Signup)
		v1.POST("/login", h.UserLogin)
	}
	return r
}

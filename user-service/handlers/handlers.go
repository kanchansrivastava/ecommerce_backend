package handlers

import (
	"os"
	"user-service/internal/auth"
	"user-service/internal/stores/kafka"
	"user-service/internal/users"
	"user-service/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	u        *users.Conf
	validate *validator.Validate
	k        *kafka.Conf
	authKeys *auth.Keys
}

func NewHandler(u *users.Conf, k *kafka.Conf, authKeys *auth.Keys) *Handler {
	return &Handler{
		u:        u,
		k:        k,
		validate: validator.New(),
		authKeys: authKeys,
	}
}

func API(u *users.Conf, k *kafka.Conf, a *auth.Keys) *gin.Engine {
	r := gin.New()
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	h := NewHandler(u, k, a)

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

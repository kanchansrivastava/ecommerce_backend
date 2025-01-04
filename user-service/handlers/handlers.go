package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
	"os"
	"user-service/internal/auth"
	"user-service/internal/stores/kafka"
	"user-service/internal/users"
	"user-service/middleware"
	"user-service/pkg/ctxmanage"
)

type Handler struct {
	u        *users.Conf
	validate *validator.Validate
	k        *kafka.Conf
	a        *auth.Keys
}

func NewHandler(u *users.Conf, a *auth.Keys, k *kafka.Conf) *Handler {
	return &Handler{
		u:        u,
		k:        k,
		a:        a,
		validate: validator.New(),
	}
}

func API(u *users.Conf, a *auth.Keys, k *kafka.Conf) *gin.Engine {
	r := gin.New()
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	h := NewHandler(u, a, k)
	m, err := middleware.NewMid(a)
	if err != nil {
		panic(err)
	}

	prefix := os.Getenv("SERVICE_ENDPOINT_PREFIX")
	if prefix == "" {
		panic("SERVICE_ENDPOINT_PREFIX is not set")
	}
	r.Use(gin.Logger(), gin.Recovery(), middleware.Logger())
	r.GET("/ping", healthCheck)
	v1 := r.Group(prefix)
	{
		v1.POST("/signup", h.Signup)
		v1.POST("/login", h.Login)

		// this middleware would be applied to the handler functions which are after it
		// it would not apply to the previous one
		v1.Use(m.Authentication())
		v1.GET("/check", func(c *gin.Context) {
			claims, err := ctxmanage.GetAuthClaimsFromContext(c.Request.Context())
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"Auth Check": "You are authenticated " + claims.Subject})
		})
		v1.GET("/stripe", h.GetStripeDetails)

	}
	return r
}

func healthCheck(c *gin.Context) {
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	fmt.Println("healthCheck handler ", traceId)
	//JSON serializes the given struct as JSON into the response body. It also sets the Content-Type as "application/json".
	c.JSON(http.StatusOK, gin.H{"status": "ok"})

}

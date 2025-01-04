package handlers

import (
	"net/http"
	"os"
	"product-service/internal/products"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	p        *products.Conf
	validate *validator.Validate
}

func NewHandler(p *products.Conf) *Handler {
	return &Handler{
		p:        p,
		validate: validator.New(),
	}
}

func API(p *products.Conf) *gin.Engine {
	r := gin.New()
	mode := os.Getenv("GIN_MODE")
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	h := NewHandler(p)
	prefix := os.Getenv("SERVICE_ENDPOINT_PREFIX")
	if prefix == "" {
		panic("SERVICE_ENDPOINT_PREFIX is not set")
	}
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ping successful"})
	})
	v1 := r.Group(prefix)
	{
		v1.POST("/create", h.CreateProduct)
		v1.GET("/stock/:productID", h.GetProductByID)
	}
	return r
}

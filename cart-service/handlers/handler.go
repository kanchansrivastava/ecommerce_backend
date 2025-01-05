package handlers

import (
	"cart-service/internal/auth"
	"cart-service/internal/cart"
	"cart-service/middleware"
	"cart-service/pkg/ctxmanage"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
)

type Handler struct {
	cConf  cart.Conf
	a      *auth.Keys
	client *consulapi.Client
}

func NewHandler(cConf cart.Conf, a *auth.Keys, client *consulapi.Client) *Handler {
	return &Handler{
		cConf:  cConf,
		a:      a,
		client: client,
	}
}

func API(endpointPrefix string, a *auth.Keys, client *consulapi.Client, cConf cart.Conf) *gin.Engine {

	r := gin.New()
	mode := os.Getenv("GIN_MODE")
	if mode == gin.ReleaseMode {
		gin.SetMode(mode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	m := middleware.NewMid(a)
	//s := models.NewStore(&c)
	h := NewHandler(cConf, a, client)
	//apply middleware to all the endpoints using r.Use
	r.Use(middleware.Logger(), gin.Recovery())
	r.GET("/ping", healthCheck)
	v1 := r.Group(endpointPrefix)
	{
		v1.Use(middleware.Logger())
		v1.Use(m.Authentication())
		v1.POST("/add-item", m.Authorize(h.AddToCart, auth.RoleUser))
		v1.GET("/items", m.Authorize(h.GetActiveCartItems, auth.RoleUser))
	}

	return r
}

func healthCheck(c *gin.Context) {
	traceId := ctxmanage.GetTraceIdOfRequest(c)

	fmt.Println("healthCheck handler ", traceId)
	//JSON serializes the given struct as JSON into the response body. It also sets the Content-Type as "application/json".
	c.JSON(http.StatusOK, gin.H{"status": "ok"})

}

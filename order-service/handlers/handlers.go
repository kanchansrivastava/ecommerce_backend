package handlers

import (
	pb "order-service/gen/proto"
	"order-service/internal/auth"
	"order-service/internal/orders"
	"order-service/internal/stores/kafka"
	"order-service/middleware"
	"os"

	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
)

type Handler struct {
	client      *consulapi.Client
	o           *orders.Conf
	k           *kafka.Conf
	protoClient pb.CartItemServiceClient
}

func NewHandler(client *consulapi.Client, o *orders.Conf, k *kafka.Conf, protoClient pb.CartItemServiceClient) *Handler {
	return &Handler{
		client:      client,
		o:           o,
		k:           k,
		protoClient: protoClient,
	}
}

func API(endpointPrefix string, k *auth.Keys, client *consulapi.Client, o *orders.Conf,
	kafkaConf *kafka.Conf, protoClient pb.CartItemServiceClient) *gin.Engine {
	r := gin.New()
	mode := os.Getenv("GIN_MODE")
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	m, err := middleware.NewMid(k)
	if err != nil {
		panic(err)
	}

	h := NewHandler(client, o, kafkaConf, protoClient)
	r.Use(middleware.Logger(), gin.Recovery())

	r.GET("/ping", HealthCheck)
	v1 := r.Group(endpointPrefix)
	{
		v1.POST("/webhook", h.Webhook)
		v1.Use(m.Authentication())
		// v1.POST("/checkout", h.Checkout)
		v1.GET("/ping", HealthCheck)

	}

	v2 := r.Group(endpointPrefix)
	{
		v2.Use(m.Authentication())
		v2.POST("/checkout", h.CheckoutV2)
	}
	return r
}

func HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

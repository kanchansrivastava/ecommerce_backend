package handlers

import (
	"context"
	"log"
	"net/http"
	"user-service/internal/stores/postgres"
	"user-service/internal/users"

	"github.com/gin-gonic/gin"
)

func Signup(c *gin.Context) {
	var newUser users.NewUser

	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db, err := postgres.OpenDB()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	conf, err := users.NewConf(db)
	ctx := context.Background()

	user, err := conf.InsertUser(ctx, newUser)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User created successfully",
		"user_id": user.ID,
	})
}

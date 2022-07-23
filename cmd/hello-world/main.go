package main

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/go-test-task/internal"
)

func main() {
	log.Info("Thank you for the opportunity to do this task!")
	router := gin.Default()
	// creating separate router for v1
	v1 := router.Group("/v1")
	// adding middleware to v1 router
	v1.Use(internal.Auth())
	// adding handlers to v1 router
	v1.GET("/", internal.V1HelloWorld)
	// starting server at port 8000
	log.Info("Starting server at port 8000...")
	router.Run(":8000")
}

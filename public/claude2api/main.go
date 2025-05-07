package main

import (
	"claude2api/config"
	"claude2api/router"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	// Load configuration

	// Setup all routes
	router.SetupRoutes(r)

	// Run the server on 0.0.0.0:8080
	r.Run(config.ConfigInstance.Address)
}

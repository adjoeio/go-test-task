package internal

import (
	"github.com/gin-gonic/gin"
)

func V1HelloWorld(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Hello, World!",
	})
}

package internal

import (
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info("Checking auth...")
		SecretPublicKey := "key"
		token := c.GetHeader("test")
		if token != SecretPublicKey {
			log.Info("Unauthorized")
			c.AbortWithStatus(401)
		} else {
			log.Info("Authorized")
			c.Next()
		}
	}
}

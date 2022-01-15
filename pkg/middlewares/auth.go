package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tkestack.io/kstone/pkg/authprovider"
)

// Auth authenticates requests
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		rsp, ok, err := authprovider.MiddlewareRequest(c)
		if !ok || err != nil {
			c.JSON(http.StatusUnauthorized, *rsp)
			c.Abort()
		}
		c.Next()
	}
}

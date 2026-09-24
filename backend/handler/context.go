package handler

import (
	"backend/service"

	"github.com/gin-gonic/gin"
)

const authenticatedUserContextKey = "authenticated_user"

func SetAuthenticatedUser(c *gin.Context, user service.AuthenticatedUser) {
	c.Set(authenticatedUserContextKey, user)
}

func AuthenticatedUser(c *gin.Context) (service.AuthenticatedUser, bool) {
	value, exists := c.Get(authenticatedUserContextKey)
	if !exists {
		return service.AuthenticatedUser{}, false
	}

	user, ok := value.(service.AuthenticatedUser)
	return user, ok
}

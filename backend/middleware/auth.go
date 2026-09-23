package middleware

import (
	"context"
	"errors"

	"backend/api"
	"backend/handler"
	"backend/service"

	"github.com/gin-gonic/gin"
)

type AuthService interface {
	Authenticate(context.Context, string) (service.AuthenticatedUser, error)
}

func RequireAuthentication(authService AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(handler.SessionCookieName)
		if err != nil {
			api.Unauthenticated(c)
			return
		}

		user, err := authService.Authenticate(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, service.ErrUnauthenticated) {
				api.Unauthenticated(c)
				return
			}
			api.InternalServerError(c)
			return
		}

		handler.SetAuthenticatedUser(c, user)
		c.Next()
	}
}

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		user, ok := handler.AuthenticatedUser(c)
		if !ok {
			api.Unauthenticated(c)
			return
		}
		if _, ok := allowed[user.Role]; !ok {
			api.Forbidden(c)
			return
		}

		c.Next()
	}
}

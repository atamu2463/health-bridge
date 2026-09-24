package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/api"
	"backend/service"

	"github.com/gin-gonic/gin"
)

const SessionCookieName = "health_bridge_session"

type AuthService interface {
	Login(context.Context, string, string) (service.LoginResult, error)
	Logout(context.Context, string) error
}

type AuthHandler struct {
	authService  AuthService
	cookieSecure bool
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AuthenticatedUserResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	ManagerID *string `json:"managerId"`
}

type authResponse struct {
	User AuthenticatedUserResponse `json:"user"`
}

func NewAuthHandler(authService AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{authService: authService, cookieSecure: cookieSecure}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.InvalidRequest(c)
		return
	}

	email := strings.ToLower(strings.TrimSpace(request.Email))
	if email == "" || request.Password == "" {
		api.InvalidRequest(c)
		return
	}

	result, err := h.authService.Login(c.Request.Context(), email, request.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			api.InvalidCredentials(c)
			return
		}
		api.InternalServerError(c)
		return
	}

	h.setSessionCookie(c, result.Token, result.ExpiresAt)
	c.JSON(http.StatusOK, authResponse{User: authenticatedUserResponse(result.User)})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := AuthenticatedUser(c)
	if !ok {
		api.InternalServerError(c)
		return
	}

	c.JSON(http.StatusOK, authResponse{User: authenticatedUserResponse(user)})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.expireSessionCookie(c)

	token, err := c.Cookie(SessionCookieName)
	if err == nil {
		if err := h.authService.Logout(c.Request.Context(), token); err != nil {
			api.InternalServerError(c)
			return
		}
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/api",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: h.sameSite(),
	})
}

func (h *AuthHandler) expireSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/api",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: h.sameSite(),
	})
}

func (h *AuthHandler) sameSite() http.SameSite {
	if h.cookieSecure {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func authenticatedUserResponse(user service.AuthenticatedUser) AuthenticatedUserResponse {
	var managerID *string
	if user.ManagerID != nil {
		value := strconv.FormatUint(uint64(*user.ManagerID), 10)
		managerID = &value
	}

	return AuthenticatedUserResponse{
		ID:        strconv.FormatUint(uint64(user.ID), 10),
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		ManagerID: managerID,
	}
}

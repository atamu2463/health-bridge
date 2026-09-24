package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	ErrorCodeInvalidRequest      = "invalid_request"
	ErrorCodeInvalidCredentials  = "invalid_credentials"
	ErrorCodeUnauthenticated     = "unauthenticated"
	ErrorCodeForbidden           = "forbidden"
	ErrorCodeOriginNotAllowed    = "origin_not_allowed"
	ErrorCodeTooManyRequests     = "too_many_requests"
	ErrorCodeNotFound            = "not_found"
	ErrorCodeInternalServerError = "internal_server_error"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func AbortWithError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

func InvalidRequest(c *gin.Context) {
	AbortWithError(c, http.StatusBadRequest, ErrorCodeInvalidRequest, "リクエスト内容を確認してください")
}

func InvalidCredentials(c *gin.Context) {
	AbortWithError(c, http.StatusUnauthorized, ErrorCodeInvalidCredentials, "メールアドレスまたはパスワードが正しくありません")
}

func Unauthenticated(c *gin.Context) {
	AbortWithError(c, http.StatusUnauthorized, ErrorCodeUnauthenticated, "認証が必要です")
}

func Forbidden(c *gin.Context) {
	AbortWithError(c, http.StatusForbidden, ErrorCodeForbidden, "この操作を行う権限がありません")
}

func OriginNotAllowed(c *gin.Context) {
	AbortWithError(c, http.StatusForbidden, ErrorCodeOriginNotAllowed, "許可されていないOriginです")
}

func TooManyRequests(c *gin.Context) {
	AbortWithError(c, http.StatusTooManyRequests, ErrorCodeTooManyRequests, "しばらく待ってから再度お試しください")
}

func NotFound(c *gin.Context) {
	AbortWithError(c, http.StatusNotFound, ErrorCodeNotFound, "リソースが見つかりません")
}

func InternalServerError(c *gin.Context) {
	AbortWithError(c, http.StatusInternalServerError, ErrorCodeInternalServerError, "サーバー内部でエラーが発生しました")
}

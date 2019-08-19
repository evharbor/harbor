package middlewares

import (
	"errors"
	"harbor/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthTokenMiddlewareFunc returns a Basic HTTP Authorization middleware.
// If the realm is empty, "Authorization Required" will be used by default.
func AuthTokenMiddlewareFunc() gin.HandlerFunc {

	return func(ctx *gin.Context) {

		// already authenticated，return
		if u, exists := ctx.Get(AuthUserKey); exists && u != nil {
			return
		}

		auth, err := tokenParseHeader(ctx.GetHeader("Authorization"))
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
			return
		} else if auth == "" {
			return
		}

		user := getUserByToken(auth)
		if user == nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, "authorization failed")
			return
		}

		if !user.IsActived() {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, "authorization failed")
			return
		}

		// The user credentials was found, set user object to key AuthUserKey in this context
		ctx.Set(AuthUserKey, user)
	}
}

// tokenParseHeader try to get "xxx" from auth header "Token xxx"
//return:
//		"xx", nil: success
//		"", nil: is not basic auth header
//		"", err: error
func tokenParseHeader(authHeader string) (string, error) {

	auth := strings.Split(authHeader, " ")
	if strings.ToLower(auth[0]) != "token" {
		return "", nil
	}

	l := len(auth)
	if l == 1 {
		return "", errors.New("invalid token header, no credentials provided")
	} else if l > 2 {
		return "", errors.New("invalid token header, credentials string should not contain spaces")
	}

	return auth[1], nil
}

func getUserByToken(token string) *models.UserProfile {

	m := models.NewTokenManager(nil)
	t, err := m.GetTokenWithUser(token)
	if err != nil {
		return nil
	}
	if t != nil{
		return t.User
	}
	return nil
}

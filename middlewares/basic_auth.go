package middlewares

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// IBasicAuth basic auth interface
type IBasicAuth interface {
	IsActived() bool
	CheckPassword(string) bool
	UsernameColumnName() string
}

// AuthUserKey is the cookie name for user credential in basic auth
const AuthUserKey string = "user"

// BasicAuth returns a Basic HTTP Authorization middleware. It takes as argument a map[string]string where
// the key is the user name and the value is the password.
func BasicAuth(db *gorm.DB, user interface{}) gin.HandlerFunc {
	return BasicAuthForRealm(db, user, "")
}

// BasicAuthForRealm returns a Basic HTTP Authorization middleware.
// If the realm is empty, "Authorization Required" will be used by default.
func BasicAuthForRealm(db *gorm.DB, user interface{}, realm string) gin.HandlerFunc {
	if realm == "" {
		realm = "Authorization Required"
	}
	realm = "Basic realm=" + strconv.Quote(realm)
	return func(ctx *gin.Context) {

		auth, err := basicParseHeader(ctx.GetHeader("Authorization"))
		if err != nil {
			ctx.Header("WWW-Authenticate", realm)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, err.Error())
			return
		} else if auth == "" {
			return
		}

		username, password, ok := basicCredential(auth)
		if !ok {
			ctx.Header("WWW-Authenticate", realm)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, "Invalid basic header,Credentials not correctly base64 encoded.")
			return
		}

		nUser := CopyEmptyStruct(user)
		err = authenticate(db, username, password, nUser)
		if err != nil {
			// Credentials doesn't match, we return 401 and abort handlers chain.
			ctx.Header("WWW-Authenticate", realm)
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// The user credentials was found, set user object to key AuthUserKey in this context
		ctx.Set(AuthUserKey, nUser)
	}
}

// basicParseHeader try to get "xxx" from auth header "Basic xxx"
//return:
//		"xx", nil: success
//		"", nil: is not basic auth header
//		"", err: error
func basicParseHeader(authHeader string) (string, error) {

	auth := strings.Split(authHeader, " ")
	if strings.ToLower(auth[0]) != "basic" {
		return "", nil
	}

	l := len(auth)
	if l == 1 {
		return "", errors.New("invalid basic header, no credentials provided")
	} else if l > 2 {
		return "", errors.New("invalid basic header, credentials string should not contain spaces")
	}

	return auth[1], nil
}

func basicCredential(auth string) (username, password string, ok bool) {

	value, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return
	}
	cred := string(value)
	a := strings.Split(cred, ":")
	if len(a) != 2 {
		return
	}
	username = a[0]
	password = a[1]
	ok = true
	return
}

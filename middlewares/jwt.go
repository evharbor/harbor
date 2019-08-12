package middlewares

import (
	"harbor/config"
	"harbor/database"
	"harbor/middlewares/jwt"
	"harbor/models"
	"time"

	"github.com/gin-gonic/gin"
)

var identityKey = "user"

// JWTLoginForm jwt login form struct
type JWTLoginForm struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// JWTRefreshForm jwt login form struct
type JWTRefreshForm struct {
	Token string `json:"token" form:"token"`
}

// client post username and password to authenticate, return *UserProfile if success
func jwtAuthenticator(c *gin.Context) (interface{}, error) {
	var loginForm JWTLoginForm
	if err := c.ShouldBind(&loginForm); err != nil {
		return "", jwt.ErrMissingLoginValues
	}
	username := loginForm.Username
	password := loginForm.Password

	user := &models.UserProfile{}
	db := database.GetDB("default")
	err := authenticate(db, username, password, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Which user information is used to generate payload
func jwtPayloadFunc(data interface{}) jwt.MapClaims {
	if user, ok := data.(*models.UserProfile); ok {
		return jwt.MapClaims{
			"id":           user.ID,
			"username":     user.Username,
			"is_superuser": user.IsSuperUser,
		}
	}
	return jwt.MapClaims{}
}

// JWTAuthMiddleware return jwt auth middleware
func JWTAuthMiddleware() (*jwt.GinJWTMiddleware, error) {

	configs := config.GetConfigs()
	secretKey := configs.SecretKey

	// the jwt middleware
	return jwt.New(&jwt.GinJWTMiddleware{
		Realm:       "",
		Key:         []byte(secretKey),
		Timeout:     24 * time.Hour,
		MaxRefresh:  7 * 24 * time.Hour,
		IdentityKey: identityKey,
		PayloadFunc: jwtPayloadFunc,
		IdentityHandler: func(c *gin.Context) interface{} {
			claims := jwt.ExtractClaims(c)
			return &models.UserProfile{
				ID:          uint(claims["id"].(float64)),
				Username:    claims["username"].(string),
				IsSuperUser: claims["is_superuser"].(bool),
			}
		},
		Authenticator: jwtAuthenticator,
		Authorizator: func(data interface{}, c *gin.Context) bool {
			if v, ok := data.(*models.UserProfile); ok && (v.ID > 0) {
				return true
			}

			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":      code,
				"code_text": message,
			})
		},
		// TokenLookup is a string in the form of "<source>:<name>" that is used
		// to extract token from the request.
		// Optional. Default value "header:Authorization".
		// Possible values:
		// - "header:<name>"
		// - "query:<name>"
		// - "cookie:<name>"
		// - "param:<name>"
		TokenLookup: "header: Authorization",
		// TokenLookup: "header: Authorization, query: jwt, cookie: jwt",
		// TokenLookup: "query:token",
		// TokenLookup: "cookie:token",

		// TokenHeadName is a string in the header.
		TokenHeadName: "JWT",

		// TimeFunc provides the current time. You can override it to use another time value. This is useful for testing or if your server uses a different time zone than your tokens.
		TimeFunc:                time.Now,
		DisabledAbort:           true, //
		DoNothingIfNotJWTHeader: true,
	})
}

// jwtLoginHandler API document
// @Description jwt login handler
// @Tags jwt
// @Accept  json
// @Produce  json
// @Param   data body middlewares.JWTLoginForm true "auth info"
// @Success 200 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/jwt-token/ [post]
func jwtLoginHandler() {}

// jwtRefreshHandler API document
// @Description jwt refresh handler
// @Tags jwt
// @Accept  json
// @Accept  application/x-www-form-urlencoded
// @Produce json
// @Param   data body middlewares.JWTRefreshForm true "jwt"
// @Success 200 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/jwt-token-refresh/ [post]
func jwtRefreshHandler() {}

// UserFromJWTPayload return user or nil
func UserFromJWTPayload(ctx *gin.Context) *models.UserProfile {

	if payload, exists := ctx.Get("JWT_PAYLOAD"); exists {
		if info, ok := payload.(jwt.MapClaims); ok {
			id, _ := info["id"].(uint)
			username, _ := info["username"].(string)
			return &models.UserProfile{
				ID:       id,
				Username: username,
			}
		}
	}
	return nil
}

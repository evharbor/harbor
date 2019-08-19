package routes

import (
	ctls "harbor/controllers"
	"harbor/middlewares"

	"github.com/gin-gonic/gin"
)

// Urls config routers
func Urls(ng *gin.Engine) {

	jwtAuth, err := middlewares.JWTAuthMiddleware()
	if err != nil {
		panic("JWT Error: jwt middleware")
	}

	ng.GET("/docs/", ctls.Docs)
	ng.GET("/user/register/", ctls.UserRegister)
	ng.POST("/user/register/", ctls.UserRegister)
	ng.POST("/api/v1/jwt-token/", jwtAuth.LoginHandler)
	ng.POST("/api/v1/jwt-token-refresh/", jwtAuth.RefreshHandler)
	v1 := ng.Group("/api/v1", jwtAuth.MiddlewareFunc(),middlewares.AuthTokenMiddlewareFunc())
	{
		v1.Any("/users/", ctls.NewUserController().Init().Dispatch)
		v1.Any("/users/:id/", ctls.NewUserDetailController().Init().Dispatch)
		v1.Any("/obj/:bucketname/*objpath", ctls.NewObjController().Init().Dispatch)
		v1.Any("/buckets/", ctls.NewBucketController().Init().Dispatch)
		v1.Any("/buckets/:id/", ctls.NewBucketDetailController().Init().Dispatch)
		v1.Any("/dir/:bucketname/*dirpath", ctls.NewDirController().Init().Dispatch)
		v1.Any("/metadata/:bucketname/*path", ctls.NewMetadataController().Init().Dispatch)
		v1.Any("/move/:bucketname/*objpath", ctls.NewMoveController().Init().Dispatch)
		v1.Any("/auth-token/", ctls.NewTokenController().Init().Dispatch)
	}
	obs := ng.Group("obs", jwtAuth.MiddlewareFunc())
	{
		obs.GET("/:bucketname/*objpath", ctls.NewDownloadController().Init().Dispatch)
	}
}

package controllers

import (
	"harbor/database"
	"harbor/middlewares"
	"harbor/models"
	"strings"

	"github.com/gin-gonic/gin"
)

// TokenController token控制器结构
type TokenController struct {
	Controller
}

// NewTokenController new controller
func NewTokenController() *TokenController {
	return &TokenController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *TokenController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// GetPermissions return permission
func (ctl TokenController) GetPermissions(ctx *gin.Context) []PermissionFunc {

	method := strings.ToUpper(ctx.Request.Method)
	switch method {
	case "GET", "PUT":
		return []PermissionFunc{IsAuthenticatedUser}
	default:
		return []PermissionFunc{}
	}
}

// Get handler for get method
// @Summary 获取当前用户的token
// @Description 获取当前用户的token，需要通过身份认证权限
// @Tags auth token
// @Accept  json
// @Produce json
// @Success 200 {object} controllers.TokenJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/auth-token/ [get]
func (ctl TokenController) Get(ctx *gin.Context) {

	tm := models.NewTokenManager(ctl.user)
	token, _, err := tm.GetOrCreateToken()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}

	ctx.JSON(200, &TokenJSON{
		BaseJSON: *BaseJSONResponse(200, "ok"),
		Token:    token,
	})
}

// TokenLoginForm token login form struct
type TokenLoginForm struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// TokenJSON token json struct
type TokenJSON struct {
	BaseJSON
	Token *models.Token `json:"token"`
}

// Post controller
// @Summary 身份验证并返回一个token
// @Description 身份验证并返回一个token
// @Description 令牌应包含在AuthorizationHTTP标头中。密钥应以字符串文字“Token”为前缀，空格分隔两个字符串。
// @Description 例如Authorization: Token 9944b09199c62bcf9418ad846dd0e4bbdfc6ee4b；
// @Description 此外，可选query参数,“new”，?new=true用于刷新生成一个新token；
// @Tags auth token
// @Accept  json
// @Produce  json
// @Param   new query bool false "new token"
// @Param   data body controllers.TokenLoginForm true "auth data"
// @Success 201 {object} controllers.TokenJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Router /api/v1/auth-token/ [post]
func (ctl TokenController) Post(ctx *gin.Context) {

	loginForm := TokenLoginForm{}
	if err := ctx.ShouldBind(&loginForm); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	username := loginForm.Username
	password := loginForm.Password

	user := &models.UserProfile{}
	db := database.GetDB("default")
	err := middlewares.Authenticate(db, username, password, user)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}

	newOne, err := GetBoolParamOrDefault(ctx, "new", false)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, "invalid query param new"))
		return
	}
	tm := models.NewTokenManager(user)
	token, created, err := tm.GetOrCreateToken()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}
	if newOne && !created {
		tm.BeginTransaction()
		if err := tm.DeleteToken(token); err != nil {
			tm.RollbackTransaction()
			ctx.JSON(500, BaseJSONResponse(500, err.Error()))
			return
		}
		token = models.NewToken(user)
		if err := tm.CreateToken(token); err != nil {
			tm.RollbackTransaction()
			ctx.JSON(500, BaseJSONResponse(500, err.Error()))
			return
		}
		if err := tm.CommitTransaction(); err != nil {
			tm.RollbackTransaction()
			ctx.JSON(500, BaseJSONResponse(500, err.Error()))
			return
		}
	}

	ctx.JSON(201, &TokenJSON{
		BaseJSON: *BaseJSONResponse(201, "ok"),
		Token:    token,
	})
}

// Put controller
// @Summary 刷新当前用户的token，旧token失效
// @Description 刷新当前用户的token，旧token失效，需要通过身份认证权限
// @Tags auth token
// @Accept  json
// @Produce  json
// @Success 200 {object} controllers.TokenJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/auth-token/ [put]
func (ctl TokenController) Put(ctx *gin.Context) {

	user := ctl.user
	tm := models.NewTokenManager(user)
	token, created, err := tm.GetOrCreateToken()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}
	if !created {
		tm.BeginTransaction()
		if err := tm.DeleteToken(token); err != nil {
			tm.RollbackTransaction()
			ctx.JSON(500, BaseJSONResponse(500, err.Error()))
			return
		}
		token = models.NewToken(user)
		if err := tm.CreateToken(token); err != nil {
			tm.RollbackTransaction()
			ctx.JSON(500, BaseJSONResponse(500, err.Error()))
			return
		}
		if err := tm.CommitTransaction(); err != nil {
			tm.RollbackTransaction()
			ctx.JSON(500, BaseJSONResponse(500, err.Error()))
			return
		}
	}

	ctx.JSON(200, &TokenJSON{
		BaseJSON: *BaseJSONResponse(200, "ok"),
		Token:    token,
	})
}

package controllers

import (
	"fmt"
	"harbor/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PermissionFunc permission func
type PermissionFunc func(user *models.UserProfile) bool

// ControllerInterface 控制器接口
type ControllerInterface interface {
	Init() ControllerInterface
	Dispatch(ctx *gin.Context)
	GetPermissions(ctx *gin.Context) []PermissionFunc
	Get(ctx *gin.Context)
	Post(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Put(ctx *gin.Context)
	Head(ctx *gin.Context)
	Patch(ctx *gin.Context)
	Options(ctx *gin.Context)
}

// Controller Base controller
// 路由相同请求方法不同时，可以使用Controller简化路由的配置
// 使用：子类化Controller,重写Init()和对应请求方法处理函数
type Controller struct {
	this ControllerInterface // 指向子类的接口，通过其调用子类的方法
}

// Init 初始化this，子类要重写此方法
func (ctl *Controller) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// Get adds a request function to handle GET request.
func (ctl Controller) Get(ctx *gin.Context) {

	MethodNotAllowedJSON(ctx)
}

// Post adds a request function to handle POST request.
func (ctl Controller) Post(ctx *gin.Context) {

	MethodNotAllowedJSON(ctx)
}

// Delete adds a request function to handle DELETE request.
func (ctl Controller) Delete(ctx *gin.Context) {

	MethodNotAllowedJSON(ctx)
}

// Put adds a request function to handle PUT request.
func (ctl Controller) Put(ctx *gin.Context) {

	MethodNotAllowedJSON(ctx)
}

// Head adds a request function to handle HEAD request.
func (ctl Controller) Head(ctx *gin.Context) {

	MethodNotAllowedJSON(ctx)
}

// Patch adds a request function to handle PATCH request.
func (ctl Controller) Patch(ctx *gin.Context) {

	MethodNotAllowedJSON(ctx)
}

// Options adds a request function to handle OPTIONS request.
func (ctl Controller) Options(ctx *gin.Context) {

	MethodNotAllowedJSON(ctx)
}

// GetPermissions return permission
// 可以重写此方法，自定义配置权限
func (ctl Controller) GetPermissions(ctx *gin.Context) []PermissionFunc {

	return []PermissionFunc{}
}

// HasPermission check whether current user has permission
func (ctl Controller) HasPermission(ctx *gin.Context) bool {

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return false
	}
	perms := ctl.this.GetPermissions(ctx)
	for _, f := range perms {
		if !f(user) {
			return false
		}
	}
	return true
}

// Dispatch by request method dispatch it's controller
func (ctl Controller) Dispatch(ctx *gin.Context) {
	if ctl.this == nil {
		ctl.Init()
		fmt.Println("You must overrite Init method.")
	}
	method := strings.ToUpper(ctx.Request.Method)
	switch method {
	case "GET":
		ctl.this.Get(ctx)
	case "POST":
		ctl.this.Post(ctx)
	case "PUT":
		ctl.this.Put(ctx)
	case "DELETE":
		ctl.this.Delete(ctx)
	case "PATCH":
		ctl.this.Patch(ctx)
	default:
		MethodNotAllowedJSON(ctx)
	}
}

// MethodNotAllowedJSON json response when request not allowed method
func MethodNotAllowedJSON(ctx *gin.Context) {
	ctx.JSON(200, BaseJSONResponse(http.StatusMethodNotAllowed, "Method Not Allowed"))
}

// BaseJSON 基本json格式结构
type BaseJSON struct {
	Code     uint   `json:"code"`
	CodeText string `json:"code_text"`
}

// BaseJSONResponse 构造一个基本json格式结构对象
func BaseJSONResponse(statusCode uint, codeText string) *BaseJSON {
	return &BaseJSON{
		Code:     statusCode,
		CodeText: codeText,
	}
}

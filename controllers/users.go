package controllers

import (
	"fmt"
	"harbor/database"
	"harbor/middlewares"
	"harbor/models"
	"harbor/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UserController 用户控制器结构
type UserController struct {
	Controller
}

// UserListJSON 用户列表信息结构
type UserListJSON struct {
	BaseJSON
	Count   uint
	Next    string
	Privous string
	Results []models.UserProfile
}

// NewUserController new controller
func NewUserController() *UserController {
	return &UserController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *UserController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// Get handler for get method
// @Description 获取用户列表页
// @Tags user 用户
// @Accept  json
// @Produce  json
// @Param   offset     query    int     true        "The initial index from which to return the results"
// @Param   limit      query    int     true        "Number of results to return per page"
// @Success 200 {object} controllers.UserListJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/users/ [get]
func (ctl UserController) Get(ctx *gin.Context) {

	user, exists := ctx.Get(middlewares.AuthUserKey)
	if exists {
		u, ok := user.(*models.UserProfile)
		if ok {
			fmt.Println(u)
		}
	}

	db := database.GetDBDefault()
	paginater := utils.NewOptimizedLimitOffsetPagination()
	if err := paginater.PrePaginate(ctx); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	var users []models.UserProfile
	tableName := models.UserProfile{}.TableName()
	dbQuery := db.Table(tableName).Order("id desc")
	if err := paginater.PaginateDBQuery(&users, dbQuery); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}

	bj := BaseJSONResponse(200, "ok")
	data := UserListJSON{
		BaseJSON: *bj,
		Count:    uint(paginater.GetCount()),
		Results:  users,
		Next:     paginater.GetNextURL(),
		Privous:  paginater.GetPreviousURL(),
	}
	ctx.JSON(200, data)
}

// Post controller
// @Description 创建用户
// @Tags user 用户
// @Accept  json
// @Produce  json
// @Param   user     body    models.UserProfile     true        "用户名"
// @Success 200 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/users/ [post]
func (ctl UserController) Post(ctx *gin.Context) {

	user := models.UserProfile{}
	err := ctx.ShouldBind(&user)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	u := models.UserProfile{}
	db := database.GetDBDefault()
	if db.Where("username = ?", user.Username).First(&u).RowsAffected > 0 {
		ctx.JSON(400, BaseJSONResponse(400, "用户已存在"))
		return
	}

	user.DateJoined = time.Now()
	if r := db.Create(&user); r.RowsAffected == 1 && r.Error == nil {
		ctx.JSON(201, BaseJSONResponse(201, "用户创建成功"))
		return
	}
	ctx.JSON(200, BaseJSONResponse(200, "用户创建失败"))
}

// UserRegister 注册用户
// @Description 注册用户
// @Tags Register 注册
// @Accept  json
// @Produce  json
// @Param   some_id     path    string     true        "Some ID"
// @Param   offset     query    int     true        "The initial index from which to return the results"
// @Param   limit      query    int     true        "Number of results to return per page"
// @Success 200 {object} controllers.BaseJSON "ok"
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Router /user/register/ [post]
func UserRegister(ctx *gin.Context) {

	if strings.ToUpper(ctx.Request.Method) == "GET" {
		ctx.HTML(http.StatusOK, "users/register.tmpl", gin.H{})
	}

	username := ctx.PostForm("username")
	password := ctx.PostForm("password")
	ctx.JSON(200, gin.H{
		"username": username,
		"password": password,
	})
}

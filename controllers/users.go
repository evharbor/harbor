package controllers

import (
	"harbor/database"
	"harbor/models"
	"harbor/utils"
	"net/http"
	"strings"

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

// GetPermissions return permission
func (ctl UserController) GetPermissions(ctx *gin.Context) []PermissionFunc {

	method := strings.ToUpper(ctx.Request.Method)
	switch method {
	case "GET", "DELETE":
		return []PermissionFunc{IsSuperUser}
	default:
		return []PermissionFunc{}
	}
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

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	if !ctl.HasPermission(ctx) {
		ctx.JSON(403, BaseJSONResponse(403, "forbidded"))
		return
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

// UserPostForm post form
type UserPostForm struct {
	Username  string `form:"username" json:"username" binding:"max=100,email,required"`
	Password  string `form:"password" json:"password" binding:"min=8,max=128,required"`
	FirstName string `form:"first_name" json:"first_name,omitempty" binding:"max=30"`
	LastName  string `form:"last_name" json:"last_name,omitempty" binding:"max=30"`
	Company   string `form:"company" json:"company,omitempty" binding:"max=255"`
	Telephone string `form:"telephone" json:"telephone,omitempty" binding:"max=11"`
}

func (f *UserPostForm) isValid(ctx *gin.Context) error {

	if err := ctx.ShouldBind(f); err != nil {
		return err
	}
	return f.validate()
}

func (f UserPostForm) validate() error {

	return nil
}

// Post controller
// @Description 创建用户
// @Tags user 用户
// @Accept  json
// @Produce  json
// @Param   user     body    controllers.UserPostForm     true        "用户名"
// @Success 201 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/users/ [post]
func (ctl UserController) Post(ctx *gin.Context) {

	form := UserPostForm{}
	if err := form.isValid(ctx); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}

	user := models.UserProfile{}
	db := database.GetDBDefault()
	r := db.Where("username = ?", form.Username).First(&user)
	if r.Error == nil {
		if user.IsActived() {
			ctx.JSON(400, BaseJSONResponse(400, "user already exists"))
			return
		}
	} else if r.RecordNotFound() {
		user.IsActive = false
	}

	user.Username = form.Username
	user.Email = form.Username
	user.FirstName = form.FirstName
	user.LastName = form.LastName
	user.Company = form.Company
	user.Telephone = form.Telephone
	user.DateJoined = models.JSONTimeNow()
	user.SetPassword(form.Password)
	if r := db.Save(&user); r.RowsAffected == 1 && r.Error == nil {
		ctx.JSON(201, BaseJSONResponse(201, "用户创建成功"))
		return
	}
	ctx.JSON(500, BaseJSONResponse(500, "用户创建失败"))
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

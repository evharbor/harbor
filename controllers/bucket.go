package controllers

import (
	"errors"
	"fmt"
	"harbor/models"
	"harbor/utils/paginations"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// BucketController 存储桶控制器结构
type BucketController struct {
	Controller
}

// NewBucketController new controller
func NewBucketController() *BucketController {
	return &BucketController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *BucketController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// PageNumberInfo 分页页码信息结构
type PageNumberInfo struct {
	Current uint64 `json:"current"`
	Final   uint64 `json:"final"`
}

// BucketListJSON 存储桶列表信息结构
type BucketListJSON struct {
	BaseJSON
	Count   uint            `json:"count"`
	Next    string          `json:"next"`
	Privous string          `json:"previous"`
	Page    PageNumberInfo  `json:"page"`
	Buckets []models.Bucket `json:"buckets"`
}

// Get controller
// @Summary 获取存储桶列表
// @Description 通过query参数“offset”和“limit”自定义获取存储桶列表
// @Tags Bucket 存储桶
// @Accept  json
// @Produce  json
// @Param   offset     query    int     true        "The initial index from which to return the results"
// @Param   limit      query    int     true        "Number of results to return per page"
// @Success 200 {object} controllers.BucketListJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Failure 500 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/buckets/ [get]
func (ctl BucketController) Get(ctx *gin.Context) {

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	paginater := paginations.NewOptimizedLimitOffsetPagination()
	if err := paginater.PrePaginate(ctx); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	var buckets = make([]models.Bucket, 0)
	bManager := models.NewBucketManager("", user)
	dbQuery := bManager.GetUserBucketsQuery()
	if err := paginater.PaginateDBQuery(&buckets, dbQuery); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}

	current, final := paginater.CurrentAndFinalPageNumber()
	bj := BaseJSONResponse(200, "ok")
	data := BucketListJSON{
		BaseJSON: *bj,
		Count:    uint(paginater.GetCount()),
		Buckets:  buckets,
		Next:     paginater.GetNextURL(),
		Privous:  paginater.GetPreviousURL(),
		Page: PageNumberInfo{
			Current: current,
			Final:   final,
		},
	}
	ctx.JSON(200, data)
}

// BucketPostForm create bucket post form struct
type BucketPostForm struct {
	Name string `json:"name" form:"name" binding:"required"`
}

func (f *BucketPostForm) isValid(ctx *gin.Context) error {

	if err := ctx.ShouldBind(f); err != nil {
		return err
	}
	return f.validate()
}

func (f *BucketPostForm) validate() error {

	name := f.Name
	if strings.HasPrefix(name, "-") {
		return errors.New("bucket name can not start with '-'")
	}
	if strings.HasSuffix(name, "-") {
		return errors.New("bucket name can not end with '-'")
	}
	if len(name) < 3 {
		return errors.New("the length of bucket name should not be less than 3")
	}
	if err := bucketDNSStringValidator(name); err != nil {
		return err
	}
	f.Name = strings.ToLower(name)
	return nil
}

var bucketNameRegex = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]$")

// 验证字符串是否符合NDS标准
func bucketDNSStringValidator(s string) error {

	if bucketNameRegex.MatchString(s) {
		return nil
	}
	return errors.New("bucket name does not meet DNS standards")
}

type bucketPost400JSON struct {
	BaseJSON
	Data     *BucketPostForm `json:"data"`
	Existing bool            `json:"existing"`
}

type bucketPostJSON struct {
	BaseJSON
	Data   *BucketPostForm `json:"data"`
	Bucket *models.Bucket  `json:"bucket"`
}

// Post controller
// @Summary 创建存储桶
// @Description 存储桶名称只能由字母、数字和“-”组成，且不能以“-”开头和结尾，长度3-64字符，符合DNS标准。
// @Tags Bucket 存储桶
// @Accept  json
// @Produce  json
// @Param   data body controllers.BucketPostForm true "bucket name"
// @Success 201 {object} controllers.bucketPostJSON
// @Failure 400 {object} controllers.bucketPost400JSON
// @Failure 500 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/buckets/ [post]
func (ctl BucketController) Post(ctx *gin.Context) {

	form := BucketPostForm{}
	if err := form.isValid(ctx); err != nil {
		bj := BaseJSONResponse(400, err.Error())
		ctx.JSON(400, &bucketPost400JSON{
			BaseJSON: *bj,
			Data:     &form,
		})
		return
	}

	bucketName := form.Name
	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	bManager := models.NewBucketManager(bucketName, user)
	bucket, err := bManager.GetBucket()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	} else if bucket != nil { //bucket exists
		s := fmt.Sprintf("bucket '%s' is already exists", bucketName)
		bj := BaseJSONResponse(400, s)
		ctx.JSON(400, &bucketPost400JSON{
			BaseJSON: *bj,
			Data:     &form,
			Existing: true,
		})
		return
	}
	bucket, err = bManager.CreateBucket()
	if err != nil {
		s := fmt.Sprintf("Create bucket error:'%s'", err.Error())
		ctx.JSON(500, BaseJSONResponse(500, s))
		return
	}
	if err := bManager.CreateObjsTable(bucket); err != nil {
		bManager.DeleteBucket(bucket)
		s := fmt.Sprintf("Create bucket error:'%s'", err.Error())
		ctx.JSON(500, BaseJSONResponse(500, s))
		return
	}

	bj := BaseJSONResponse(201, "Success to create bucket")
	ctx.JSON(201, &bucketPostJSON{
		BaseJSON: *bj,
		Data:     &form,
		Bucket:   bucket,
	})
}

// BucketDetailController 存储桶控制器结构
type BucketDetailController struct {
	Controller
}

// NewBucketDetailController new controller
func NewBucketDetailController() *BucketDetailController {
	return &BucketDetailController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *BucketDetailController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

type bucketDetailForm struct {
	IDs []string `json:"ids" form:"ids" validate:"omitempty,"`
}

func (f *bucketDetailForm) isValid(ctx *gin.Context) error {

	if err := ctx.ShouldBind(&f); err != nil {
		return err
	}
	if id := ctx.Param("id"); id != "" {
		f.IDs = append(f.IDs, id)
	}

	if err := f.validate(); err != nil {
		return err
	}
	return nil
}

func (f *bucketDetailForm) validate() error {

	var ids []string
	for _, id := range f.IDs {
		if id != "" {
			if _, err := strconv.ParseUint(id, 10, 64); err != nil {
				return errors.New("invalid id")
			}
			ids = append(ids, id)
		}
	}
	f.IDs = ids
	return nil
}

type bucketDetail400JSON struct {
	BaseJSON
	Data *bucketDetailForm `json:"data"`
}

type bucketDetailJSON struct {
	BaseJSON
	Bucket *models.Bucket `json:"bucket"`
}

// Get controller
// @Summary 获取存储桶详细信息
// @Description 获取存储桶详细信息
// @Tags Bucket 存储桶
// @Accept  json
// @Produce  json
// @Param   id path int64 true "bucket id"
// @Success 200 {object} controllers.bucketDetailJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Failure 500 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/buckets/{id}/ [get]
func (ctl BucketDetailController) Get(ctx *gin.Context) {

	// get bucket
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, "invalid id"))
		return
	}

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	bManager := models.NewBucketManager("", user)
	bucket, err := bManager.GetUserBucketByID(id)
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	} else if bucket == nil {
		ctx.JSON(404, BaseJSONResponse(404, "bucket is not found"))
		return
	}

	ctx.JSON(200, &bucketDetailJSON{
		BaseJSON: BaseJSON{Code: 200, CodeText: "ok"},
		Bucket:   bucket,
	})
}

// Delete controller
// @Summary 删除存储桶
// @Description #可以一次删除多个存储桶，其余存储桶id通过form ids传递。
// @Tags Bucket 存储桶
// @Accept  json
// @Produce  json
// @Param   id path int64 true "bucket id"
// @Param   data body controllers.bucketDetailForm true "bucket id list"
// @Success 204 {string} string
// @Failure 400 {object} controllers.BaseJSON
// @Failure 500 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/buckets/{id}/ [delete]
func (ctl BucketDetailController) Delete(ctx *gin.Context) {

	form := bucketDetailForm{}
	if err := form.isValid(ctx); err != nil {
		bj := BaseJSONResponse(400, err.Error())
		ctx.JSON(400, &bucketDetail400JSON{
			BaseJSON: *bj,
			Data:     &form,
		})
		return
	}

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	ids := form.IDs
	bManager := models.NewBucketManager("", user)
	if err := bManager.SoftDeleteUserBucketsByIDs(ids); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}

	ctx.JSON(204, nil)
}

type bucketPatchJSON struct {
	BaseJSON
	Public bool   `json:"public,omitempty"`
	Rename string `json:"rename,omitempty"`
}

// Patch controller
// @Summary  设置存储桶访问权限或重命名存储桶
// @Description	#设置存储桶访问权限，提交query参数“public”, true(公有)，false(私有);
// @Description	#重命名存储桶，提交query参数“rename”,其值为新名称;
// @Description	#可以一次设置多个存储桶访问权限，其余存储桶id通过form ids传递, 重命名时ids无效。
// @Description	#同时提交“public”和“rename”参数,忽略“rename”参数
// @Tags Bucket 存储桶
// @Accept  json
// @Produce  json
// @Param   id path int64 true "bucket id"
// @Param   public query bool false "设置对象公有或私有, true(公有)，false(私有)"
// @Param   rename query string false "重命名桶,值为存储桶新名称"
// @Param   data body controllers.bucketDetailForm true "bucket id list,一次设置多个桶的权限时使用，命重名桶时无效"
// @Success 200 {object} controllers.bucketPatchJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Failure 500 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/buckets/{id} [patch]
func (ctl BucketDetailController) Patch(ctx *gin.Context) {

	if public, exists := ctx.GetQuery("public"); exists {
		ctl.patchPublic(ctx, public)
		return
	}

	if rename, exists := ctx.GetQuery("rename"); exists {
		ctl.patchRename(ctx, rename)
		return
	}

	ctx.JSON(400, BaseJSONResponse(400, "invalid request"))
	return
}

func (ctl BucketDetailController) patchRename(ctx *gin.Context, rename string) {

	form := BucketPostForm{Name: rename}
	if err := form.validate(); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	rename = form.Name

	// get bucket
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, "invalid id"))
		return
	}

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	bManager := models.NewBucketManager("", user)
	bucket, err := bManager.GetUserBucketByID(id)
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	} else if bucket == nil {
		ctx.JSON(404, BaseJSONResponse(404, "bucket is not found"))
		return
	}

	//check new bucket name exists
	if b, err := bManager.GetBucketByName(rename); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	} else if b != nil {
		ctx.JSON(400, BaseJSONResponse(400, "A bucket with the same name already exists"))
		return
	}

	if err := bManager.BucketRename(bucket, rename); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}

	bj := BaseJSONResponse(200, "success to rename bucket")
	ctx.JSON(200, &bucketPatchJSON{
		BaseJSON: *bj,
		Rename:   rename,
	})
}

func (ctl BucketDetailController) patchPublic(ctx *gin.Context, pub string) {

	var public bool
	if pub == "true" {
		public = true
	} else if pub != "false" {
		ctx.JSON(400, BaseJSONResponse(400, "the value of query param public is invalid"))
		return
	}

	form := bucketDetailForm{}
	if err := form.isValid(ctx); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}

	ids := form.IDs
	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	bManager := models.NewBucketManager("", user)
	if err := bManager.SetUserBucketsAccessByIDs(ids, public); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}

	bj := BaseJSONResponse(200, "Success to set bucket access permission")
	ctx.JSON(200, &bucketPatchJSON{
		BaseJSON: *bj,
		Public:   public,
	})
	return
}

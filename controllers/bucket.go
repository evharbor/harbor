package controllers

import (
	"fmt"
	"harbor/models"
	"harbor/utils"

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
// @Description 获取存储桶列表
// @Tags Bucket 存储桶
// @Accept  json
// @Produce  json
// @Param   offset     query    int     true        "The initial index from which to return the results"
// @Param   limit      query    int     true        "Number of results to return per page"
// @Success 200 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/buckets/ [get]
func (ctl BucketController) Get(ctx *gin.Context) {

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	paginater := utils.NewOptimizedLimitOffsetPagination()
	if err := paginater.PrePaginate(ctx); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	var buckets []models.Bucket
	bManager := models.NewBucketManager("", nil)
	db := bManager.GetDB()
	dbQuery := db.Where("user_id = ?", user.ID).Order("id desc")
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

// PostBucketForm create bucket post form struct
type PostBucketForm struct {
	Name string `json:"name"`
}

// Post controller
// @Description 创建存储桶
// @Tags Bucket 存储桶
// @Accept  json
// @Produce  json
// @Param   data body controllers.PostBucketForm true "bucket name"
// @Success 200 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/buckets/ [post]
func (ctl BucketController) Post(ctx *gin.Context) {

	form := PostBucketForm{}
	if err := ctx.ShouldBind(&form); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	bucketName := form.Name
	if bucketName == "" {
		ctx.JSON(500, BaseJSONResponse(500, "name cannot be empty"))
		return
	}
	iUser, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(401, BaseJSONResponse(401, "unauthenticated user"))
		return
	}
	user, ok := iUser.(*models.UserProfile)
	if !ok {
		ctx.JSON(401, BaseJSONResponse(401, "unauthenticated user"))
		return
	}

	bManager := models.NewBucketManager(bucketName, user)
	bucket, err := bManager.GetBucket()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	} else if bucket != nil { //bucket not exists
		s := fmt.Sprintf("bucket '%s' is already exists", bucketName)
		ctx.JSON(400, BaseJSONResponse(400, s))
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

	ctx.JSON(201, BaseJSONResponse(201, "Success to create bucket"))
}

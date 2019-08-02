package controllers

import (
	"harbor/models"
	"harbor/utils/paginations"

	"github.com/gin-gonic/gin"
)

// DirController 目录控制器结构
type DirController struct {
	Controller
}

// NewDirController new controller
func NewDirController() *DirController {
	return &DirController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *DirController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// DirListJSON 目录列表信息结构
type DirListJSON struct {
	BaseJSON
	BucketName string                `json:"bucket_name"`
	DirPath    string                `json:"dir_path"`
	Count      uint                  `json:"count"`
	Next       string                `json:"next"`
	Privous    string                `json:"previous"`
	Page       PageNumberInfo        `json:"page"`
	Files      []models.HarborObject `json:"files"`
}

// Get controller
// @Description 获取目录下目录和对象列表
// @Tags Dir 目录
// @Accept  json
// @Produce  json
// @Param   bucketname path string true "bucketname"
// @Param   dirpath path string false "dirpath"
// @Param   offset     query    int     true        "The initial index from which to return the results"
// @Param   limit      query    int     true        "Number of results to return per page"
// @Success 200 {object} controllers.DirListJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/dir/{bucketname}/{dirpath} [get]
func (ctl DirController) Get(ctx *gin.Context) {

	bucketName := ctx.Param("bucketname")
	dirPath := ctx.Param("dirpath")
	dirPath = ClearPath(dirPath)

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	bm := models.NewBucketManager(bucketName, user)
	bucket, err := bm.GetUserBucket()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}
	if bucket == nil {
		ctx.JSON(404, BaseJSONResponse(404, "bucket not found"))
		return
	}

	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, dirPath, "")
	dbQuery, err := manager.GetObjectsQuery()
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, "directory not found"))
		return
	}

	paginater := paginations.NewOptimizedLimitOffsetPagination()
	if err := paginater.PrePaginate(ctx); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	var objs []models.HarborObject
	if err := paginater.PaginateDBQuery(&objs, dbQuery); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}
	current, final := paginater.CurrentAndFinalPageNumber()
	bj := BaseJSONResponse(200, "ok")
	data := DirListJSON{
		BaseJSON:   *bj,
		BucketName: bucketName,
		DirPath:    dirPath,
		Count:      uint(paginater.GetCount()),
		Files:      objs,
		Next:       paginater.GetNextURL(),
		Privous:    paginater.GetPreviousURL(),
		Page: PageNumberInfo{
			Current: current,
			Final:   final,
		},
	}
	ctx.JSON(200, data)
}

// DirCreateJSON create dir response json struct
type DirCreateJSON struct {
	BaseJSON
	Dir *models.HarborObject
}

// DirCreate400JSON create dir response json struct
type DirCreate400JSON struct {
	BaseJSON
	Existing bool `json:"existing"`
}

// Post controller
// @Description 创建目录
// @Tags Dir 目录
// @Accept  json
// @Produce  json
// @Param   bucketname path string true "bucketname"
// @Param   dirpath path string true "dirpath"
// @Success 200 {object} controllers.DirCreateJSON
// @Failure 400 {object} controllers.DirCreate400JSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/dir/{bucketname}/{dirpath} [post]
func (ctl DirController) Post(ctx *gin.Context) {

	bucketName := ctx.Param("bucketname")
	dirPath := ctx.Param("dirpath")
	dirPath, dirName := SplitPathAndFilename(dirPath)

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	bm := models.NewBucketManager(bucketName, user)
	bucket, err := bm.GetUserBucket()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}
	if bucket == nil {
		ctx.JSON(404, BaseJSONResponse(404, "bucket not found"))
		return
	}

	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, dirPath, "")
	dir, created, err := manager.GetDirOrCreateUnderCurrent(dirName)
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	}
	if dir != nil && created == false {
		s := "A directory with the same name already exists"
		existing := true
		if dir.IsFile() {
			s = "A object with the same name already exists"
			existing = false
		}
		bj := BaseJSONResponse(400, s)
		ctx.JSON(400, &DirCreate400JSON{BaseJSON: *bj, Existing: existing})
		return
	}
	ret := &DirCreateJSON{
		BaseJSON: BaseJSON{Code: 201, CodeText: "Create ok"},
		Dir:      dir,
	}
	ctx.JSON(201, ret)
}

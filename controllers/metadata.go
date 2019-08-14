package controllers

import (
	"harbor/models"
	"strings"

	"github.com/gin-gonic/gin"
)

// MetadataController 对象或目录元数据控制器结构
type MetadataController struct {
	Controller
}

// NewMetadataController new controller
func NewMetadataController() *MetadataController {
	return &MetadataController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *MetadataController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// GetPermissions return permission
func (ctl MetadataController) GetPermissions(ctx *gin.Context) []PermissionFunc {

	method := strings.ToUpper(ctx.Request.Method)
	switch method {
	case "GET":
		return []PermissionFunc{IsAuthenticatedUser}
	default:
		return []PermissionFunc{}
	}
}

// ObjMetadataJSON object metadata json struct
type ObjMetadataJSON struct {
	BaseJSON
	Data *models.HarborObject `json:"data"`
}

// Get handler for get method
// @Summary 获取目录或对象元数据
// @Description 通过绝对路径获取目录或对象元数据
// @Tags metadata元数据
// @Accept  json
// @Produce json
// @Param   bucketname 	path string true "bucketname"
// @Param   path 		path string true "objpath"
// @Success 200 {object} controllers.ObjMetadataJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/metadata/{bucketname}/{path} [get]
func (ctl MetadataController) Get(ctx *gin.Context) {

	var (
		err error
	)

	objPath := ClearPath(ctx.Param("path"))
	dirPath, objName := SplitPathAndFilename(objPath)
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "path is invalid"))
		return
	}

	// bucket
	bucket := ctl.getUserBucketOrResponse(ctx)
	if bucket == nil {
		return
	}

	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, dirPath, objName)
	hobj, err := manager.GetObjOrDirExists()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	} else if hobj == nil {
		ctx.JSON(404, BaseJSONResponse(404, "object not found"))
		return
	}

	if hobj.IsFile() {
		dPath := URLPathJoin([]string{"obs", bucket.Name, objPath})
		dURL := ctl.buildAbsoluteURI(ctx, dPath, nil)
		hobj.DownloadURL = dURL
	}

	ctx.JSON(200, &ObjMetadataJSON{
		BaseJSON: *BaseJSONResponse(200, "ok"),
		Data:     hobj,
	})
}

// getUserBucketOrResponse get user own bucket
// return:
//		nil: error
//		bucket: success
func (ctl MetadataController) getUserBucketOrResponse(ctx *gin.Context) *models.Bucket {

	bucketName := ctx.Param("bucketname")
	user := ctl.user
	bm := models.NewBucketManager(bucketName, user)
	bucket, err := bm.GetUserBucket()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return nil
	}
	if bucket == nil {
		ctx.JSON(404, BaseJSONResponse(404, "bucket not found"))
		return nil
	}

	return bucket
}

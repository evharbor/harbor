package controllers

import (
	"errors"
	"harbor/models"
	"strings"

	"github.com/gin-gonic/gin"
)

// MoveController 对象移动或重命名控制器
type MoveController struct {
	Controller
}

// NewMoveController new controller
func NewMoveController() *MoveController {
	return &MoveController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *MoveController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// GetPermissions return permission
func (ctl MoveController) GetPermissions(ctx *gin.Context) []PermissionFunc {

	method := strings.ToUpper(ctx.Request.Method)
	switch method {
	case "POST":
		return []PermissionFunc{IsAuthenticatedUser}
	default:
		return []PermissionFunc{}
	}
}

// ObjMoveJSON object metadata json struct
type ObjMoveJSON struct {
	BaseJSON
	BucketName string               `json:"bucket_name"`
	DirPath    string               `json:"dir_path"`
	Obj        *models.HarborObject `json:"obj"`
}

// Post handler for post method
// @Summary 对象移动或重命名
// @Description 移动或重命名一个对象
// @Description        参数move_to指定对象移动的目标路径（bucket桶下的目录路径），/或空字符串表示桶下根目录；参数rename指定重命名对象的新名称；
// @Description       请求时至少提交其中一个参数，亦可同时提交两个参数；只提交参数move_to只移动对象，只提交参数rename只重命名对象；
// @Tags move移动或重命名
// @Accept  json
// @Produce json
// @Param   bucketname 	path string true "bucketname"
// @Param   objpath 	path string true "objpath"
// @Param   move_to 	query string false "path for move"
// @Param   rename 		query string false "rename object"
// @Success 200 {object} controllers.ObjMoveJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/move/{bucketname}/{objpath} [post]
func (ctl MoveController) Post(ctx *gin.Context) {

	var (
		err error
	)

	objPath := ClearPath(ctx.Param("objpath"))
	dirPath, objName := SplitPathAndFilename(objPath)
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	moveTo, rename, err := ctl.postQueryParamOrResponse(ctx)
	if err != nil {
		return
	}

	// bucket
	bucket := ctl.getUserBucketOrResponse(ctx)
	if bucket == nil {
		return
	}

	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, dirPath, objName)
	hobj, err := manager.GetObjExists()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, err.Error()))
		return
	} else if hobj == nil {
		ctx.JSON(404, BaseJSONResponse(404, "object not found"))
		return
	}
	ctl.moveRenameObj(ctx, bucket, hobj, moveTo, rename)

	// dPath := URLPathJoin([]string{"obs", bucket.Name, objPath})
	// dURL := ctl.buildAbsoluteURI(ctx, dPath, nil)
	// hobj.DownloadURL = dURL
	// ctx.JSON(200, &ObjMetadataJSON{
	// 	BaseJSON: *BaseJSONResponse(200, "ok"),
	// 	Data:     hobj,
	// })
}

func (ctl MoveController) renameObj(ctx *gin.Context, bucket *models.Bucket,
	obj *models.HarborObject, rename string) {

	dirPath, _ := SplitPathAndFilename(obj.PathName)
	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, dirPath, rename)
	targetObj, err := manager.GetObjOrDirByDidName(obj.ParentID, rename)
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "重命名对象时发生错误"))
		return
	}

	if targetObj != nil {
		ctx.JSON(400, BaseJSONResponse(400, "无法重命名，已存在同名的对象或目录"))
		return
	}

	// rename
	if dirPath != "" {
		obj.PathName = dirPath + "/" + rename
	} else {
		obj.PathName = rename
	}
	obj.Name = rename
	if err := manager.SaveObject(obj); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "重命名对象时发生错误"))
		return
	}

	ctx.JSON(201, &ObjMoveJSON{
		BaseJSON:   *BaseJSONResponse(201, "重命名对象成功"),
		BucketName: bucket.Name,
		DirPath:    dirPath,
		Obj:        obj,
	})
}

// moveRenameObj 移动重命名对象
// :param bucket: 对象所在桶
// :param obj: 文件对象
// :param move_to: 移动目标路径
// :param rename: 重命名的新名称
func (ctl MoveController) moveRenameObj(ctx *gin.Context, bucket *models.Bucket,
	obj *models.HarborObject, moveTo, rename string) {

	var newObjName string

	// 仅仅重命名对象，不移动
	if moveTo == "" {
		ctl.renameObj(ctx, bucket, obj, rename)
		return
	}

	// 移动后对象的名称，对象名称不变或重命名
	if rename == "" {
		newObjName = obj.Name
	} else {
		newObjName = rename
	}

	// 检查是否符合移动或重命名条件，目标路径下是否已存在同名对象或子目录
	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, moveTo, newObjName)
	targetObj, err := manager.GetObjOrDirExists()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "无法完成对象的移动操作:"+err.Error()))
		return
	}

	if targetObj != nil {
		ctx.JSON(400, BaseJSONResponse(400, "无法完成对象的移动操作，指定的目标路径下已存在同名的对象或目录"))
		return
	}

	// 移动对象或重命名
	did, _ := manager.GetCurDirID()
	obj.ParentID = did
	obj.PathName = manager.GetObjPathName()
	obj.Name = newObjName
	if err := manager.SaveObject(obj); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "移动或重命名对象时发生错误"))
		return
	}

	dPath := URLPathJoin([]string{"obs", bucket.Name, obj.PathName})
	dURL := ctl.buildAbsoluteURI(ctx, dPath, nil)
	obj.DownloadURL = dURL

	ctx.JSON(201, &ObjMoveJSON{
		BaseJSON:   *BaseJSONResponse(201, "重命名对象成功"),
		BucketName: bucket.Name,
		DirPath:    moveTo,
		Obj:        obj,
	})
}

// getUserBucketOrResponse get user own bucket
// return:
//		nil: error
//		bucket: success
func (ctl MoveController) getUserBucketOrResponse(ctx *gin.Context) *models.Bucket {

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

// postQueryParamOrResponse validate query param of post request
// moveTo == ""		:param not exists
// moveTo == "/"	:root dir
func (ctl MoveController) postQueryParamOrResponse(ctx *gin.Context) (moveTo, rename string, err error) {

	var existsRn, existsMt bool

	rename, existsRn = ctx.GetQuery("rename")
	if existsRn {
		if rename == "" {
			ctx.JSON(400, BaseJSONResponse(400, "rename参数的值不能为空"))
			err = errors.New("rename参数的值不能为空")
			return
		}

		if strings.Contains(rename, "/") {
			ctx.JSON(400, BaseJSONResponse(400, "对象新名称不能含“/”"))
			err = errors.New("对象新名称不能含“/”")
			return
		}

		if len(rename) > 255 {
			ctx.JSON(400, BaseJSONResponse(400, "对象名称不能大于255个字符长度"))
			err = errors.New("对象名称不能大于255个字符长度")
			return
		}
	}

	moveTo, existsMt = ctx.GetQuery("move_to")
	if !existsMt {
		if !existsRn {
			ctx.JSON(400, BaseJSONResponse(400, "请至少提交一个要执行操作的参数"))
			err = errors.New("请至少提交一个要执行操作的参数")
			return
		}
	} else if moveTo == "" {
		moveTo = "/"
	}
	return
}

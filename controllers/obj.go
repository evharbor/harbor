package controllers

import (
	"fmt"
	"harbor/models"
	"harbor/utils/storages"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ObjController 对象控制器结构
type ObjController struct {
	Controller
}

// FormUploadChunk Upload chunk form struct
type FormUploadChunk struct {
	ChunkOffset int64                 `form:"chunk_offset"`
	ChunkSize   int64                 `form:"chunk_size"`
	Chunk       *multipart.FileHeader `form:"chunk"`
}

// NewObjController new controller
func NewObjController() *ObjController {
	return &ObjController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *ObjController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// GetPermissions return permission
func (ctl ObjController) GetPermissions(ctx *gin.Context) []PermissionFunc {

	method := strings.ToUpper(ctx.Request.Method)
	switch method {
	case "GET", "POST", "PATCH", "DELETE":
		return []PermissionFunc{IsAuthenticatedUser}
	default:
		return []PermissionFunc{}
	}
}

// ObjGetInfoJSON object info json struct
// type ObjGetInfoJSON struct {
// 	BaseJSON
// 	BucketName string               `json:"bucket_name"`
// 	DirPath    string               `json:"dir_path"`
// 	Obj        *models.HarborObject `json:"obj"`
// }

// Get handler for get method
// @Summary 下载对象
// @Description 通过文件对象绝对路径,下载文件对象,可通过参数获取文件对象详细信息，或者自定义读取对象数据块
// @Description         * 注：
// @Description         1. offset && size(最大20MB，否则400错误) 参数校验失败时返回状态码400和对应参数错误信息，无误时，返回bytes数据流
// @Description         2. 不带参数时，返回整个文件对象；
// @Description     	* Http Code: 状态码200：
// @Description             evhb_obj_size,文件对象总大小信息,通过标头headers传递：自定义读取时：返回指定大小的bytes数据流；
// @Description             其他,返回整个文件对象bytes数据流；
// @Tags object对象
// @Accept  json
// @Produce application/octet-stream
// @Param   bucketname path string true "bucketname"
// @Param   objpath path string true "objpath"
// @Param   offset     query    int     false        "The byte offset of object to read"
// @Param   size       query    int     false        "Byte size to read"
// @Success 200 {string} string "file"
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/obj/{bucketname}/{objpath} [get]
func (ctl ObjController) Get(ctx *gin.Context) {

	var (
		offset uint64
		size   uint64
		err    error
	)

	objPath := ctx.Param("objpath")
	dirPath, objName := SplitPathAndFilename(objPath)
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	offset, size, err = GetOffsetSizeParam(ctx)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
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

	objkey := hobj.GetObjKey(bucket)
	fs := storages.NewFileStorage(objkey)
	filesize := strconv.FormatInt(fs.FileSize(), 10)
	if size > 0 {
		data, err := fs.Read(int64(offset), int32(size))
		if err != nil {
			ctx.JSON(500, BaseJSONResponse(500, "error read object"))
			return
		}
		chunksize := strconv.FormatInt(int64(len(data)), 10)
		ctx.Header("Content-Type", "application/octet-stream") // 注意格式
		ctx.Header("evob_obj_size", filesize)
		ctx.Header("Content-Length", chunksize)
		ctx.Data(200, "application/octet-stream", data)
		return
	}
	stepFunc, err := fs.StepWriteFunc()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "error read object"))
		return
	}

	filename := url.PathEscape(hobj.Name)                  // 中文文件名需要
	ctx.Header("Content-Type", "application/octet-stream") // 注意格式
	ctx.Header("Content-Length", filesize)
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment;filename*=utf-8''%s", filename)) // 注意filename 这个是下载后的名字
	ctx.Header("evob_obj_size", filesize)
	ctx.Stream(stepFunc)
}

// Post controller
// @Summary 上传对象分片
// @Description 通过文件对象绝对路径分片上传文件对象
// @Description ## 说明：
// @Description * 小文件可以作为一个分片上传，大文件请自行分片上传，分片过大可能上传失败，建议分片大小5-10MB；对象上传支持部分上传，
// @Description 分片上传数据直接写入对象，已成功上传的分片数据永久有效且不可撤销，请自行记录上传过程以实现断点续传；
// @Description * 文件对象已存在时，数据上传会覆盖原数据，文件对象不存在，会自动创建文件对象，并且文件对象的大小只增不减；
// @Description 如果覆盖（已存在同名的对象）上传了一个新文件，新文件的大小小于原同名对象，上传完成后的对象大小仍然保持
// @Description 原对象大小（即对象大小只增不减），如果这不符合你的需求，参考以下2种方法：
// @Description (1)先尝试删除对象（对象不存在返回404，成功删除返回204），再上传；
// @Description (2)访问API时，提交reset参数，reset=true时，在保存分片数据前会先调整对象大小（如果对象已存在），未提供reset参
// @Description  数或参数为其他值，忽略之。
// @Description ## 特别提醒：
// @Description 切记在需要时只在上传第一个分片时提交reset参数，否者在上传其他分片提交此参数会调整对象大小，已上传的分片数据会丢失。
// @Description
// @Description ## 注意：
// @Description 	分片上传现不支持并发上传，并发上传可能造成脏数据，上传分片顺序没有要求，请一个分片上传成功后再上传另一个分片
// @Tags object对象
// @Accept  multipart/form-data
// @Produce  json
// @Param   bucketname path string true "bucketname"
// @Param   objpath path string true "objpath"
// @Param   chunk formData file true "chunk"
// @Param   chunk_offset formData int64 true "chunk_offset"
// @Param   chunk_size formData int64 true "chunk_size"
// @Success 200 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/obj/{bucketname}/{objpath} [post]
func (ctl ObjController) Post(ctx *gin.Context) {

	var reset bool
	var err error
	var hobj *models.HarborObject

	objPath := ctx.Param("objpath")
	dirPath, objName := SplitPathAndFilename(objPath)
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	if reset, err = GetBoolParamOrDefault(ctx, "reset", false); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, "reset param is invalid"))
		return
	}

	form := FormUploadChunk{}
	if err = ctx.ShouldBind(&form); err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}

	chunk := form.Chunk
	offset := form.ChunkOffset
	size := chunk.Size //form.ChunkSize
	if size != chunk.Size {
		ctx.JSON(400, BaseJSONResponse(400, "param 'chunk_size' is not equal to post file size"))
		return
	}

	// bucket
	bucket := ctl.getUserBucketOrResponse(ctx)
	if bucket == nil {
		return
	}

	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, dirPath, objName)
	hobj, err = manager.GetObjExists()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "Get harbor object metadata error"))
		return
	}
	// object exists and param reset == true; reset object size
	if (hobj != nil) && reset {
		oldSize := hobj.Size
		oldTime := hobj.UpdateTime
		objkey := hobj.GetObjKey(bucket)
		fs := storages.NewFileStorage(objkey)

		// modify metadata
		hobj.Size = uint64(size)
		hobj.UpdateModyfiedTime()
		if err := manager.SaveObject(hobj); err != nil {
			ctx.JSON(500, BaseJSONResponse(500, "reset object size failed"))
			return
		}
		// delete object data
		if err := fs.Delete(); err != nil {
			hobj.Size = oldSize
			hobj.UpdateTime = oldTime
			manager.SaveObject(hobj)
			ctx.JSON(500, BaseJSONResponse(500, "reset object size failed"))
			return
		}
	}

	manager.BeginTransaction()
	// create new object
	if hobj == nil {
		hobj, err = manager.CreatObject()
		if err != nil {
			ctx.JSON(500, BaseJSONResponse(500, "create harbor object metadata error"))
			return
		}
	}

	hobj.SetSizeOnlyIncrease(uint64(offset + size))
	hobj.UpdateModyfiedTime()
	if err := manager.SaveObject(hobj); err != nil {
		manager.RollbackTransaction()
		ctx.JSON(500, BaseJSONResponse(500, "upload fialed:"+err.Error()))
		return
	}

	// storage object data
	objkey := hobj.GetObjKey(bucket)
	fs := storages.NewFileStorage(objkey)
	err = fs.WriteFile(offset, chunk)
	if err != nil {
		manager.RollbackTransaction()
		ctx.JSON(500, BaseJSONResponse(500, "upload fialed:"+err.Error()))
		return
	}

	if err := manager.CommitTransaction(); err != nil {
		manager.RollbackTransaction()
		ctx.JSON(500, BaseJSONResponse(500, "upload fialed:"+err.Error()))
		return
	}
	ctx.JSON(200, BaseJSONResponse(200, "success to upload"))
}

// Patch controller
// @Summary 对象共享或私有权限设置
// @Description 对象共享或私有权限设置
// @Tags object对象
// @Accept  json
// @Produce  json
// @Param   bucketname path string true "bucketname"
// @Param   objpath path string true "objpath"
// @Param   share query bool false "是否分享，用于设置对象公有或私有, true(公开)，false(私有)"
// @Param   days query int false "对象公开分享天数(share=true时有效)，0表示永久公开，负数表示不公开，默认为0"
// @Success 200 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/obj/{bucketname}/{objpath} [patch]
func (ctl ObjController) Patch(ctx *gin.Context) {

	// path param
	dirPath, objName := SplitPathAndFilename(ctx.Param("objpath"))
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	// query param
	days, err := GetIntParamOrDefault(ctx, "days", 0)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}
	share, err := GetBoolParamOrDefault(ctx, "share", false)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}

	// bucket
	bucket := ctl.getUserBucketOrResponse(ctx)
	if bucket == nil {
		return
	}

	// object
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

	// update
	hobj.SetShared(share, int(days))
	if err := manager.SaveObject(hobj); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "share object fialed:"+err.Error()))
		return
	}

	ctx.JSON(200, BaseJSONResponse(200, "success to share object"))
}

// Delete controller
// @Summary 删除对象
// @Description 删除一个对象
// @Tags object对象
// @Accept  json
// @Produce  json
// @Param   bucketname path string true "bucketname"
// @Param   objpath path string true "objpath"
// @Success 204 {string} string "No content"
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /api/v1/obj/{bucketname}/{objpath} [delete]
func (ctl ObjController) Delete(ctx *gin.Context) {

	// path param
	dirPath, objName := SplitPathAndFilename(ctx.Param("objpath"))
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	// bucket
	bucket := ctl.getUserBucketOrResponse(ctx)
	if bucket == nil {
		return
	}

	// object
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

	// delete
	if err := manager.DeleteObject(hobj); err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "delete object fialed:"+err.Error()))
		return
	}

	ctx.JSON(200, BaseJSONResponse(200, "success to delete object"))
}

// getUserBucketOrResponse get user own bucket
// return:
//		nil: error
//		bucket: success
func (ctl ObjController) getUserBucketOrResponse(ctx *gin.Context) *models.Bucket {

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

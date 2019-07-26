package controllers

import (
	"fmt"
	"harbor/models"
	"harbor/utils/storages"
	"mime/multipart"
	"net/url"
	"strconv"

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

// Get handler for get method
// @Description 通过文件对象绝对路径,下载文件对象,可通过参数获取文件对象详细信息，或者自定义读取对象数据块
//         *注：可选参数优先级判定顺序：info > offset && size
//         1. 如果携带了info参数，info=true时,返回文件对象详细信息，其他返回400错误；
//         2. offset && size(最大20MB，否则400错误) 参数校验失败时返回状态码400和对应参数错误信息，无误时，返回bytes数据流
//         3. 不带参数时，返回整个文件对象；
//     	>>Http Code: 状态码200：
//             * info=true,返回文件对象详细信息：
//             {
//                 'code': 200,
//                 'bucket_name': 'xxx',   //所在存储桶名称
//                 'dir_path': 'xxxx',      //所在目录
//                 'obj': {},              //文件对象详细信息
//             }
//             * 自定义读取时：返回bytes数据流，其他信息通过标头headers传递：
//             {
//                 evhb_chunk_size: 返回文件块大小
//                 evhb_obj_size: 文件对象总大小
//             }
//             * 其他,返回FileResponse对象,bytes数据流；
//         >>Http Code: 状态码400：文件路径参数有误：对应参数错误信息;
//             {
//                 'code': 400,
//                 'code_text': 'xxxx参数有误'
//             }
//         >>Http Code: 状态码404：找不到资源;
//         >>Http Code: 状态码500：服务器内部错误;
// @Tags object对象
// @Accept  json
// @Produce application/octet-stream
// @Param   bucketname path string true "bucketname"
// @Param   objpath path string true "objpath"
// @Param   offset     query    int     false        "The byte offset of object to read"
// @Param   size       query    int     false        "Byte size to read"
// @Success 200 {object} controllers.UserListJSON
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
	bucketName := ctx.Param("bucketname")
	objPath := ctx.Param("objpath")
	dirPath, objName := SplitPathAndFilename(objPath)
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}
	offset, size, err = GetOffsetSizeParam(ctx)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
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
		ctx.Header("Content-Type", "application/octet-stream") // 注意格式
		ctx.Header("evob_obj_size", filesize)
		ctx.Data(200, "application/octet-stream", data)
		return
	}
	stepFunc, err := fs.StepWriteFunc()
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "error read object"))
		return
	}

	filename := url.PathEscape(hobj.Name) // 中文文件名需要
	ctx.Header("Content-Type", "application/octet-stream") // 注意格式
	ctx.Header("Content-Length", filesize)
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment;filename*=utf-8''%s", filename)) // 注意filename 这个是下载后的名字
	ctx.Header("evob_obj_size", filesize)
	ctx.Stream(stepFunc)
}

// Post controller
// @Description 上传对象分片
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

	bucketName := ctx.Param("bucketname")
	objPath := ctx.Param("objpath")
	dirPath, objName := SplitPathAndFilename(objPath)
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	user := AuthUserOrAbort(ctx)
	if user == nil {
		return
	}

	form := FormUploadChunk{}
	// form, _ := ctx.MultipartForm()
	err := ctx.ShouldBind(&form)
	if err != nil {
		ctx.JSON(400, BaseJSONResponse(400, err.Error()))
		return
	}

	chunk := form.Chunk
	offset := form.ChunkOffset
	size := chunk.Size //form.ChunkSize

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
	manager := models.NewHarborObjectManager(tableName, dirPath, objName)
	manager.BeginTransaction()
	hobj, _ := manager.GetObjOrCreat()
	if hobj == nil {
		manager.RollbackTransaction()
		ctx.JSON(500, BaseJSONResponse(500, "Get harbor object metadata error"))
		return
	}
	hobj.SetSizeOnlyIncrease(uint64(offset + size))
	hobj.UpdateModyfiedTime()
	if err := manager.SaveObject(hobj); err != nil {
		manager.RollbackTransaction()
		ctx.JSON(500, BaseJSONResponse(500, "upload fialed:"+err.Error()))
		return
	}

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
	ctx.JSON(201, BaseJSONResponse(201, "success to upload"))
}

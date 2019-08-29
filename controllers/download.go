package controllers

import (
	"errors"
	"fmt"
	"harbor/models"
	"harbor/utils/convert"
	"harbor/utils/storages"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DownloadController 对象下载控制器结构
type DownloadController struct {
	Controller
}

// NewDownloadController new controller
func NewDownloadController() *DownloadController {
	return &DownloadController{}
}

// Init 初始化this，子类要重写此方法
func (ctl *DownloadController) Init() ControllerInterface {

	ctl.this = ctl
	return ctl
}

// GetPermissions return permission
func (ctl DownloadController) GetPermissions(ctx *gin.Context) []PermissionFunc {

	return []PermissionFunc{}
}

// Get handler for get method
// @Summary 公共或私有对象下载
// @Description 浏览器端下载文件对象，公共文件对象或当前用户(如果用户登录了)文件对象下载，没有权限下载非公共文件对象或不属于当前用户文件对象
// @Description * 支持断点续传，通过HTTP头 Range和Content-Range
// @Description * 跨域访问和安全
// @Description    跨域又需要传递token进行权限认证，我们推荐token通过header传递，不推荐在url中传递token,处理不当会增加token泄露等安全问题的风险。
// @Description    我们支持token通过url参数传递，auth-token和jwt token两种token对应参数名称分别为token和jwt。出于安全考虑，请不要直接把token明文写到前端<a>标签href属性中，以防token泄密。请动态拼接token到url，比如如下方式：
// @Description    $("xxx").on('click', function(e){
// @Description       e.preventDefault();
// @Description       let token = 从SessionStorage、LocalStorage、内存等等存放token的安全地方获取
// @Description       let url = $(this).attr('href') + '?token=' + token; // auth-token
// @Description       let url = $(this).attr('href') + '?jwt=' + token;   // jwt token
// @Description       window.location.href = url;
// @Description    }
// @Tags 对象下载
// @Accept  json
// @Produce application/octet-stream
// @Param   bucketname path string true "bucketname"
// @Param   objpath path string true "objpath"
// @Success 200 {string} string "file"
// @Failure 206 {object} controllers.BaseJSON
// @Failure 400 {object} controllers.BaseJSON
// @Failure 404 {object} controllers.BaseJSON
// @Failure 416 {object} controllers.BaseJSON
// @Security BasicAuth
// @Security ApiKeyAuth
// @Router /obs/{bucketname}/{objpath} [get]
func (ctl DownloadController) Get(ctx *gin.Context) {

	var (
		err error
	)

	objPath := ctx.Param("objpath")
	dirPath, objName := SplitPathAndFilename(objPath)
	if objName == "" {
		ctx.JSON(400, BaseJSONResponse(400, "objpath is invalid"))
		return
	}

	// bucket
	bucket := ctl.getBucketOrResponse(ctx)
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

	// 是否有文件对象的访问权限
	if !ctl.hasAccessPermission(ctx, bucket, hobj) {
		ctx.JSON(403, BaseJSONResponse(403, "forbidden"))
		return
	}

	ctl.fileResponse(ctx, bucket, hobj)
}

// getBucketOrResponse get bucket
// return:
//		nil: error
//		bucket: success
func (ctl DownloadController) getBucketOrResponse(ctx *gin.Context) *models.Bucket {

	bucketName := ctx.Param("bucketname")
	user := ctl.user
	bm := models.NewBucketManager(bucketName, user)
	bucket, err := bm.GetBucket()
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

// hasAccessPermission return true if have download permission
func (ctl DownloadController) hasAccessPermission(ctx *gin.Context,
	bucket *models.Bucket, obj *models.HarborObject) bool {

	// 存储桶是否是公有权限
	if bucket.IsPublic() {
		return true
	}

	// # 可能通过url传递token的身份权限认证
	// self.authentication_url_token(request)

	// 存储桶是否属于当前用户
	if bucket.IsBelongToUser(ctl.user) {
		return true
	}

	// 对象是否共享的，并且在有效共享事件内
	if obj.IsSharedAndInSharedTime() {
		return true
	}

	return false
}

// fileResponse file response
func (ctl DownloadController) fileResponse(ctx *gin.Context,
	bucket *models.Bucket, obj *models.HarborObject) {

	// 是否是断点续传部分读取
	hRange := ctx.GetHeader("Range")
	if hRange != "" {
		ctl.rangeFileResponse(ctx, bucket, obj, hRange)
		return
	}

	ctl.totalFileResponse(ctx, bucket, obj)
}

// totalFileResponse total file response
func (ctl DownloadController) totalFileResponse(ctx *gin.Context,
	bucket *models.Bucket, obj *models.HarborObject) {

	filesize := obj.Size
	objkey := obj.GetObjKey(bucket)
	cho := storages.NewCephHarborObject(objkey, obj.Size)

	stepFunc, err := cho.StepWriteFunc(0, filesize-1)
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "error read object"))
		return
	}
	filename := url.PathEscape(obj.Name) // 中文文件名需要
	strFilesize := strconv.FormatUint(filesize, 10)
	ctx.Header("Accept-Ranges", "bytes")                   // 接受类型，支持断点续传
	ctx.Header("Content-Type", "application/octet-stream") // 注意格式
	ctx.Header("Content-Length", strFilesize)
	ctx.Header("evob_obj_size", strFilesize)
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment;filename*=utf-8''%s", filename)) // 注意filename 这个是下载后的名字
	ctx.Status(http.StatusOK)
	ctx.Stream(stepFunc)

	tableName := bucket.GetObjsTableName()
	manager := models.NewHarborObjectManager(tableName, "", "")
	manager.IncreaseDownloadCount(obj) // 下载次数+1
	return
}

// rangeFileResponse  partial file response
func (ctl DownloadController) rangeFileResponse(ctx *gin.Context,
	bucket *models.Bucket, obj *models.HarborObject, hRange string) {

	var offset int64

	filesize := obj.Size
	objkey := obj.GetObjKey(bucket)
	cho := storages.NewCephHarborObject(objkey, obj.Size)

	start, end, err := ctl.parseHeaderRange(hRange)
	if err != nil {
		ctx.JSON(416, BaseJSONResponse(416, "Header Ranges is invalid"))
		return
	}
	// 读最后end个字节
	if start < 0 && end > 0 {
		offset = convert.MaxInt(int64(filesize)-end, 0)
		end = int64(filesize - 1)

	} else {
		offset = start
		if end < 0 {
			end = int64(filesize - 1)
		} else {
			end = convert.MinInt(end, int64(filesize-1))
		}
	}

	stepFunc, err := cho.StepWriteFunc(uint64(offset), uint64(end))
	if err != nil {
		ctx.JSON(500, BaseJSONResponse(500, "error read object"))
		return
	}

	filename := url.PathEscape(obj.Name) // 中文文件名需要
	ctx.Header("Content-Ranges", fmt.Sprintf("bytes %d-%d/%d", offset, end, filesize))
	ctx.Header("Content-Length", strconv.FormatInt(end-offset+1, 10))
	ctx.Header("Accept-Ranges", "bytes")                                                       // 接受类型，支持断点续传
	ctx.Header("Content-Type", "application/octet-stream")                                     // 注意格式
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment;filename*=utf-8''%s", filename)) // 注意filename 这个是下载后的名字
	ctx.Status(http.StatusPartialContent)
	ctx.Stream(stepFunc)

	if offset == 0 {
		tableName := bucket.GetObjsTableName()
		manager := models.NewHarborObjectManager(tableName, "", "")
		manager.IncreaseDownloadCount(obj) // 下载次数+1
	}
	return
}

// parseHeaderRange parse header 'Range'
// :param ranges: 'bytes={start}-{end}'  //下载第M－N字节范围的内容
// :return:
// 		start >= 0: success; 	start < 0: not exists
// 		end > 0: success; 		end < 0: not exists
func (ctl DownloadController) parseHeaderRange(s string) (start, end int64, err error) {

	var val uint64
	start = -1
	end = -1

	headerRangeRegex := regexp.MustCompile("^bytes=([0-9]*)-([0-9]*)$")
	r := headerRangeRegex.FindStringSubmatch(s)
	if r == nil {
		return
	}
	fmt.Println(r)
	if r[1] != "" {
		val, err = strconv.ParseUint(r[1], 10, 64)
		start = int64(val)
		if err != nil {
			err = errors.New("invalid header 'Range'")
			return
		}
	}
	if r[2] != "" {
		val, err = strconv.ParseUint(r[2], 10, 64)
		end = int64(val)
		if err != nil {
			err = errors.New("invalid header 'Range'")
			return
		}
	}

	if start > end {
		err = errors.New("invalid header 'Range'")
		return
	}

	return
}

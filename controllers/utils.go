package controllers

import (
	"errors"
	"fmt"
	"harbor/models"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ClearPath return a string removed the space and "/" at the front and back ends of input string
func ClearPath(s string) string {
	return strings.Trim(strings.TrimSpace(s), "/")
}

// SplitPathAndFilename return path and filename
// 分割一个绝对路径，获取文件名和父路径,优先获取文件名
func SplitPathAndFilename(s string) (path, filename string) {

	fullpath := ClearPath(s)
	if fullpath == "" {
		// path = ""
		// filename = ""
		return
	}

	i := strings.LastIndex(fullpath, "/")
	if i < 0 {
		// path = ""
		filename = fullpath
		return
	}
	return fullpath[:i], fullpath[i+1:]
}

// SplitBucketPathAndFilename return bucket name、path and filename
// 分割一个绝对路径，获取文件名、存储通名和父路径，优先获取文件名、存储通名
func SplitBucketPathAndFilename(s string) (bucket, path, filename string) {

	fullpath := ClearPath(s)
	if fullpath == "" {
		return
	}
	bucketPath, filename := SplitPathAndFilename(fullpath)
	if bucketPath == "" {
		return
	}
	bucket, path = SplitBucketAndPath(bucketPath)
	return
}

// SplitBucketAndPath return bucket and path string
// 分割一个绝对路径，获取存储通名、文件夹路径，优先获取存储桶
func SplitBucketAndPath(s string) (bucket, path string) {

	bucketPath := ClearPath(s)
	if bucketPath == "" {
		return
	}

	i := strings.Index(bucketPath, "/")
	if i < 0 {
		// path = ""
		bucket = bucketPath
		return
	}
	return bucketPath[:i], bucketPath[i+1:]
}

// BreadcrumbItem 面包屑单个元素结构
type BreadcrumbItem struct {
	Key  string
	Path string
}

// BuildPathBreadcrumb return *[]BreadcrumbItem
// example:
// 		path: "/a/b/c/d"
//		return: &[]BreadcrumbItem{
//			BreadcrumbItem{Key:"a", Path:"a"},
//			BreadcrumbItem{Key:"b", Path:"a/b"},
//			BreadcrumbItem{Key:"c", Path:"a/b/c"},
//			BreadcrumbItem{Key:"d", Path:"a/b/c/d"},
// 		}
func BuildPathBreadcrumb(path string) *[]BreadcrumbItem {

	breadcrumb := []BreadcrumbItem{}
	s := ClearPath(path)
	if s == "" {
		return &breadcrumb
	}

	splits := strings.Split(s, "/")
	for i, k := range splits {
		p := strings.Join(splits[:i+1], "/")
		breadcrumb = append(breadcrumb, BreadcrumbItem{Key: k, Path: p})
	}
	return &breadcrumb
}

// AuthUserOrNil try get auth user or nil
func AuthUserOrNil(ctx *gin.Context) *models.UserProfile {

	if iUser, exists := ctx.Get("user"); exists {
		user, ok := iUser.(*models.UserProfile)
		if ok {
			return user
		}
	}
	// if user := middlewares.UserFromJWTPayload(ctx); user != nil {
	// 	return user
	// }
	return nil
}

// AuthUserOrAbort try get auth user or abort
func AuthUserOrAbort(ctx *gin.Context) *models.UserProfile {

	user := AuthUserOrNil(ctx)
	if user != nil {
		return user
	}

	ctx.JSON(401, BaseJSONResponse(401, "unauthenticated user"))
	ctx.Abort()
	return nil
}

// GetUintParamOrDefault return param's value by name or a error if exists, otherwise return default
func GetUintParamOrDefault(ctx *gin.Context, name string, dft uint64) (uint64, error) {

	value, exists := ctx.GetQuery(name)
	if !exists {
		return dft, nil
	}
	offset, err := strconv.ParseUint(value, 10, 0)
	if err != nil {
		s := fmt.Sprintf("value of query param %s is invalid", name)
		return 0, errors.New(s)
	}

	return offset, nil
}

// GetOffsetParam return offset param value try get from gin.Context
// default return 0 if not found
func GetOffsetParam(ctx *gin.Context) (ofset uint64, err error) {

	return GetUintParamOrDefault(ctx, "offset", 0)
}

// GetLimitParam return limit param value try get from gin.Context
// default return 0 if not found
func GetLimitParam(ctx *gin.Context) (ofset uint64, err error) {

	return GetUintParamOrDefault(ctx, "limit", 0)
}

// GetLimitOffsetParam return offset and limit params that try get from gin.Context
// default return 0 if not found param
func GetLimitOffsetParam(ctx *gin.Context) (offset, limit uint64, err error) {

	limit, err = GetLimitParam(ctx)
	if err != nil {
		return
	}

	offset, err = GetOffsetParam(ctx)
	if err != nil {
		return
	}

	return
}

// GetOffsetSizeParam return offset and size params that try get from gin.Context
// default return 0 if not found param
func GetOffsetSizeParam(ctx *gin.Context) (offset, size uint64, err error) {

	offset, err = GetOffsetParam(ctx)
	if err != nil {
		return
	}
	size, err = GetUintParamOrDefault(ctx, "size", 0)

	return
}

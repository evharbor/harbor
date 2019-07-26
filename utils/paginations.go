package utils

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

// 外键关联查询
// db.SqlDB.First(&device).Preload("CommWeimaqi").Related(&device.DeviceModular)   //查询单条device记录
// db.SqlDB.Preload("DeviceModular.CommWeimaqi").Preload("DeviceModular").Find(&device) //查询所有device记录

// LimitOffsetPagination paginater
// order Of usage:
//  	1) PrePaginate(*gin.Context)
//		2) PaginateDBQuery(out interface{}, db *gorm.DB)
// 		3) other method
type LimitOffsetPagination struct {
	limit  uint64
	offset uint64
	count  uint64
	ctx    *gin.Context
	db     *gorm.DB
}

// NewLimitOffsetPagination return a paginater
func NewLimitOffsetPagination() *LimitOffsetPagination {

	return &LimitOffsetPagination{}
}

// PrePaginate init some params before paginate
// try to get query params limit and offset from request context
func (p *LimitOffsetPagination) PrePaginate(ctx *gin.Context) error {

	var err error
	p.limit, err = strconv.ParseUint(ctx.DefaultQuery("limit", "200"), 10, 0)
	if err != nil {
		return errors.New("error:value of query param limit is invalid")
	}

	p.offset, err = strconv.ParseUint(ctx.DefaultQuery("offset", "0"), 10, 0)
	if err != nil {
		return errors.New("error:value of query param offset is invalid")
	}

	p.ctx = ctx

	return nil
}

// SetLimit reset value of limit
func (p *LimitOffsetPagination) SetLimit(limit uint64) {

	p.limit = limit
}

// GetLimit return limit value
func (p *LimitOffsetPagination) GetLimit() uint64 {

	return p.limit
}

// GetOffset return offset value
func (p *LimitOffsetPagination) GetOffset() uint64 {

	return p.offset
}

// GetCount return count of data
func (p *LimitOffsetPagination) GetCount() uint64 {

	return p.count
}

// GetPreviousURL return previous page url
func (p *LimitOffsetPagination) GetPreviousURL() string {

	limit := p.GetLimit()
	offset := p.GetOffset()
	// first page current
	if offset == 0 {
		return ""
	}

	preLimit := limit
	var preOffset int64
	preOffset = int64(offset - limit)
	if preOffset < 0 {
		preOffset = 0
		preLimit = p.offset
	}

	querys := map[string]string{}
	querys["offset"] = strconv.FormatUint(uint64(preOffset), 10)
	querys["limit"] = strconv.FormatUint(preLimit, 10)

	return p.buildAbsoluteURI(querys)
}

//GetNextURL return next page url
func (p *LimitOffsetPagination) GetNextURL() string {

	count := p.GetCount()
	limit := p.GetLimit()
	offset := p.GetOffset()
	if (offset + limit) >= count {
		return ""
	}

	nextLimit := limit
	nextOffset := offset + limit

	querys := map[string]string{}
	querys["offset"] = strconv.FormatUint(nextOffset, 10)
	querys["limit"] = strconv.FormatUint(nextLimit, 10)

	return p.buildAbsoluteURI(querys)
}

// initCount get count of data rows
func (p *LimitOffsetPagination) initCount(db *gorm.DB) error {

	var count uint64
	if err := db.Count(&count).Error; err != nil {
		return errors.New("database error:" + err.Error())
	}
	p.count = count

	return nil
}

// PaginateDBQuery Output data of current page into out if no errors occur, otherwise return a error
// @params db: *gorm.DB Return from Model() or Table() and filtered by where () or not ()
func (p *LimitOffsetPagination) PaginateDBQuery(out interface{}, db *gorm.DB) error {

	if err := p.initCount(db); err != nil {
		return err
	}

	p.db = db
	return p.normalPaginate(out)
}

// normalPaginate Output data of current page into out if no errors occur, otherwise return a error
func (p *LimitOffsetPagination) normalPaginate(out interface{}) error {

	db := p.db
	count := p.GetCount()
	offset := p.GetOffset()
	limit := p.GetLimit()

	if (count == 0) || (offset > count) {
		return nil
	}

	if err := db.Offset(offset).Limit(limit).Find(out).Error; err != nil {
		return errors.New("Error when query database:" + err.Error())
	}

	return nil
}

func (p *LimitOffsetPagination) buildAbsoluteURI(querys map[string]string) string {

	req := p.ctx.Request
	u := *(req.URL) // deep copy
	q := u.Query()
	for key, value := range querys {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	if u.Host == "" {
		u.Host = req.Host
	}

	if u.Scheme == "" {
		if req.ProtoMajor == 1 {
			u.Scheme = "http"
		} else {
			u.Scheme = "https"
		}
	}

	return u.String()
}

// CurrentAndFinalPageNumber return current ande final page number
func (p *LimitOffsetPagination) CurrentAndFinalPageNumber() (uint64, uint64) {

	count := p.GetCount()
	limit := p.GetLimit()
	offset := p.GetOffset()

	current := DivideWithCeil(offset, limit) + 1

	final := DivideWithCeil(count-offset, limit) + DivideWithCeil(offset, limit)

	if final < 1 {
		final = 1
	} else {
		current = 1
		final = 1
	}

	if current > final {
		current = final
	}

	return current, final
}

// DivideWithCeil Returns 'a' divided by 'b', with any remainder rounded up
func DivideWithCeil(a, b uint64) uint64 {

	return (a / b) + 1
}

// OptimizedLimitOffsetPagination paginater
// order Of usage:
//  	1) PrePaginate(*gin.Context)
//		2) PaginateDBQuery(out interface{}, db *gorm.DB)
// 		3) other method
type OptimizedLimitOffsetPagination struct {
	LimitOffsetPagination
}

// NewOptimizedLimitOffsetPagination return a NewOptimizedLimitOffsetPagination type paginater
func NewOptimizedLimitOffsetPagination() *OptimizedLimitOffsetPagination {
	return &OptimizedLimitOffsetPagination{}
}

// PaginateDBQuery Output data of current page into out if no errors occur, otherwise return a error
// @params db: *gorm.DB Return from Model() or Table() and filtered by where () or not ()
func (p *OptimizedLimitOffsetPagination) PaginateDBQuery(out interface{}, db *gorm.DB) error {

	if err := p.initCount(db); err != nil {
		return err
	}

	p.db = db
	count := p.GetCount()
	offset := p.GetOffset()
	limit := p.GetLimit()

	if (count == 0) || (offset > count) {
		return nil
	}

	// 当数据少时
	if count <= 10000 {
		return p.normalPaginate(out)
	}

	if err := db.Select("id").Offset(offset).Limit(1).Find(out).Error; err != nil {
		return errors.New("Error when query database:" + err.Error())
	}

	id, err := p.GetID(out)
	if err != nil {
		return errors.New("Error when query database:" + err.Error())
	}

	where := fmt.Sprintf("id <= %d", id)
	if err := db.Where(where).Offset(offset).Limit(limit).Find(out).Error; err != nil {
		return errors.New("Error when query database:" + err.Error())
	}

	return nil
}

// GetID from paginate out
func (p *OptimizedLimitOffsetPagination) GetID(paginateOut interface{}) (id uint64, err error) {

	value, e := GetValueFromStructByName(paginateOut, "ID")
	if e != nil {
		err = e
		return
	}
	id, err = ToUint(value)
	return
}

// DirListLimitOffsetPagination paginater
// order Of usage:
//  	1) PrePaginate(*gin.Context)
//		2) PaginateDBQuery(out interface{}, db *gorm.DB)
// 		3) other method
type DirListLimitOffsetPagination struct {
	OptimizedLimitOffsetPagination
}

// NewDirListLimitOffsetPagination return a NewDirListLimitOffsetPagination type paginater
func NewDirListLimitOffsetPagination() *DirListLimitOffsetPagination {
	return &DirListLimitOffsetPagination{}
}

// PaginateDBQuery Output data of current page into out if no errors occur, otherwise return a error
// @params out: *[]struct
// @params db: *gorm.DB Return from Model() or Table() and filtered by where () or not ()
func (p *DirListLimitOffsetPagination) PaginateDBQuery(out interface{}, db *gorm.DB) error {

	if err := p.initCount(db); err != nil {
		return err
	}

	p.db = db
	count := p.GetCount()
	offset := p.GetOffset()
	limit := p.GetLimit()

	if (count == 0) || (offset > count) {
		return nil
	}

	// 当数据少时
	if count <= 10000 {
		return p.normalPaginate(out)
	}

	//当数据很多时，目录和对象分开考虑
	var dirsCount uint64
	if err := db.Where("fod = ?", "false").Count(&dirsCount).Error; err != nil {
		return errors.New("Error when query database:" + err.Error())
	}
	// 分页数据只有目录
	if (offset + limit) <= dirsCount {
		if err := db.Where("fod = ?", "false").Offset(offset).Limit(limit).Find(out).Error; err != nil {
			return errors.New("Error when query database:" + err.Error())
		}
		return nil
	}

	// 分页数据只有对象
	if offset >= dirsCount {
		objsOffset := offset - dirsCount
		// 偏移量objsOffset较小时
		if objsOffset <= 10000 {
			if err := db.Where("fod = ?", "true").Offset(objsOffset).Limit(limit).Find(out).Error; err != nil {
				return errors.New("Error when query database:" + err.Error())
			}
			return nil
		}

		// 偏移量objsOffset较大时
		if err := db.Where("fod = ?", "true").Select("id").Offset(objsOffset).Limit(1).Find(out).Error; err != nil {
			return errors.New("Error when query database:" + err.Error())
		}

		id, err := p.GetID(out)
		if err != nil {
			return errors.New("Error when query database:" + err.Error())
		}

		where := fmt.Sprintf("id <= %d", id)
		if err := db.Where(where).Offset(objsOffset).Limit(limit).Find(out).Error; err != nil {
			return errors.New("Error when query database:" + err.Error())
		}

		return nil
	}

	// 分页数据包含目录和对象
	// 目录
	if err := db.Where("fod = ?", "false").Offset(offset).Limit(limit).Find(out).Error; err != nil {
		return errors.New("Error when query database:" + err.Error())
	}
	refValue := indirect(reflect.ValueOf(out))
	retOut := reflect.MakeSlice(refValue.Type(), 0, 0)
	if err := copySlice(retOut, refValue); err != nil {
		return err
	}
	dl := retOut.Len()
	ol := limit - uint64(dl)
	// 对象
	if err := db.Where("fod = ?", "false").Offset(offset).Limit(ol).Find(out).Error; err != nil {
		return errors.New("Error when query database:" + err.Error())
	}

	refValue = indirect(reflect.ValueOf(out))
	if err := copySlice(retOut, refValue); err != nil {
		return err
	}
	out = retOut.Interface()

	return nil
}

func copySlice(retOut, refValue reflect.Value) error {

	var (
		isPtr      bool
		resultType reflect.Type
	)

	if kind := refValue.Kind(); kind != reflect.Slice {
		return errors.New("input must be slice")
	}
	resultType = refValue.Type().Elem()
	if resultType.Kind() == reflect.Ptr {
		isPtr = true
		resultType = resultType.Elem()
	}
	// elem = reflect.New(resultType).Elem()
	for i := 0; i < refValue.Len(); i++ {
		elem := refValue.Index(i)
		if isPtr {
			retOut.Set(reflect.Append(retOut, elem.Addr()))
		} else {
			retOut.Set(reflect.Append(retOut, elem))
		}
	}
	return nil
}

// Clone 完整复制数据
func Clone(a, b interface{}) error {
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	dec := gob.NewDecoder(buff)
	if err := enc.Encode(a); err != nil {
		return err
	}
	if err := dec.Decode(b); err != nil {
		return err
	}
	return nil
}

func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

// GetValueFromStructByName get value by name from a struct or struct slice
func GetValueFromStructByName(obj interface{}, name string) (value interface{}, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
			return
		}
	}()

	refValue := indirect(reflect.ValueOf(obj))
	if kind := refValue.Kind(); kind == reflect.Slice {
		obj := indirect(refValue.Index(0))
		if obj.Kind() == reflect.Struct {
			value = obj.FieldByName(name).Interface()
			return
		}
	} else if kind := refValue.Kind(); kind == reflect.Struct {
		value = refValue.FieldByName(name).Interface()
		return
	}

	err = errors.New("Input must be struct or struct slice")
	return
}

// ToFloat try convert interface to float64.
func ToFloat(value interface{}) (val float64, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = errors.New("can not convert to as type float")
			return
		}
	}()

	val = reflect.ValueOf(value).Float()
	return
}

// ToInt try convert interface to int
func ToInt(value interface{}) (val int64, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = errors.New("can not convert to as type int")
			return
		}
	}()

	val = reflect.ValueOf(value).Int()
	return
}

// ToUint try convert interface to uint
func ToUint(value interface{}) (val uint64, err error) {

	defer func() {
		if e := recover(); e != nil {
			err = errors.New("can not convert to as type uint")
			return
		}
	}()

	val = reflect.ValueOf(value).Uint()
	return
}

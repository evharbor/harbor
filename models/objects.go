package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const (
	// BucketPublic 公有
	BucketPublic uint8 = 1
	// BucketPrivate 私有
	BucketPrivate uint8 = 2
)

// TypeBucketPermission access permission date type
type TypeBucketPermission uint8

// MarshalJSON on TypeBucketPermission convert uint8 to string
func (v TypeBucketPermission) MarshalJSON() ([]byte, error) {
	var ap string
	if uint8(v) == BucketPrivate {
		ap = "私有"
	} else if uint8(v) == BucketPublic {
		ap = "公有"
	}
	formatted := fmt.Sprintf("\"%s\"", ap)
	return []byte(formatted), nil
}

// Value insert value into mysql need this function.
func (v TypeBucketPermission) Value() (driver.Value, error) {

	return int64(v), nil
}

// Scan valueof
func (v *TypeBucketPermission) Scan(val interface{}) error {
	value, ok := val.(int64)
	if !ok {
		return fmt.Errorf("can not convert %v to uint8", v)
	}
	*v = TypeBucketPermission(value)
	return nil
}

// Bucket 存储桶结构
type Bucket struct {
	ID               uint64               `gorm:"PRIMARY_KEY;AUTO_INCREMENT;not null" json:"id"`
	Name             string               `gorm:"type:varchar(63);unique_index:uidx_name" json:"name"`
	User             UserProfile          `gorm:"ForeignKey:UserID;SAVE_ASSOCIATIONS:false" json:"-"` //所属用户
	UserID           uint                 `gorm:"column:user_id;index:idx_user_id;" json:"user_id"`   //所属用户id
	CreatedTime      TypeJSONTime         `gorm:"column:created_time;type:datetime;" json:"created_time"`
	CollectionName   string               `gorm:"column:collection_name;type:varchar(50)" json:"-"`                //存储桶对应的表名
	AccessPermission TypeBucketPermission `gorm:"column:access_permission;type:smallint" json:"access_permission"` //访问权限
	SoftDelete       bool                 `gorm:"column:soft_delete;" json:"-"`                                    // True->删除状态
	ModifiedTime     TypeJSONTime         `gorm:"column:modyfied_time;type:datetime;" json:"-"`                    // 修改时间可以指示删除时间
	ObjsCount        uint32               `gorm:"column:objs_count;" json:"-"`                                     //桶内对象的数量
	Size             uint64               `gorm:"column:size;" json:"-"`                                           //桶内对象的总大小
	StatsTime        TypeJSONTime         `gorm:"column:stats_time;type:datetime;" json:"-"`                       //统计时间
}

// NewBucketDefault create a bucket initialized with default value
func NewBucketDefault() *Bucket {
	now := JSONTimeNow()
	return &Bucket{
		AccessPermission: TypeBucketPermission(BucketPrivate),
		CreatedTime:      now,
		ModifiedTime:     now,
		SoftDelete:       false,
	}
}

// NewBucket create a Bucket
func NewBucket() *Bucket {
	return &Bucket{}
}

// TableName Set Bucket's table name
func (Bucket) TableName() string {
	return "buckets_bucket"
}

// GetObjsTableName 获得bucket对象元数据对应的数据库表名
func (b *Bucket) GetObjsTableName() string {

	if b.CollectionName == "" {
		name := fmt.Sprintf("bucket_%d", b.ID)
		b.CollectionName = name
	}

	return b.CollectionName
}

// IsSoftDelete return true if bucket is soft deleted, otherwise return false
func (b *Bucket) IsSoftDelete() bool {

	return b.SoftDelete
}

// GetSoftDeleteName 获得bucket软删除后的名称
func (b *Bucket) GetSoftDeleteName() string {

	if b.IsSoftDelete() {
		return b.Name
	}
	name := fmt.Sprintf("_%d-%s", b.ID, b.Name) // 唯一name
	if len(name) > 63 {
		name = string(name[0:63])
	}

	return name
}

// SetSoftDeleteName 设置bucket名称为软删除后的名称
func (b *Bucket) SetSoftDeleteName() {

	b.Name = b.GetSoftDeleteName()
}

// UpdateModyfiedTime  update modyfied time of bucket
// @Tips: This change will not be updated to the database,you need to update it explicitly.
func (b *Bucket) UpdateModyfiedTime() {

	b.ModifiedTime = JSONTimeNow()
}

// IsPublic return true if bucket is public access permission
func (b *Bucket) IsPublic() bool {

	if uint8(b.AccessPermission) == BucketPublic {
		return true
	}

	return false
}

// HarborObject 对象结构
type HarborObject struct {
	ID               uint64       `gorm:"PRIMARY_KEY;AUTO_INCREMENT;not null" json:"id"`
	PathName         string       `gorm:"column:na;not null" json:"na"`                                             //全路径文件名或目录名
	FileOrDir        bool         `gorm:"column:fod;index:idx_fod_did;not null" json:"fod"`                         //True==文件，False==目录
	ParentID         uint64       `gorm:"column:did;index:idx_fod_did;unique_index:udx_did_name;not null" json:"-"` //父节点id
	Name             string       `gorm:"type:varchar(255);unique_index:udx_did_name;not null" json:"name"`         //文件名或目录名
	Size             uint64       `gorm:"cloumn:si;not null" json:"si"`                                             //文件大小, 字节数
	UploadTime       TypeJSONTime `gorm:"column:ult;not null" json:"ult"`                                           //文件的上传时间，或目录的创建时间
	UpdateTime       TypeJSONTime `gorm:"column:upt;not null" json:"upt"`                                           //修改时间
	DownloadCount    uint64       `gorm:"column:dlc;not null" json:"dlc"`                                           //该文件的下载次数，目录时dlc为0
	IsShared         bool         `gorm:"column:sh;not null" json:"-"`                                              //为True，则文件可共享，为False，则文件不能共享
	ShareCode        string       `gorm:"column:shp;type:varchar(10);not null" json:"-"`                            //该文件的共享密码，目录时为空
	IsSharedLimit    bool         `gorm:"column:stl;default:true;not null" json:"-"`                                //True: 文件有共享时间限制; False: 则文件无共享时间限制
	SharedStartTime  time.Time    `gorm:"column:sst;not null" json:"-"`                                             //该文件的共享起始时间
	SharedEndTime    time.Time    `gorm:"column:set;not null" json:"-"`                                             //该文件的共享终止时间
	SoftDeleted      bool         `gorm:"column:sds;not null" json:"-"`                                             //软删除,True->删除状态
	AccessPermission string       `gorm:"-" json:"access_permission"`
	DownloadURL      string       `gorm:"-" json:"download_url"`
}

// NewHarborObject create a harbor object
func NewHarborObject() *HarborObject {
	return &HarborObject{}
}

// NewHarborObjectDefault create a harbor object initialized with default value
func NewHarborObjectDefault() *HarborObject {
	now := JSONTimeNow()
	return &HarborObject{
		UploadTime:    now,
		UpdateTime:    now,
		IsSharedLimit: true,
		FileOrDir:     true,
	}
}

// NewHarborDirDefault create a harbor dir initialized with default value
func NewHarborDirDefault() *HarborObject {
	now := JSONTimeNow()
	return &HarborObject{
		UploadTime: now,
		UpdateTime: now,
		FileOrDir:  false,
	}
}

// MarshalJSON on TypeBucketPermission convert uint8 to string
func (ho *HarborObject) MarshalJSON() ([]byte, error) {

	type Alias HarborObject

	if ho.AccessPermission != "公有" {
		if ho.IsFile() {
			if ho.IsSharedAndInSharedTime() {
				ho.AccessPermission = "公有"
			} else {
				ho.AccessPermission = "私有"
			}
		}
	}
	return json.Marshal((*Alias)(ho))
}

// SetSizeOnlyIncrease set HarborObject size only when input large than current size
// @Tips: This change will not be updated to the database,you need to update it explicitly.
func (ho *HarborObject) SetSizeOnlyIncrease(size uint64) {

	if size > ho.Size {
		ho.Size = size
	}
}

// UpdateModyfiedTime update modyfied time of HarborObject
// @Tips: This change will not be updated to the database,you need to update it explicitly.
func (ho *HarborObject) UpdateModyfiedTime() {

	ho.UpdateTime = JSONTimeNow()
}

// UpdateUploadTime  update create time of HarborObject
// @Tips: This change will not be updated to the database,you need to update it explicitly.
func (ho *HarborObject) UpdateUploadTime() {

	ho.UploadTime = JSONTimeNow()
}

// GetObjKey return object's identify key
func (ho *HarborObject) GetObjKey(b *Bucket) string {

	return fmt.Sprintf("%d_%d", b.ID, ho.ID)
}

// IsFile return true if it's object, return false if it's dir
func (ho *HarborObject) IsFile() bool {

	return ho.FileOrDir
}

// SetShared share object
// :param sh: 共享(True)或私有(False)
// :param days: 共享天数，0表示永久共享, <0表示不共享
func (ho *HarborObject) SetShared(share bool, days int) {

	if share {
		ho.IsShared = true // 共享
		now := time.Now()
		ho.SharedStartTime = now // 共享时间
		if days == 0 {
			ho.IsSharedLimit = false // 永久共享,没有共享时间限制
		} else if days < 0 {
			ho.IsShared = false // 私有
		} else {
			ho.IsSharedLimit = true                    // 有共享时间限制
			ho.SharedEndTime = now.AddDate(0, 0, days) // 共享终止时间
		}
	} else {
		ho.IsShared = false // 私有
	}
}

// IsSharedAndInSharedTime return true if the object is shared and within the effective sharing time
func (ho HarborObject) IsSharedAndInSharedTime() bool {

	// 对象是否是分享的
	if !ho.IsShared {
		return false
	}

	// 是否有分享时间限制
	if !ho.IsSharedLimit {
		return true
	}

	// 检查是否已过共享终止时间
	if ho.IsNowAfterSharedEndTime() {
		return false
	}

	return true
}

// IsNowAfterSharedEndTime return true if now after HarborObject's shared end time
// :return: True(已过共享终止时间)，False(未超时)
func (ho HarborObject) IsNowAfterSharedEndTime() bool {

	now := time.Now()
	return now.After(ho.SharedEndTime)
}

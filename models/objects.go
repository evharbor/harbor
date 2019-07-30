package models

import (
	"fmt"
	"time"
)

const (
	// BucketPublic 公有
	BucketPublic uint8 = 1
	// BucketPrivate 私有
	BucketPrivate uint8 = 2
)

// Bucket 存储桶结构
type Bucket struct {
	ID               uint64      `gorm:"PRIMARY_KEY;AUTO_INCREMENT;not null" json:"id"`
	Name             string      `gorm:"type:varchar(63);unique_index:uidx_name" json:"name"`
	User             UserProfile `gorm:"ForeignKey:UserID" json:"-"`                       //所属用户
	UserID           uint        `gorm:"column:user_id;index:idx_user_id;" json:"user_id"` //所属用户id
	CreatedTime      time.Time   `gorm:"column:created_time;" json:"created_time"`
	CollectionName   string      `gorm:"column:collection_name;type:varchar(50)" json:"-"`   //存储桶对应的表名
	AccessPermission uint8       `gorm:"column:access_permission;" json:"access_permission"` //访问权限
	SoftDelete       bool        `gorm:"column:soft_delete;" json:"-"`                       // True->删除状态
	ModifiedTime     time.Time   `gorm:"column:modyfied_time;" json:"-"`                     // 修改时间可以指示删除时间
	ObjsCount        uint32      `gorm:"column:objs_count;" json:"-"`                        //桶内对象的数量
	Size             uint64      `gorm:"column:size;" json:"-"`                              //桶内对象的总大小
	StatsTime        time.Time   `gorm:"column:stats_time;" json:"-"`                        //统计时间
}

// NewBucketDefault create a bucket initialized with default value
func NewBucketDefault() *Bucket {
	now := time.Now()
	return &Bucket{
		AccessPermission: BucketPrivate,
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

// HarborObject 对象结构
type HarborObject struct {
	ID              uint64    `gorm:"PRIMARY_KEY;AUTO_INCREMENT;not null" json:"id"`
	PathName        string    `gorm:"column:na;not null" json:"na"`                                             //全路径文件名或目录名
	FileOrDir       bool      `gorm:"column:fod;index:idx_fod_did;not null" json:"fod"`                         //True==文件，False==目录
	ParentID        uint64    `gorm:"column:did;index:idx_fod_did;unique_index:udx_did_name;not null" json:"-"` //父节点id
	Name            string    `gorm:"type:varchar(255);unique_index:udx_did_name;not null" json:"name"`         //文件名或目录名
	Size            uint64    `gorm:"cloumn:si;not null" json:"si"`                                             //文件大小, 字节数
	UploadTime      time.Time `gorm:"column:ult;not null" json:"ult"`                                           //文件的上传时间，或目录的创建时间
	UpdateTime      time.Time `gorm:"column:upt;not null" json:"upt"`                                           //修改时间
	DownloadCount   uint64    `gorm:"column:dlc;not null" json:"dlc"`                                           //该文件的下载次数，目录时dlc为0
	IsShared        bool      `gorm:"column:sh;not null" json:"sh"`                                             //为True，则文件可共享，为False，则文件不能共享
	ShareCode       string    `gorm:"column:shp;type:varchar(10);not null" json:"-"`                            //该文件的共享密码，目录时为空
	IsSharedLimit   bool      `gorm:"column:stl;default:true;not null" json:"-"`                                //True: 文件有共享时间限制; False: 则文件无共享时间限制
	SharedStartTime time.Time `gorm:"column:sst;not null" json:"-"`                                             //该文件的共享起始时间
	SharedEndTime   time.Time `gorm:"column:set;not null" json:"-"`                                             //该文件的共享终止时间
	SoftDeleted     bool      `gorm:"column:sds;not null" json:"-"`                                             //软删除,True->删除状态
}

// NewHarborObject create a harbor object
func NewHarborObject() *HarborObject {
	return &HarborObject{}
}

// NewHarborObjectDefault create a harbor object initialized with default value
func NewHarborObjectDefault() *HarborObject {
	now := time.Now()
	return &HarborObject{
		UploadTime:    now,
		UpdateTime:    now,
		IsSharedLimit: true,
		FileOrDir:     true,
	}
}

// NewHarborDirDefault create a harbor dir initialized with default value
func NewHarborDirDefault() *HarborObject {
	now := time.Now()
	return &HarborObject{
		UploadTime: now,
		UpdateTime: now,
		FileOrDir:  false,
	}
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

	ho.UpdateTime = time.Now()
}

// GetObjKey return object's identify key
func (ho *HarborObject) GetObjKey(b *Bucket) string {

	return fmt.Sprintf("%d_%d", b.ID, ho.ID)
}

// IsFile return true if it's object, return false if it's dir
func (ho *HarborObject) IsFile() bool {

	return ho.FileOrDir
}

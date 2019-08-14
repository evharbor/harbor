package models

import (
	"errors"
	"harbor/database"
	"strings"

	"github.com/jinzhu/gorm"
)

// Manager base manager for db operation
type Manager struct {
	dbAlias         string
	TableName       string
	db              *gorm.DB
	tx              *gorm.DB //transaction db
	isInTransaction bool     //是否开启了事务
}

// NewManager return a manager
func NewManager(dbAlias, tableName string) *Manager {

	return &Manager{
		dbAlias:   dbAlias,
		TableName: tableName,
	}
}

// GetDBAlias return db alias
func (m *Manager) GetDBAlias() string {

	if m.dbAlias == "" {
		return "default"
	}

	return m.dbAlias
}

func (m *Manager) getTableName() string {

	return m.TableName
}

// BeginTransaction start a transaction
func (m *Manager) BeginTransaction() {

	m.isInTransaction = true
	m.tx = nil
	m.GetDB()
}

// CommitTransaction Commit a transaction
func (m *Manager) CommitTransaction() error {

	tx := m.GetDB()
	if err := tx.Commit().Error; err != nil {
		return err
	}
	m.isInTransaction = false
	return nil
}

// RollbackTransaction Rollback a transaction
func (m *Manager) RollbackTransaction() error {

	tx := m.GetDB()
	if err := tx.Rollback().Error; err != nil {
		return err
	}
	m.isInTransaction = false
	return nil
}

// GetDB return *gorm.DB
func (m *Manager) GetDB() *gorm.DB {

	if m.db == nil {
		m.db = database.GetDB(m.GetDBAlias()).Table(m.getTableName())
	}

	// 是否开启了数据库事务
	if m.isInTransaction {
		if m.tx == nil {
			m.tx = m.db.Begin()
		}

		return m.tx
	}

	return m.db
}

// JoinPath return "a/b/c" if JoinPath("a", "b", "c")
func JoinPath(p ...string) string {

	var paths []string
	for _, path := range p {
		path := strings.Trim(path, "/")
		if path != "" {
			paths = append(paths, path)
		}
	}
	return strings.Join(paths, "/")
}

// HarborObjectManager manage HarborObject
type HarborObjectManager struct {
	Manager
	DirPath string
	ObjName string
	curDir  *HarborObject
}

// NewHarborObjectManager return manager for manage HarborObject
func NewHarborObjectManager(tableName, dirPath, objName string) *HarborObjectManager {

	if dirPath == "/" {
		dirPath = ""
	}
	return &HarborObjectManager{
		Manager: *NewManager("objs", tableName),
		DirPath: dirPath,
		ObjName: objName,
	}
}

// GetObjPathName return full path object name
func (m HarborObjectManager) GetObjPathName() string {

	return JoinPath(m.DirPath, m.ObjName)
}

// GetDirByPathName return dir filter by full dir name
// return:
//		dir, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (m HarborObjectManager) GetDirByPathName(dirPathName string) (dir *HarborObject, err error) {

	if m.DirPath == "" {
		m.curDir = &HarborObject{ID: 0}
		dir = m.curDir
		return
	}

	d := NewHarborObject()
	db := m.GetDB()
	if r := db.Where("fod = ? and na = ?", false, dirPathName).First(d); r.Error != nil {
		if r.RecordNotFound() {
			return
		}

		err = errors.New(r.Error.Error())
		return
	}
	dir = d
	return
}

// GetDirOrCreateUnderCurrent get or create dir under current dir, return it if no error
// return
//		dir, false, nil    if dir exists
//		dir, true, nil    if dir not exists, create new dir
//		nil, false, error  if error
func (m HarborObjectManager) GetDirOrCreateUnderCurrent(name string) (dir *HarborObject, created bool, err error) {

	did, err := m.GetCurDirID()
	if err != nil {
		return
	}

	d := NewHarborObject()
	db := m.GetDB()
	// r := db.Where(&HarborObject{ParentID: did, Name: name}).First(d)
	r := db.Where("did = ? and name = ?", did, name).First(d)
	// exists
	if r.Error == nil {
		dir = d
		created = false
		return
	}
	// error
	if !r.RecordNotFound() {
		err = r.Error
		return
	}
	// create new
	d = NewHarborDirDefault()
	d.ParentID = did
	d.Name = name
	d.PathName = JoinPath(m.DirPath, name)
	if r := db.Create(d); r.Error != nil {
		err = r.Error
		return
	}
	dir = d
	created = true
	return
}

// GetCurDir return dir if exists
// return:
//		dir, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (m *HarborObjectManager) GetCurDir() (dir *HarborObject, err error) {

	if m.curDir != nil {
		dir = m.curDir
		return
	}
	dir, err = m.GetDirByPathName(m.DirPath)
	if dir != nil {
		m.curDir = dir
	}
	return
}

// GetCurDirID return current dir id if no error
// return 0 if current dir is ""
// return error if current dir is not exists
func (m *HarborObjectManager) GetCurDirID() (id uint64, err error) {

	if m.curDir == nil {
		dir, e := m.GetCurDir()
		if e != nil {
			err = e
			return
		}
		if dir == nil {
			err = errors.New("parent directory is not found")
			return
		}
	}
	id = m.curDir.ID
	return
}

// ResetObjName reset object name
func (m *HarborObjectManager) ResetObjName(name string) {

	m.ObjName = name
}

// GetObjOrDirByDidName return HarborObject or dir filter by did and name
// return:
//		obj, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (m HarborObjectManager) GetObjOrDirByDidName(did uint64, name string) (obj *HarborObject, err error) {

	err = nil
	obj = nil

	object := NewHarborObject()
	db := m.GetDB()

	if r := db.Where("did = ? and name = ?", did, name).First(object); r.Error != nil {
		if r.RecordNotFound() {
			return
		}

		err = errors.New(r.Error.Error())
		return
	}

	obj = object
	return
}

// GetObjOrDirExists return HarborObject or dir if exists
// return:
//		obj, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (m HarborObjectManager) GetObjOrDirExists() (obj *HarborObject, err error) {

	did, e := m.GetCurDirID()
	if e != nil {
		err = e
		return
	}

	obj, err = m.GetObjOrDirByDidName(did, m.ObjName)

	return
}

// GetObjExists return HarborObject if exists
// return:
//		obj, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (m HarborObjectManager) GetObjExists() (obj *HarborObject, err error) {

	obj, err = m.GetObjOrDirExists()
	if err != nil {
		return
	}
	if obj == nil {
		return
	}
	if !obj.IsFile() {
		obj = nil
		return
	}

	return
}

// CreatObject create a new HarborObject
// return:
//		obj, nil: success
//		nil, error: have a error
func (m HarborObjectManager) CreatObject() (obj *HarborObject, err error) {

	obj = NewHarborObjectDefault()
	obj.PathName = m.GetObjPathName()
	obj.Name = m.ObjName
	obj.FileOrDir = true
	db := m.GetDB()
	if err = db.Create(obj).Error; err != nil {
		obj = nil
		return
	}
	return
}

// GetObjOrCreat return a HarborObject or create one
// return:
//		obj, true: not exists and create one
//		obj, false: exists
//		nil, false: have a error
func (m HarborObjectManager) GetObjOrCreat() (obj *HarborObject, created bool) {

	obj = nil
	created = false

	if object, err := m.GetObjExists(); err != nil {
		return
	} else if object != nil {
		obj = object
		return
	}

	object, err := m.CreatObject()
	if err != nil {
		return
	}
	obj = object
	created = true
	return
}

// SaveObject update all changes of object to database
func (m HarborObjectManager) SaveObject(obj *HarborObject) error {

	db := m.GetDB()
	if r := db.Save(&obj); r.Error != nil {
		return errors.New("failed to update object's metadata")
	}
	return nil
}

// DeleteObject delete a object
func (m HarborObjectManager) DeleteObject(obj *HarborObject) error {

	db := m.GetDB()
	if r := db.Delete(obj); r.Error != nil {
		if r.RecordNotFound() {
			return nil
		}
		return errors.New(r.Error.Error())
	}

	return nil
}

// DeleteDir delete a dir
func (m HarborObjectManager) DeleteDir(dir *HarborObject) error {

	return m.DeleteObject(dir)
}

// GetObjectsQuery return a gorm.DB that select all objs or subdirs under current dir
func (m HarborObjectManager) GetObjectsQuery() (query *gorm.DB, err error) {

	db := m.GetDB()
	did, e := m.GetCurDirID()
	if e != nil {
		err = e
		return
	}
	return db.Order("id desc").Where("did = ?", did), nil
}

// GetCurrentCount return count of subdir and objects under current dir
func (m HarborObjectManager) GetCurrentCount() (count int64, err error) {

	db := m.GetDB()
	did, e := m.GetCurDirID()
	if e != nil {
		err = e
		return
	}

	err = db.Where("did = ?", did).Count(&count).Error
	return
}

// IsCurrentDirEmpty Check whether the current directory is empty
func (m HarborObjectManager) IsCurrentDirEmpty() (bool, error) {

	count, err := m.GetCurrentCount()
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	return true, nil
}

// BucketManager manage buckets
type BucketManager struct {
	Manager
	Name string
	User *UserProfile
}

// NewBucketManager return manager for manage buckets
func NewBucketManager(bucketName string, user *UserProfile) *BucketManager {

	tableName := Bucket{}.TableName()
	return &BucketManager{
		Manager: *NewManager("default", tableName),
		Name:    bucketName,
		User:    user,
	}
}

// GetBucketByName return Bucket instance
// return:
//		*Bucket, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (bm BucketManager) GetBucketByName(name string) (*Bucket, error) {

	bucket := &Bucket{}
	db := bm.GetDB()
	if r := db.Where("name = ? AND soft_delete = ?", name, false).Find(&bucket); r.Error != nil {
		if r.RecordNotFound() {
			return nil, nil
		}

		return nil, errors.New(r.Error.Error())
	}
	return bucket, nil
}

// GetBucketByID return Bucket instance
// return:
//		*Bucket, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (bm BucketManager) GetBucketByID(id uint64) (*Bucket, error) {

	bucket := &Bucket{}
	db := bm.GetDB()
	if r := db.Where("id = ? AND soft_delete = ?", id, false).Find(&bucket); r.Error != nil {
		if r.RecordNotFound() {
			return nil, nil
		}

		return nil, errors.New(r.Error.Error())
	}
	return bucket, nil
}

// GetUserBucketByID return user's bucket instance by id
// return:
//		*Bucket, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (bm BucketManager) GetUserBucketByID(id uint64) (*Bucket, error) {

	bucket := &Bucket{}
	db := bm.GetDB()
	if r := db.Where(&Bucket{ID: id, UserID: bm.User.ID}).Where("soft_delete = ?", false).Find(&bucket); r.Error != nil {
		if r.RecordNotFound() {
			return nil, nil
		}

		return nil, errors.New(r.Error.Error())
	}
	return bucket, nil
}

// GetBucket return Bucket instance
// return:
//		*Bucket, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (bm BucketManager) GetBucket() (*Bucket, error) {

	name := bm.Name
	if name == "" {
		return nil, errors.New("Query bucket error: bucket name is empty string")
	}
	return bm.GetBucketByName(name)
}

// GetUserBucketByName return Bucket instance filter by user and bucket name
// return:
//		*Bucket, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (bm BucketManager) GetUserBucketByName(user *UserProfile, name string) (*Bucket, error) {

	bucket := &Bucket{}
	db := bm.GetDB()
	if r := db.Where("soft_delete = ?", false).Find(&bucket, Bucket{UserID: user.ID, Name: name}); r.Error != nil {
		if r.RecordNotFound() {
			return nil, nil
		}

		return nil, errors.New(r.Error.Error())
	}
	return bucket, nil
}

// GetUserBucket return current user's Bucket
// return:
//		*Bucket, nil: exists and no error
//		nil, nil: not exists and no error
//		nil, error: have a error
func (bm BucketManager) GetUserBucket() (*Bucket, error) {

	user := bm.User
	if user == nil {
		return nil, errors.New("bucket manager's field 'User' is invalid")
	}
	bucketName := bm.Name

	return bm.GetUserBucketByName(user, bucketName)
}

// CreateBucketByName create an bucket instance and save to database,return the instance if no error
// return:
//		*Bucket, nil: exists and no error
//		nil, error: have a error
func (bm BucketManager) CreateBucketByName(name string, user *UserProfile) (*Bucket, error) {

	bucket := NewBucketDefault()
	bucket.Name = name
	bucket.UserID = user.ID
	db := bm.GetDB()
	if r := db.Create(bucket); r.Error != nil {
		return nil, errors.New(r.Error.Error())
	}

	return bucket, nil
}

// CreateBucket create an bucket instance and save to database,return the instance if no error
// return:
//		*Bucket, nil: exists and no error
//		nil, error: have a error
func (bm BucketManager) CreateBucket() (*Bucket, error) {

	return bm.CreateBucketByName(bm.Name, bm.User)
}

// CreateObjsTable create table for bucket
func (bm BucketManager) CreateObjsTable(bucket *Bucket) error {

	db := database.GetDB("objs")
	tableName := bucket.GetObjsTableName()

	exists := db.HasTable(tableName)
	if exists {
		return nil
	}

	if r := db.Table(tableName).CreateTable(NewHarborObject()); r.Error != nil {
		// fmt.Println(r.)
		return errors.New(r.Error.Error())
	}

	return nil
}

// DeleteBucket delete a bucket
func (bm BucketManager) DeleteBucket(bucket *Bucket) error {

	db := bm.GetDB()
	if r := db.Delete(bucket); r.Error != nil {
		if r.RecordNotFound() {
			return nil
		}
		return errors.New(r.Error.Error())
	}

	return nil
}

// SoftDeleteBucket soft delete a bucket
func (bm BucketManager) SoftDeleteBucket(bucket *Bucket) error {

	db := bm.GetDB()
	b := Bucket{SoftDelete: true}
	b.UpdateModyfiedTime()
	b.SetSoftDeleteName()
	if r := db.Model(bucket).Updates(b); r.Error != nil {
		return errors.New(r.Error.Error())
	}

	return nil
}

// SoftDeleteUserBucketsByIDs only soft delete user's buckets by ids
func (bm BucketManager) SoftDeleteUserBucketsByIDs(ids []string) error {

	var buckets []Bucket
	db := bm.GetDB()
	if r := db.Where("user_id = ? AND id IN (?)", bm.User.ID, ids).Find(&buckets); r.Error != nil {
		if r.RecordNotFound() {
			return nil
		}
		return errors.New(r.Error.Error())
	}
	for _, b := range buckets {
		if !b.IsSoftDelete() {
			if err := bm.SoftDeleteBucket(&b); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetUserBucketsQuery return user's bucket list query db
func (bm BucketManager) GetUserBucketsQuery() *gorm.DB {

	db := bm.GetDB()
	user := bm.User
	return db.Where("user_id = ? AND soft_delete = ?", user.ID, false).Order("id desc")
}

// SetUserBucketsAccessByIDs set user's buckets access permission by ids
func (bm BucketManager) SetUserBucketsAccessByIDs(ids []string, public bool) error {

	bucket := Bucket{}
	bucket.UpdateModyfiedTime()
	if public {
		bucket.AccessPermission = TypeBucketPermission(BucketPublic)
	} else {
		bucket.AccessPermission = TypeBucketPermission(BucketPrivate)
	}
	db := bm.GetDB()
	if r := db.Where("user_id = ? AND id IN (?)", bm.User.ID, ids).Updates(bucket); r.Error != nil {
		return errors.New(r.Error.Error())
	}

	return nil
}

// BucketRename rename a bucket
func (bm BucketManager) BucketRename(bucket *Bucket, rename string) error {

	db := bm.GetDB()
	if bucket.Name == rename {
		return nil
	}
	if r := db.Model(bucket).Update("name", rename); r.Error != nil {
		return errors.New(r.Error.Error())
	}
	return nil
}

package storages

import (
	"harbor/config"
	"harbor/utils/storages/filesystem"
	"harbor/utils/storages/radosio"
	"os"
	"path/filepath"
)

var configs = config.GetConfigs()

// NewCephHarborObject return CephHarborObject manage object's data in ceph
func NewCephHarborObject(objID string, objSize uint64) *radosio.CephHarborObject {

	c := configs.CephRados

	r := &radosio.CephHarborObject{}
	r.SetCephConfig(c.ClusterName, c.Username, c.ConfFile, c.KeyringFile, c.PoolName)
	r.ResetObjIDAndSize(objID, objSize)

	return r
}

// NewFileStorage return a filestorage
func NewFileStorage(filename string) *filesystem.FileStorage {

	// 目录路径不存在存在则创建
	dirPath := filepath.Clean(getUploadPath())
	if exist, _ := DirExists(dirPath); !exist {
		os.MkdirAll(dirPath, os.ModeDir)
	}
	return &filesystem.FileStorage{
		Filename:   filename,
		UploadPath: dirPath,
	}
}

func getUploadPath() string {

	return filepath.Join(configs.BaseDir, "upload")
}

// DirExists 目录是否存在
// return:
// 		true and nil,文件夹存在
//		false and nil, 不存在
// 		error != nil ,则不确定是否存在
func DirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		if fi.Mode().IsDir() {
			return true, nil
		}
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

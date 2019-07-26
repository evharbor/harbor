package storages

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// FileStorage manage write or read file on local file system
type FileStorage struct {
	filename   string
	uploadPath string
}

// NewFileStorage return a filestorage
func NewFileStorage(filename string) *FileStorage {

	// 目录路径不存在存在则创建
	dirPath := filepath.Clean(getUploadPath())
	if exist, _ := DirExists(dirPath); !exist {
		os.MkdirAll(dirPath, os.ModeDir)
	}
	return &FileStorage{
		filename:   filename,
		uploadPath: dirPath,
	}
}

// GetFilename return file path name
func (fs FileStorage) GetFilename() string {

	if fs.uploadPath == "" {
		fs.uploadPath = getUploadPath()
	}
	return filepath.Join(fs.uploadPath, fs.filename)
}

// WriteFile write a file-like to a file
func (fs FileStorage) WriteFile(offset int64, file *multipart.FileHeader) error {

	inputFile, err := file.Open()
	if err != nil {
		return err
	}
	defer inputFile.Close()

	fileName := fs.GetFilename()
	saveFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer saveFile.Close()

	size := file.Size
	var fileOffset int64 // 文件已写入的偏移量
	for {
		// 文件是否已完全写入
		if fileOffset == size {
			break
		}

		s, err := inputFile.Seek(fileOffset, os.SEEK_SET)
		if err != nil {
			return err
		}
		if s != offset {
			return errors.New("seek文件偏移量错误")
		}

		bufSize := 5 * 1024 * 1024
		chunk := make([]byte, bufSize)
		readSize, err := inputFile.Read(chunk)
		if err != nil {
			return err
		}
		err = fs.writeChunk(saveFile, offset+fileOffset, chunk[0:readSize])
		if err != nil {
			return err
		}

		fileOffset += int64(readSize) // 更新已写入大小
	}
	return nil
}

// Write write bytes to a file
func (fs FileStorage) Write(offset int64, data []byte) error {

	fileName := fs.GetFilename()
	saveFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer saveFile.Close()

	return fs.writeChunk(saveFile, offset, data)
}

// WriteChunk write bytes to a file
func (fs FileStorage) writeChunk(file *os.File, offset int64, data []byte) error {

	s, err := file.Seek(offset, os.SEEK_SET)
	if err != nil {
		return err
	}
	if s != offset {
		return errors.New("seek文件偏移量错误")
	}
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// Read read bytes from a file
func (fs FileStorage) Read(offset int64, size int32) (data []byte, err error) {

	fileName := fs.GetFilename()
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 文件偏移量设置
	s, err := file.Seek(offset, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	if s != offset {
		return nil, errors.New("seek文件偏移量错误")
	}

	buf := make([]byte, size)
	_, err = io.ReadFull(file, buf)
	if (err != nil) && (err != io.ErrUnexpectedEOF) {
		return
	}

	data = buf
	err = nil
	return
}

//FileSize return file's size, return 0 if error
func (fs FileStorage) FileSize() int64 {

	fileName := fs.GetFilename()
	file, err := os.Open(fileName)
	if err != nil {
		return 0
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}

// Delete remove a file
func (fs FileStorage) Delete() error {

	fileName := fs.GetFilename()
	return os.Remove(fileName)
}

// StepWriteFunc return
func (fs FileStorage) StepWriteFunc() (StepWriteFunc, error) {

	var fileSize int64
	fileName := fs.GetFilename()
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize = fileInfo.Size()
	sr, err := NewFileStepRead(file, fileSize, 0, 5*1024*1024)
	if err != nil {
		return nil, err
	}
	return sr.StepWrite, nil
}

// StepWriteFunc defines the handler used by gin Stream() as return value.
type StepWriteFunc func(io.Writer) bool

// Stepwisable 可分步
type Stepwisable interface {
	io.Reader
	io.Seeker
	io.Closer
}

// FileStepRead 分步读文件
type FileStepRead struct {
	Offset   int64
	SizeStep uint
	Size     int64
	file     Stepwisable
	buf      []byte
}

// NewFileStepRead return FileStepRead instance
func NewFileStepRead(file Stepwisable, size, offset int64, sizeStep uint) (*FileStepRead, error) {

	// 文件偏移量设置
	s, err := file.Seek(offset, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	if s != offset {
		return nil, errors.New("seek文件偏移量错误")
	}

	return &FileStepRead{
		file:     file,
		Size:     size,
		Offset:   offset,
		SizeStep: sizeStep,
		buf:      make([]byte, sizeStep),
	}, nil
}

// StepWrite write a file by step, return false after writing
func (fsr *FileStepRead) StepWrite(w io.Writer) bool {

	readSize, err := fsr.file.Read(fsr.buf)
	if err == io.EOF {
		fsr.file.Close()
		return false
	}
	// error
	if (err != nil) && (err != io.ErrUnexpectedEOF) {
		fsr.file.Close()
		return false
	}
	writeSize, err := w.Write(fsr.buf[0:readSize])
	if err != nil {
		return true
	}
	fsr.Offset += int64(writeSize)
	return true
}

// GetCurrentPath return executive file's path
func GetCurrentPath() (string, error) {
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return path, nil
	// if runtime.GOOS == "windows" {
	// 	path = strings.Replace(path, "\\", "/", -1)
	// }
	// i := strings.LastIndex(path, "/")
	// if i < 0 {
	// 	return "", errors.New(`Can't find "/" or "\".`)
	// }
	// return string(path[0 : i+1]), nil
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

func getUploadPath() string {
	baseDir, err := GetCurrentPath()
	if err != nil {
		baseDir = "."
	}
	return filepath.Join(baseDir, "upload")
}

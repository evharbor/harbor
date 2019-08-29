package filesystem

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// FileStorage manage write or read file on local file system
type FileStorage struct {
	Filename   string
	UploadPath string
}

// GetFilename return file path name
func (fs FileStorage) GetFilename() string {

	return filepath.Join(fs.UploadPath, fs.Filename)
}

// WriteFile write a file-like to a file
func (fs FileStorage) WriteFile(offset int64, file *multipart.FileHeader) error {

	inputFile, err := file.Open()
	if err != nil {
		return err
	}
	defer inputFile.Close()

	fileName := fs.GetFilename()
	saveFile, err := fs.OpenOrCreateFile(fileName)
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
		if s != fileOffset {
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

// OpenOrCreateFile open a file or create it if it is not exists
func (fs FileStorage) OpenOrCreateFile(filename string) (*os.File, error) {

	return os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
}

// Write write bytes to a file
func (fs FileStorage) Write(offset int64, data []byte) error {

	fileName := fs.GetFilename()
	saveFile, err := fs.OpenOrCreateFile(fileName)
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
	err := os.Remove(fileName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// StepWriteFunc return
func (fs FileStorage) StepWriteFunc(offset, end int64) (StepWriteFunc, error) {

	fileName := fs.GetFilename()
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	sr, err := NewFileStepRead(file, offset, end, 5*1024*1024)
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
	offset   int64 // read offset
	start    int64 // read start offset
	end      int64 // read end offset(containing current offset)
	sizeStep uint  // read size per step
	file     Stepwisable
	buf      []byte
}

// NewFileStepRead return FileStepRead instance
func NewFileStepRead(file Stepwisable, start, end int64, sizeStep uint) (*FileStepRead, error) {

	offset := start

	// 文件偏移量设置
	s, err := file.Seek(offset, os.SEEK_SET)
	if err != nil {
		return nil, err
	}
	if s != offset {
		return nil, errors.New("seek文件偏移量错误")
	}

	if start < 0 || start > end {
		return nil, errors.New("参数start或end值无效")
	}

	return &FileStepRead{
		file:     file,
		start:    start,
		end:      end,
		offset:   offset,
		sizeStep: sizeStep,
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

	// check whether out of Read Range
	if end := fsr.end + 1; (fsr.offset + int64(readSize)) > end {
		readSize = int(end - fsr.offset)
	}

	writeSize, err := w.Write(fsr.buf[0:readSize]) // slice不含结束下标
	if err != nil {
		return true
	}
	// next read start offset
	fsr.offset += int64(writeSize)
	// read end
	if fsr.offset > fsr.end {
		fsr.file.Close()
		return false
	}
	return true
}

// GetCurrentPath return executive file's path
func GetCurrentPath() (string, error) {
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return path, nil
}

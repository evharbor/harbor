package radosio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"os"

	"github.com/ceph/go-ceph/rados"
)

// SizePerRadosObj 每个rados object 最大2Gb
const SizePerRadosObj int64 = 2147483648

// 构造对象part对应的id
// :param obj_id: 对象id
// :param part_num: 对象part的编号
// :return: string
func buildPartID(objID string, partNum uint) string {

	// 第一个part id等于对象id
	if partNum == 0 {
		return objID
	}

	return fmt.Sprintf("%s_%d", objID, partNum)
}

type writeTaskStruct struct {
	partID     string
	offset     int64
	sliceStart int64 // slice start index
	sliceEnd   int64 // slice end index
}

// 分析对象写入操作具体写入任务, 即对象part的写入操作
// 注：此函数实现基于下面情况考虑，当不满足以下情况时，请重新实现此函数:
// 	由于每个part(rados对象)比较大，每次写入数据相对较小，即写入最多涉及到两个part。
// :param objID: 对象id
// :param offset: 数据写入的偏移量
// :param bytesLen: 要写入的bytes数组长度
// :return:
// 	[writeTaskStruct{partID, offset, sliceStart, sliceEnd)}, ]
// 	列表每项为一个writeTaskStruct结构体，依次为涉及到的对象part的id，数据写入part的偏移量，数据切片的前索引，数据切片的后索引;
func writePartTasks(objID string, offset, bytesLen int64) ([]writeTaskStruct, error) {

	if offset < 0 || bytesLen < 0 {
		return nil, errors.New("input parameter should not be less than 0")
	}

	if bytesLen > SizePerRadosObj {
		return nil, errors.New("write or read bytes size too long")
	}

	startPartNum := int(offset / SizePerRadosObj)
	endPartNum := int(math.Ceil(float64(offset+bytesLen)/float64(SizePerRadosObj))) - 1 // 向上取整数-1
	startPartOffset := offset % SizePerRadosObj

	// 要写入的数据在1个part上
	if startPartNum == endPartNum {
		partID := buildPartID(objID, uint(startPartNum))
		tasks := []writeTaskStruct{
			writeTaskStruct{
				partID:     partID,
				offset:     startPartOffset,
				sliceStart: 0,
				sliceEnd:   bytesLen,
			},
		}

		return tasks, nil
	}

	// 要写入的数据在2个part上
	startPartID := buildPartID(objID, uint(startPartNum))
	endPartID := buildPartID(objID, uint(endPartNum))
	sliceIndex := SizePerRadosObj - startPartOffset

	tasks := []writeTaskStruct{
		writeTaskStruct{
			partID:     startPartID,
			offset:     startPartOffset,
			sliceStart: 0,
			sliceEnd:   sliceIndex,
		},
		writeTaskStruct{
			partID:     endPartID,
			offset:     0,
			sliceStart: sliceIndex,
			sliceEnd:   bytesLen,
		},
	}

	return tasks, nil
}

type readTaskStruct struct {
	partID  string
	offset  int64
	readLen int64
}

// :param objID: 对象id
// :param offset: 读取对象的偏移量
// :param bytesLen: 读取字节长度
// :return:
// 	[readTaskStruct{partID, offset, readLen}, ]
// 	slice每项为一个readTaskStruct，依次为涉及到的对象part的id，从part读取数据的偏移量，读取数据长度
func readPartTasks(objID string, offset, bytesLen int64) ([]readTaskStruct, error) {

	var rTasks []readTaskStruct

	wTasks, err := writePartTasks(objID, offset, bytesLen)
	if err != nil {
		return nil, err
	}
	for _, t := range wTasks {
		task := readTaskStruct{
			partID:  t.partID,
			offset:  t.offset,
			readLen: t.sliceEnd - t.sliceStart,
		}
		rTasks = append(rTasks, task)
	}
	return rTasks, nil
}

// 每个HarborObject对象可能有多个部分part(rados对象)组成
//     OBJ(part0, part1, part2, ...)
//     part0 id == objID;  partN id == {objID}_{N}
func buildHarborObjectParts(objID string, objSize uint64) []string {

	lastPartNum := int(math.Ceil(float64(objSize)/float64(SizePerRadosObj))) - 1
	if lastPartNum < 0 {
		lastPartNum = 0
	}

	parts := make([]string, lastPartNum+1)
	for i := 0; i <= lastPartNum; i++ {
		parts[i] = buildPartID(objID, uint(i))
	}

	return parts
}

// class CephClusterCommand(dict):
//     '''
//     执行ceph 命令
//     '''

//     def __init__(self, cluster, prefix, format='json', **kwargs):
//         dict.__init__(self)
//         kwargs['prefix'] = prefix
//         kwargs['format'] = format
//         try:
//             ret, buf, err = cluster.mon_command(json.dumps(kwargs), '', timeout=5)
//         except rados.Error as e:
//             self['err'] = str(e)
//         else:
//             if ret != 0:
//                 self['err'] = err
//             else:
//                 self.update(json.loads(buf))

// RadosAPI ceph cluster rados对象接口封装
type RadosAPI struct {
	clusterName string
	userName    string
	poolName    string
	confFile    string
	keyringFile string
	conn        *rados.Conn
}

// NewRadosAPI return *RadosAPI
func NewRadosAPI(clusterName, userName, poolName, confFile, keyringFile string) (*RadosAPI, error) {

	if exists, err := FileExists(confFile); err != nil || !exists {
		return nil, errors.New("配置文件路径不存在")
	}
	if keyringFile != "" {
		if exists, err := FileExists(keyringFile); err != nil || !exists {
			return nil, errors.New("keyring配置文件路径不存在")
		}
	}

	return &RadosAPI{
		clusterName: clusterName,
		userName:    userName,
		poolName:    poolName,
		confFile:    confFile,
		keyringFile: keyringFile,
	}, nil

}

// newConn return a connect to ceph cluster
func (r RadosAPI) newConn() (*rados.Conn, error) {

	conn, err := rados.NewConnWithClusterAndUser(r.clusterName, r.userName)
	if err != nil {
		return nil, err
	}

	err = conn.ReadConfigFile(r.confFile)
	if err != nil {
		return nil, errors.New("read ceph config file error")
	}

	err = conn.Connect()
	if err != nil {
		return nil, errors.New("connect to ceph cluster error")
	}

	return conn, nil
}

// Close the connection to CEPH cluster
func (r *RadosAPI) Close() error {

	if r.conn != nil {
		r.conn.Shutdown()
	}
	return nil
}

// GetConn return a connect to ceph cluster
func (r *RadosAPI) GetConn() (*rados.Conn, error) {

	if r.conn != nil {
		return r.conn, nil
	}

	conn, err := r.newConn()
	if err != nil {
		return nil, err
	}
	r.conn = conn
	return r.conn, nil
}

// Write write data to a HarborObject
// :param objID: 对象id
// :param offset: 数据写入偏移量
// :param data: 数据，bytes
func (r RadosAPI) Write(objID string, offset uint64, data []byte) error {

	tasks, err := writePartTasks(objID, int64(offset), int64(len(data)))
	if err != nil {
		return err
	}
	conn, err := r.GetConn()
	if err != nil {
		return err
	}

	ioctx, err := conn.OpenIOContext(r.poolName)
	if err != nil {
		return errors.New("error when openIOContext:" + err.Error())
	}
	defer ioctx.Destroy()

	for _, t := range tasks {
		err := ioctx.Write(t.partID, data[t.sliceStart:t.sliceEnd], uint64(t.offset))
		if err != nil {
			return errors.New("error when write to rados:" + err.Error())
		}
	}

	return nil
}

// 从rados对象指定偏移量开始读取指定长度的字节数据
// :param ioctx: 输入/输出上下文
// :param radosID: rados id
// :param offset: 对象偏移量
// :param readSize: 要读取的字节长度
func (r RadosAPI) radosRead(ioctx *rados.IOContext, radosID string, offset, readSize uint64) ([]byte, error) {

	buf := make([]byte, readSize)
	_, err := ioctx.Read(radosID, buf, offset)
	if err != nil {
		// rados对象不存在，构造一个指定长度的bytes
		if err == rados.RadosErrorNotFound {
			return buf, nil
		}

		return nil, err
	}

	// 读取数据不足，补0
	// if rLen < readSize{}

	return buf, nil
}

// Read read bytes from a HarborObject
// :param objID: 对象id
// :param offset: 数据读取偏移量
// :param readSize: 读取数据byte大小
// :return
//		nil,error
//		[]byte, nil
func (r RadosAPI) Read(objID string, offset, readSize uint64) ([]byte, error) {

	if offset < 0 || readSize <= 0 {
		return []byte{}, nil
	}

	tasks, err := readPartTasks(objID, int64(offset), int64(readSize))
	if err != nil {
		return nil, err
	}
	conn, err := r.GetConn()
	if err != nil {
		return nil, err
	}

	ioctx, err := conn.OpenIOContext(r.poolName)
	if err != nil {
		return nil, errors.New("error when openIOContext:" + err.Error())
	}
	defer ioctx.Destroy()

	// 要读取的数据在一个rados对象上
	if len(tasks) == 1 {
		t := tasks[0]
		return r.radosRead(ioctx, t.partID, uint64(t.offset), uint64(t.readLen))
	}

	var buf bytes.Buffer
	for _, t := range tasks {
		data, err := r.radosRead(ioctx, t.partID, uint64(t.offset), uint64(t.readLen))
		if err != nil {
			return nil, err
		}
		_, err = buf.Write(data)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Delete a HarborObject
// :param objID: 对象id
// :param objSize: 对象大小
func (r RadosAPI) Delete(objID string, objSize uint64) error {

	conn, err := r.GetConn()
	if err != nil {
		return err
	}
	ioctx, err := conn.OpenIOContext(r.poolName)
	if err != nil {
		return errors.New("error when openIOContext:" + err.Error())
	}
	defer ioctx.Destroy()

	partIDs := buildHarborObjectParts(objID, objSize)
	for _, id := range partIDs {
		// NotFound == deleted, try again when error
		if err := ioctx.Delete(id); err != nil && err != rados.RadosErrorNotFound {
			if err := ioctx.Delete(id); err != nil && err != rados.RadosErrorNotFound {
				return err
			}
		}
	}

	return nil
}

// GetClusterStats return ceph cluster stats information
func (r RadosAPI) GetClusterStats() (rados.ClusterStat, error) {

	conn, err := r.GetConn()
	if err != nil {
		return rados.ClusterStat{}, err
	}

	return conn.GetClusterStats()
}

// MonCommand sends a command to one of the monitors
func (r RadosAPI) MonCommand(args []byte) (buffer []byte, info string, err error) {

	conn, err := r.GetConn()
	if err != nil {
		return []byte{}, "", err
	}
	return conn.MonCommand(args)
}

//     def mgr_command(self, prefix, format='json', **kwargs):
//         '''
//         :return: io status string
//         :raises: class:`RadosError`
//         '''
//         cluster = self.get_cluster()
//         kwargs['prefix'] = prefix
//         kwargs['format'] = format
//         try:
//             ret, buf, err = cluster.mgr_command(json.dumps(kwargs), '', timeout=5)
//         except rados.Error as e:
//             raise RadosError(str(e), errno=e.errno)
//         else:
//             if ret != 0:
//                 raise RadosError(err)
//             else:
//                 return err

//     def get_ceph_io_status(self):
//         '''
//         :return: {
//                 'bw_rd': 0.0,   # Kb/s ,float
//                 'bw_wr': 0.0,   # Kb/s ,float
//                 'bw': 0.0       # Kb/s ,float
//                 'op_rd': 0,     # op/s, int
//                 'op_wr': 0,     # op/s, int
//                 'op': 0,        # op/s, int
//             }
//         :raises: class:`RadosError`
//         '''
//         try:
//             s = self.mgr_command(prefix='iostat')
//         except RadosError as e:
//             raise e
//         return self.parse_io_str(io_str=s)

//     def parse_io_str(self, io_str:str):
//         '''
//         :param io_str:  '  | 1623 KiB/s |   20 KiB/s | 1643 KiB/s |          2 |          1 |          4 |  '
//         :return: {
//                 'bw_rd': 0.0,   # Kb/s ,float
//                 'bw_wr': 0.0,   # Kb/s ,float
//                 'bw': 0.0       # Kb/s ,float
//                 'op_rd': 0,     # op/s, int
//                 'op_wr': 0,     # op/s, int
//                 'op': 0,        # op/s, int
//             }
//         '''
//         def to_kb(value, unit):
//             u = unit[0]
//             if u == 'b':
//                 value = value / 1024
//             elif u == 'k':
//                 value = value
//             elif u == 'm':
//                 value = value * 1024
//             elif u == 'g':
//                 value = value * 1024 * 1024

//             return value

//         keys = ['bw_rd', 'bw_wr', 'bw', 'op_rd', 'op_wr', 'op']
//         data = {}
//         items = io_str.strip(' |').split('|')
//         for i, item in enumerate(items):
//             item = item.strip()
//             item = item.lower()
//             units = item.split(' ')
//             try:
//                 val = float(units[0])
//             except:
//                 data[keys[i]] = 0
//                 continue

//             # iobw
//             if item.endswith('b/s'):
//                 val = to_kb(val, units[-1])
//             # iops
//             else:
//                 val = int(val)

//             data[keys[i]] = val

//         return data

// CephHarborObject 对象操作接口封装
type CephHarborObject struct {
	clusterName string
	userName    string
	confFile    string
	keyringFile string
	poolName    string
	objID       string
	objSize     uint64
	radosAPI    *RadosAPI
}

// SetCephConfig set ceph settings
func (cho *CephHarborObject) SetCephConfig(clusterName, userName, confFile, keyringFile, poolName string) {

	cho.clusterName = clusterName
	cho.userName = userName
	cho.confFile = confFile
	cho.keyringFile = keyringFile
	cho.poolName = poolName
}

// GetRados return rados api
func (cho *CephHarborObject) GetRados() (*RadosAPI, error) {

	if cho.radosAPI == nil {
		api, err := NewRadosAPI(cho.clusterName, cho.userName, cho.poolName, cho.confFile, cho.keyringFile)
		if err != nil {
			return nil, err
		}
		cho.radosAPI = api
	}

	return cho.radosAPI, nil
}

// GetObjSize return size of HarborObject
func (cho *CephHarborObject) GetObjSize() uint64 {

	return cho.objSize
}

// ResetObjIDAndSize reset an HarborObject id and size
func (cho *CephHarborObject) ResetObjIDAndSize(objID string, objSize uint64) {

	cho.objID = objID
	cho.objSize = objSize
}

// Read read 'size' bytes start at 'offset'
// 从指定字节偏移位置读取指定长度的数据块
// :param offset: 偏移位置
// :param size: 读取长度
// :return:
//		[]byte{}, nil : end of object,读到了对象结尾
func (cho CephHarborObject) Read(offset uint64, size uint) (data []byte, err error) {

	var readSize = size
	// 读取偏移量超出对象大小，直接返回空bytes
	objSize := cho.GetObjSize()
	if offset >= objSize {
		data = []byte{}
		return
	}

	// 读取数据超出对象大小，计算可读取大小
	if (offset + uint64(size)) > objSize {
		readSize = uint(objSize - offset)
	}

	api, err := cho.GetRados()
	if err != nil {
		data = []byte{}
		return
	}

	data, err = api.Read(cho.objID, offset, uint64(readSize))
	return
}

// Write data start at offset to harborObject
// :param data: data will be writed
// :param offset: write start at
func (cho *CephHarborObject) Write(data []byte, offset uint64) error {

	chunkSize := 20 * 1024 * 1024 // 20MB
	dataLen := len(data)
	start := 0
	end := start + chunkSize

	api, err := cho.GetRados()
	if err != nil {
		return err
	}

	for {
		if start >= dataLen {
			break
		}
		if end > dataLen {
			end = dataLen
		}
		chunk := data[start:end]
		err := api.Write(cho.objID, offset, chunk)
		if err != nil {
			return err
		}
		start = start + len(chunk)
		end = start + chunkSize
	}

	// object's size maybe enlarge after write some data
	s := offset + uint64(dataLen)
	if s > cho.objSize {
		cho.objSize = s
	}

	return nil
}

// WriteFile write a file-like to a HarborObject
func (cho CephHarborObject) WriteFile(offset int64, file *multipart.FileHeader) error {

	var fileOffset, bufSize int64 // 文件已写入的偏移量
	bufSize = 10 * 1024 * 1024    //10Mb
	size := file.Size
	if size < bufSize {
		bufSize = size
	}
	chunk := make([]byte, bufSize)

	inputFile, err := file.Open()
	if err != nil {
		return err
	}
	defer inputFile.Close()

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

		readSize, err := inputFile.Read(chunk)
		if err != nil {
			return err
		}
		err = cho.Write(chunk[0:readSize], uint64(offset+fileOffset))
		if err != nil {
			return err
		}

		fileOffset += int64(readSize) // 更新已写入大小
	}
	return nil
}

// Delete an HarborObject
func (cho CephHarborObject) Delete() error {

	objSize := cho.GetObjSize()
	api, err := cho.GetRados()
	if err != nil {
		return err
	}

	return api.Delete(cho.objID, objSize)
}

// Close connetion to ceph
func (cho CephHarborObject) Close() error {

	if cho.radosAPI != nil {
		return cho.radosAPI.Close()
	}

	return nil
}

// StepWriteFunc return
// :param offset: 读起始偏移量
// :param end: 读结束偏移量(包含)
func (cho *CephHarborObject) StepWriteFunc(offset, end uint64) (StepWriteFunc, error) {

	sr, err := NewObjStepRead(cho, offset, end, 5*1024*1024)
	if err != nil {
		return nil, err
	}
	return sr.StepWrite, nil
}

// GetClusterStats return ceph cluster stats
func (cho CephHarborObject) GetClusterStats() (rados.ClusterStat, error) {

	api, err := cho.GetRados()
	if err != nil {
		return rados.ClusterStat{}, err
	}
	return api.GetClusterStats()
}

//     def get_ceph_io_status(self):
//         '''
//         :return:
//             success: True, {
//                     'bw_rd': 0.0,   # Kb/s ,float
//                     'bw_wr': 0.0,   # Kb/s ,float
//                     'bw': 0.0       # Kb/s ,float
//                     'op_rd': 0,     # op/s, int
//                     'op_wr': 0,     # op/s, int
//                     'op': 0,        # op/s, int
//                 }
//             error: False, err:tsr
//         '''
//         try:
//             rados = self.get_rados_api()
//             status = rados.get_ceph_io_status()
//         except RadosError as e:
//             return False, str(e)

//         return True, status

// StepWriteFunc defines the handler used by gin Stream() as return value.
type StepWriteFunc func(io.Writer) bool

// ObjStepRead 分步读对象
type ObjStepRead struct {
	offset   uint64 // read offset
	start    uint64 // read start offset
	end      uint64 // read end offset(containing current offset)
	sizeStep uint   // read size per step
	obj      *CephHarborObject
}

// NewObjStepRead return ObjStepRead instance
func NewObjStepRead(obj *CephHarborObject, start, end uint64, sizeStep uint) (*ObjStepRead, error) {

	offset := uint64(start)

	if end > obj.GetObjSize() {
		return nil, errors.New("invalid input param, the reading range is beyond the size of the object")
	}

	return &ObjStepRead{
		obj:      obj,
		start:    start,    // read start offset
		end:      end,      // read end offset(containing current offset)
		offset:   offset,   // read offset
		sizeStep: sizeStep, // read size per step
	}, nil
}

// StepWrite write a HarborObject by step, return false after writing
func (osr *ObjStepRead) StepWrite(w io.Writer) bool {

	data, err := osr.obj.Read(osr.offset, osr.sizeStep)
	if err != nil {
		osr.obj.Close()
		return false
	}
	readSize := uint64(len(data))
	if readSize == 0 {
		osr.obj.Close()
		return false
	}

	// check whether out of Read Range
	if end := osr.end + 1; (osr.offset + readSize) > end {
		readSize = end - osr.offset
	}

	writeSize, err := w.Write(data)
	if err != nil {
		return true
	}
	// next read start offset
	osr.offset += uint64(writeSize)
	// read end
	if osr.offset > osr.end {
		osr.obj.Close()
		return false
	}
	return true
}

// FileExists 文件是否存在
// return:
// 		true and nil,文件夹存在
//		false and nil, 不存在
// 		error != nil ,则不确定是否存在
func FileExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		if fi.Mode().IsRegular() {
			return true, nil
		}
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

package fdfs

import (
	"fmt"
	"net"
	"sync"
)

type Config struct {
	Tracker []string
	MaxConn int
}

type Client struct {
	trackerPool     map[string]*connPool
	storagePool     map[string]*connPool
	storagePoolLock *sync.RWMutex
	config          *Config
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("NIL Config")
	}
	if len(cfg.Tracker) == 0 {
		return nil, fmt.Errorf("NIL Tracker")
	}
	if cfg.MaxConn == 0 {
		return nil, fmt.Errorf("Zero MaxConn")
	}
	cli := &Client{
		config:          cfg,
		storagePoolLock: new(sync.RWMutex),
	}
	cli.trackerPool = make(map[string]*connPool)
	cli.storagePool = make(map[string]*connPool)

	for _, addr := range cfg.Tracker {
		trackerPool, err := newConnPool(addr, cfg.MaxConn)
		if err != nil {
			return nil, err
		}
		cli.trackerPool[addr] = trackerPool
	}

	return cli, nil
}

func (cli *Client) Destory() {
	if cli == nil {
		return
	}
	for _, pool := range cli.trackerPool {
		pool.Destory()
	}
	for _, pool := range cli.storagePool {
		pool.Destory()
	}
}

func (cli *Client) UploadByFilename(fileName string) (string, error) {
	fileInfo, err := newFileInfo(fileName, nil, "")
	defer fileInfo.Close()
	if err != nil {
		return "", err
	}

	storageInfo, err := cli.queryStorageInfoWithTracker(TRACKER_PROTO_CMD_SERVICE_QUERY_STORE_WITHOUT_GROUP_ONE, "", "")
	if err != nil {
		return "", err
	}

	task := &storageUploadTask{}
	//req
	task.fileInfo = fileInfo
	task.storagePathIndex = storageInfo.storagePathIndex

	if err := cli.doStorage(task, storageInfo); err != nil {
		return "", err
	}
	return task.fileID, nil
}

func (cli *Client) UploadByBuffer(buffer []byte, fileExtName string) (string, error) {
	fileInfo, err := newFileInfo("", buffer, fileExtName)
	defer fileInfo.Close()
	if err != nil {
		return "", err
	}
	storageInfo, err := cli.queryStorageInfoWithTracker(TRACKER_PROTO_CMD_SERVICE_QUERY_STORE_WITHOUT_GROUP_ONE, "", "")
	if err != nil {
		return "", err
	}

	task := &storageUploadTask{}
	//req
	task.fileInfo = fileInfo
	task.storagePathIndex = storageInfo.storagePathIndex

	if err := cli.doStorage(task, storageInfo); err != nil {
		return "", err
	}
	return task.fileID, nil
}

func (cli *Client) DownloadToFile(fileID string, localFilename string, offset int64, downloadBytes int64) error {
	groupName, remoteFilename, err := splitFileId(fileID)
	if err != nil {
		return err
	}
	storageInfo, err := cli.queryStorageInfoWithTracker(TRACKER_PROTO_CMD_SERVICE_QUERY_FETCH_ONE, groupName, remoteFilename)
	if err != nil {
		return err
	}

	task := &storageDownloadTask{}
	// request
	task.groupName = groupName
	task.remoteFilename = remoteFilename
	task.offset = offset
	task.downloadBytes = downloadBytes

	// response
	task.localFilename = localFilename

	return cli.doStorage(task, storageInfo)
}

//deprecated
func (cli *Client) DownloadToBuffer(fileID string, offset int64, downloadBytes int64) ([]byte, error) {
	groupName, remoteFilename, err := splitFileId(fileID)
	if err != nil {
		return nil, err
	}
	storageInfo, err := cli.queryStorageInfoWithTracker(TRACKER_PROTO_CMD_SERVICE_QUERY_FETCH_ONE, groupName, remoteFilename)
	if err != nil {
		return nil, err
	}

	task := &storageDownloadTask{}
	// request
	task.groupName = groupName
	task.remoteFilename = remoteFilename
	task.offset = offset
	task.downloadBytes = downloadBytes

	// response
	if err := cli.doStorage(task, storageInfo); err != nil {
		return nil, err
	}
	return task.buffer, nil
}

func (cli *Client) DownloadToAllocatedBuffer(fileID string, buffer []byte, offset int64, downloadBytes int64) error {
	groupName, remoteFilename, err := splitFileId(fileID)
	if err != nil {
		return err
	}
	storageInfo, err := cli.queryStorageInfoWithTracker(TRACKER_PROTO_CMD_SERVICE_QUERY_FETCH_ONE, groupName, remoteFilename)
	if err != nil {
		return err
	}

	task := &storageDownloadTask{}
	//req
	task.groupName = groupName
	task.remoteFilename = remoteFilename
	task.offset = offset
	task.downloadBytes = downloadBytes
	task.buffer = buffer //allocate buffer by user

	//res
	if err := cli.doStorage(task, storageInfo); err != nil {
		return err
	}
	return nil
}

func (cli *Client) DeleteFile(fileID string) error {
	groupName, remoteFilename, err := splitFileId(fileID)
	if err != nil {
		return err
	}
	storageInfo, err := cli.queryStorageInfoWithTracker(TRACKER_PROTO_CMD_SERVICE_QUERY_FETCH_ONE, groupName, remoteFilename)
	if err != nil {
		return err
	}

	task := &storageDeleteTask{}
	//req
	task.groupName = groupName
	task.remoteFilename = remoteFilename

	return cli.doStorage(task, storageInfo)
}

func (cli *Client) doTracker(task task) error {
	trackerConn, err := cli.getTrackerConn()
	if err != nil {
		return err
	}
	defer trackerConn.Close()

	if err := task.SendReq(trackerConn); err != nil {
		return err
	}
	if err := task.RecvRes(trackerConn); err != nil {
		return err
	}

	return nil
}

func (cli *Client) doStorage(task task, storageInfo *storageInfo) error {
	storageConn, err := cli.getStorageConn(storageInfo)
	if err != nil {
		return err
	}
	defer storageConn.Close()

	if err := task.SendReq(storageConn); err != nil {
		return err
	}
	if err := task.RecvRes(storageConn); err != nil {
		return err
	}

	return nil
}

func (cli *Client) queryStorageInfoWithTracker(cmd int8, groupName string, remoteFilename string) (*storageInfo, error) {
	task := &trackerTask{}
	task.cmd = cmd
	task.groupName = groupName
	task.remoteFilename = remoteFilename

	if err := cli.doTracker(task); err != nil {
		return nil, err
	}
	return &storageInfo{
		addr:             fmt.Sprintf("%s:%d", task.ipAddr, task.port),
		storagePathIndex: task.storePathIndex,
	}, nil
}

func (cli *Client) getTrackerConn() (net.Conn, error) {
	var trackerConn net.Conn
	var err error
	var getOne bool
	for _, trackerPool := range cli.trackerPool {
		trackerConn, err = trackerPool.get()
		if err == nil {
			getOne = true
			break
		}
	}
	if getOne {
		return trackerConn, nil
	}
	if err == nil {
		return nil, fmt.Errorf("no connPool can be use")
	}
	return nil, err
}

func (cli *Client) getStorageConn(storageInfo *storageInfo) (net.Conn, error) {
	cli.storagePoolLock.Lock()
	storagePool, ok := cli.storagePool[storageInfo.addr]
	if ok {
		cli.storagePoolLock.Unlock()
		return storagePool.get()
	}
	storagePool, err := newConnPool(storageInfo.addr, cli.config.MaxConn)
	if err != nil {
		cli.storagePoolLock.Unlock()
		return nil, err
	}
	cli.storagePool[storageInfo.addr] = storagePool
	cli.storagePoolLock.Unlock()
	return storagePool.get()
}

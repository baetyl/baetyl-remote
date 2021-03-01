package main

import (
	"compress/flate"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/baetyl/baetyl-go/log"
	"github.com/baetyl/baetyl-go/utils"
	"github.com/docker/distribution/uuid"
	"github.com/mholt/archiver"
	"github.com/panjf2000/ants"
)

type arch interface {
	// Archive archivies an archive file on disk
	Archive(source []string, destination string) error
}

// nonArch origin file
type nonArch struct{}

// Archive no action
func (a *nonArch) Archive(source []string, destination string) error {
	return nil
}

// Task StorageClient
type Task struct {
	msg *EventMessage
	cb  func(msg *EventMessage, err error)
}

// FileStats upload stats
type FileStats struct {
	success uint64
	fail    uint64
	limit   uint64
	deleted uint64
}

// StorageClient StorageClient
type StorageClient struct {
	cfg   ClientInfo
	sh    IObjectStorage
	stats Stats
	pwd   string
	fs    *FileStats
	arch  arch
	log   *log.Logger
	tomb  utils.Tomb
	pool  *ants.PoolWithFunc
	lock  sync.RWMutex
}

// NewStorageClient creates a new newStorageClient
func NewStorageClient(cfg ClientInfo) (*StorageClient, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	sh, err := NewObjectStorageHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client (%s): %s", cfg.Name, err.Error())
	}
	b := &StorageClient{
		cfg:  cfg,
		sh:   sh,
		pwd:  pwd,
		arch: &nonArch{},
		fs:   &FileStats{},
		log:  log.With(log.Any("remote-object", "storage")),
	}
	return b, nil
}

// CallAsync submit task
func (cli *StorageClient) CallAsync(msg *EventMessage, cb func(msg *EventMessage, err error)) error {
	if !cli.tomb.Alive() {
		return fmt.Errorf("client (%s) closed", cli.cfg.Name)
	}
	return cli.invoke(msg, cb)
}

func (cli *StorageClient) invoke(msg *EventMessage, cb func(msg *EventMessage, err error)) error {
	if cli.pool.Running() == cli.cfg.Pool.Worker {
		cb(msg, fmt.Errorf("failed to submit task: no worker can be used"))
		return nil
	}
	task := &Task{
		msg: msg,
		cb:  cb,
	}
	if err := cli.pool.Invoke(task); err != nil {
		cb(msg, fmt.Errorf("failed to invoke pool task: %s", err.Error()))
		return nil
	}
	return nil
}

func (cli *StorageClient) call(task interface{}) {
	t, ok := task.(*Task)
	if !ok {
		return
	}
	var err error
	switch t.msg.Event.Type {
	case Upload:
		uploadEvent := t.msg.Event.Content.(*UploadEvent)
		err = cli.handleUploadEvent(uploadEvent)
	default:
		err = fmt.Errorf("EventMessage type unexpected")
	}
	if err != nil {
		cli.log.Error("failed to fetch message")
	}
	if t.cb != nil {
		t.cb(t.msg, err)
	}
}

// upload upload object to service(BOS, CEPH or AWS S3)
func (cli *StorageClient) upload(f, remotePath string, meta map[string]string) error {
	fsize, md5 := cli.fileSizeMd5(f)
	saved := cli.checkFile(remotePath, md5)
	if saved {
		return nil
	}
	if cli.cfg.Limit.Enable {
		month := time.Unix(0, time.Now().UnixNano()).Format("2006-01")
		err := cli.checkData(fsize, month)
		if err != nil {
			cli.log.Error("failed to pass data check", log.Error(err))
			atomic.AddUint64(&cli.fs.limit, 1)
			return nil
		}
		err = cli.putObjectWithStats(cli.cfg.Bucket, remotePath, f, meta)
		if err != nil {
			return err
		}
		return cli.increaseData(fsize, month)
	}
	err := cli.putObjectWithStats(cli.cfg.Bucket, remotePath, f, meta)
	if err != nil {
		return err
	}
	return nil
}

func (cli *StorageClient) putObjectWithStats(bucket, remotePath, f string, meta map[string]string) error {
	err := cli.sh.PutObjectFromFile(bucket, remotePath, f, meta)
	if err != nil {
		cli.log.Error("failed to upload file", log.Error(err))
		atomic.AddUint64(&cli.fs.fail, 1)
		return err
	}
	atomic.AddUint64(&cli.fs.success, 1)
	cli.log.Info("upload file success")
	return nil
}

func (cli *StorageClient) handleUploadEvent(e *UploadEvent) error {
	if strings.Contains(e.LocalPath, "..") {
		cli.log.Error("failed to pass LocalPath check: the local path can't contains ..")
		return nil
	}
	var t string
	p, err := filepath.EvalSymlinks(path.Join(cli.pwd, e.LocalPath))
	if err != nil {
		cli.log.Error("failed get real path", log.Error(err))
		atomic.AddUint64(&cli.fs.deleted, 1)
		return nil
	}
	if ok := utils.FileExists(p); ok {
		if e.Zip {
			t = path.Join(cli.cfg.TempPath, uuid.Generate().String())
			cli.arch = &archiver.Zip{
				CompressionLevel:     flate.DefaultCompression,
				MkdirAll:             true,
				SelectiveCompression: true,
				OverwriteExisting:    true,
			}
		} else {
			t = p
		}
	} else if ok = utils.DirExists(p); ok {
		t = path.Join(cli.cfg.TempPath, uuid.Generate().String())
		if e.Zip {
			cli.arch = &archiver.Zip{
				CompressionLevel:     flate.DefaultCompression,
				MkdirAll:             true,
				SelectiveCompression: true,
				OverwriteExisting:    true,
			}
		} else {
			cli.arch = &archiver.Tar{
				MkdirAll:          true,
				OverwriteExisting: true,
			}
		}
	} else {
		atomic.AddUint64(&cli.fs.deleted, 1)
		return fmt.Errorf("failed to find path: %s", p)
	}
	err = cli.arch.Archive([]string{p}, t)
	if t != p {
		defer os.RemoveAll(t)
	}
	if err != nil {
		return fmt.Errorf("failed to zip/tar dir: %s", err.Error())
	}
	return cli.upload(t, e.RemotePath, e.Meta)
}

func (cli *StorageClient) checkFile(remotePath, md5 string) bool {
	return cli.sh.FileExists(cli.cfg.Bucket, remotePath, md5)
}

func (cli *StorageClient) fileSizeMd5(f string) (int64, string) {
	fi, err := os.Stat(f)
	if err != nil {
		cli.log.Error("failed to get file info", log.Error(err))
		return 0, ""
	}
	fsize := fi.Size()
	md5, err := utils.CalculateFileMD5(f)
	if err != nil {
		cli.log.Error("failed to calculate file MD5", log.Error(err))
		return fsize, ""
	}
	return fsize, md5
}

func (cli *StorageClient) checkData(fsize int64, month string) error {
	if cli.cfg.Limit.Data <= 0 {
		return fmt.Errorf("limit data should be greater than 0(Byte)")
	}
	cli.lock.RLock()
	defer cli.lock.RUnlock()
	if _, ok := cli.stats.Months[month]; ok {
		new := cli.stats.Months[month].Bytes + fsize
		if new > int64(cli.cfg.Limit.Data) {
			return fmt.Errorf("exceeds max upload data size of this month，stop to upload and will reset next month")
		}
	}
	return nil
}

func (cli *StorageClient) increaseData(fsize int64, month string) error {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if _, ok := cli.stats.Months[month]; !ok {
		cli.stats.Months[month] = &Item{}
	}
	cli.stats.Total.Bytes = cli.stats.Total.Bytes + fsize
	cli.stats.Total.Count++
	cli.stats.Months[month].Bytes = cli.stats.Months[month].Bytes + fsize
	cli.stats.Months[month].Count++
	return DumpYAML(cli.cfg.Limit.Path, &cli.stats)
}

// Start start all worker
func (cli *StorageClient) Start() error {
	err := os.MkdirAll(cli.cfg.TempPath, 0755)
	if err != nil {
		cli.log.Error("failed to make directory", log.Error(err))
		return err
	}
	if ok := utils.FileExists(cli.cfg.Limit.Path); !ok {
		basepath := path.Dir(cli.cfg.Limit.Path)
		err = os.MkdirAll(basepath, 0755)
		if err != nil {
			cli.log.Error("failed to make directory", log.Error(err))
			return err
		}
		f, err := os.Create(cli.cfg.Limit.Path)
		defer f.Close()
		if err != nil {
			cli.log.Error("failed to make file", log.Error(err))
			return err
		}
	}
	utils.LoadYAML(cli.cfg.Limit.Path, &cli.stats)
	p, err := ants.NewPoolWithFunc(cli.cfg.Pool.Worker, cli.call, ants.WithExpiryDuration(cli.cfg.Pool.Idletime))
	if err != nil {
		cli.log.Error("failed to create a pool", log.Error(err))
		return err
	}
	cli.pool = p
	cli.log.Debug("storage client start")
	return nil
}

// Close close client and all worker
func (cli *StorageClient) Close() error {
	cli.pool.Release()
	cli.tomb.Kill(nil)
	cli.log.Debug("storage client closed")
	return cli.tomb.Wait()
}

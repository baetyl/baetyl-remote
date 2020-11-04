package main

import (
	"compress/flate"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/baetyl/baetyl-go/v2/errors"
	"github.com/baetyl/baetyl-go/v2/log"
	"github.com/baetyl/baetyl-go/v2/utils"
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

// Client ObjectClient
type Client struct {
	cfg     ClientInfo
	handler StorageHandler
	stats   Stats
	pwd     string
	fs      *FileStats
	arch    arch
	log     *log.Logger
	pool    *ants.PoolWithFunc
	lock    sync.Mutex
	tomb    utils.Tomb
}

// NewClient creates a new object client
func NewClient(cfg ClientInfo) (*Client, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Trace(err)
	}
	handler, err := NewObjectStorageHandler(cfg)
	if err != nil {
		return nil, errors.Trace(err)
	}
	cli := &Client{
		cfg:     cfg,
		pwd:     pwd,
		handler: handler,
		arch:    &nonArch{},
		fs:      &FileStats{},
		log:     log.With(log.Any("client", cfg.Name)),
	}

	err = os.MkdirAll(cfg.TempPath, 0755)
	if err != nil {
		return nil, errors.Errorf("failed to make dir (%s): %s", cli.cfg.TempPath, err)
	}
	if ok := utils.FileExists(cli.cfg.Limit.Path); !ok {
		basepath := path.Dir(cli.cfg.Limit.Path)
		err = os.MkdirAll(basepath, 0755)
		if err != nil {
			return nil, errors.Errorf("failed to make dir (%s): %s", basepath, err)
		}
		f, err := os.Create(cli.cfg.Limit.Path)
		if err != nil {
			return nil, errors.Errorf("failed to make file (%s): %s", cli.cfg.Limit.Path, err.Error())
		}
		defer f.Close()
	}
	err = utils.LoadYAML(cli.cfg.Limit.Path, &cli.stats)
	if err != nil {
		return nil, errors.Trace(err)
	}
	p, err := ants.NewPoolWithFunc(cli.cfg.Pool.Worker, cli.call, ants.WithExpiryDuration(cli.cfg.Pool.Idletime))
	if err != nil {
		return nil, errors.Errorf("failed to create a pool: %s", err.Error())
	}
	cli.pool = p

	cli.tomb.Go(cli.recording)
	cli.log.Debug("client starts")
	return cli, nil
}

// CallAsync submit task
func (cli *Client) CallAsync(msg *EventMessage, cb func(msg *EventMessage, err error)) error {
	if cli.pool.Running() == cli.cfg.Pool.Worker {
		cb(msg, errors.New("failed to submit task: no worker can be used"))
		return nil
	}
	task := &Task{
		msg: msg,
		cb:  cb,
	}
	if err := cli.pool.Invoke(task); err != nil {
		cb(msg, errors.Errorf("failed to invoke pool task: %s", err.Error()))
		return nil
	}
	return nil
}

func (cli *Client) call(task interface{}) {
	t, ok := task.(*Task)
	if !ok {
		log.Error(errors.New("failed to convert interface{} to *Task"))
		return
	}
	var err error
	switch t.msg.Event.Type {
	case Upload:
		uploadEvent, ok := t.msg.Event.Content.(*UploadEvent)
		if !ok {
			log.Error(errors.New("failed to convert interface{} to *UploadEvent"))
			return
		}
		err = cli.handleUploadEvent(uploadEvent)
	default:
		err = errors.New("EventMessage type unexpected")
	}
	if err != nil {
		cli.log.Error("error occurred in Client.call", log.Error(err))
	}
	if t.cb != nil {
		t.cb(t.msg, err)
	}
}

// upload upload object to service(BOS, CEPH or AWS S3)
func (cli *Client) upload(f, remotePath string, meta map[string]string) error {
	fsize, md5 := cli.fileSizeMd5(f)
	saved, err := cli.checkFile(remotePath, md5)
	if err != nil {
		return errors.Trace(err)
	}
	if saved {
		return nil
	}
	if cli.cfg.Limit.Enable {
		month := time.Unix(0, time.Now().UnixNano()).Format("2006-01")
		err := cli.checkData(fsize, month)
		if err != nil {
			atomic.AddUint64(&cli.fs.limit, 1)
			return errors.Errorf("failed to pass data check: %s", err.Error())
		}
		err = cli.putObjectWithStats(cli.cfg.Bucket, remotePath, f, meta)
		if err != nil {
			return err
		}
		return cli.increaseData(fsize, month)
	}
	err = cli.putObjectWithStats(cli.cfg.Bucket, remotePath, f, meta)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (cli *Client) putObjectWithStats(bucket, remotePath, f string, meta map[string]string) error {
	err := cli.handler.PutObjectFromFile(bucket, remotePath, f, meta)
	if err != nil {
		cli.log.Error("failed to put object from file", log.Any("localFile", f), log.Any("remotePath", remotePath), log.Any("bucket", bucket), log.Error(err))
		atomic.AddUint64(&cli.fs.fail, 1)
		return errors.Trace(err)
	}
	cli.log.Info("put object from file successfully", log.Any("localFile", f), log.Any("remotePath", remotePath), log.Any("bucket", bucket))
	atomic.AddUint64(&cli.fs.success, 1)
	return nil
}

func (cli *Client) handleUploadEvent(e *UploadEvent) error {
	if strings.Contains(e.LocalPath, "..") {
		return errors.Errorf("failed to pass LocalPath (%s) check: the local path can't contains ..", e.LocalPath)
	}
	var t string
	p, err := filepath.EvalSymlinks(path.Join(cli.pwd, e.LocalPath))
	if err != nil {
		atomic.AddUint64(&cli.fs.deleted, 1)
		return errors.Errorf("failed get real dir path: %s", err.Error())
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
		return errors.Errorf("failed to find path: %s", p)
	}
	err = cli.arch.Archive([]string{p}, t)
	if t != p {
		defer os.RemoveAll(t)
	}
	if err != nil {
		return errors.Errorf("failed to zip/tar dir: %s", err.Error())
	}
	return cli.upload(t, e.RemotePath, e.Meta)
}

func (cli *Client) checkFile(remotePath, md5 string) (bool, error) {
	res, err := cli.handler.FileExists(cli.cfg.Bucket, remotePath, md5)
	if err != nil {
		return false, errors.Trace(err)
	}
	return res, nil
}

func (cli *Client) fileSizeMd5(f string) (int64, string) {
	fi, err := os.Stat(f)
	if err != nil {
		cli.log.Error("failed to get file info: %s", log.Error(err))
		return 0, ""
	}
	fsize := fi.Size()

	md5, err := utils.CalculateFileMD5(f)
	if err != nil {
		cli.log.Error("failed to calculate file MD5", log.Any("filename", f), log.Error(err))
		return fsize, ""
	}
	return fsize, md5
}

func (cli *Client) checkData(fsize int64, month string) error {
	if cli.cfg.Limit.Data <= 0 {
		return errors.New("limit data should be greater than 0(Byte)")
	}
	cli.lock.Lock()
	defer cli.lock.Unlock()
	if _, ok := cli.stats.Months[month]; ok {
		newSize := cli.stats.Months[month].Bytes + fsize
		if newSize > cli.cfg.Limit.Data {
			return errors.New("exceeds max upload data size of this month，stop to upload and will reset next month")
		}
	}
	return nil
}

func (cli *Client) increaseData(fsize int64, month string) error {
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

func (cli *Client) recording() error {
	defer cli.log.Debug("client recording task stopped")
	t := time.NewTicker(cli.cfg.Record.Interval)
	defer t.Stop()
	for {
		select {
		case <-cli.tomb.Dying():
			return nil
		case <-t.C:
			cli.log.Info("client stats data",
				log.Any("success", cli.fs.success),
				log.Any("fail", cli.fs.fail),
				log.Any("limit", cli.fs.limit),
				log.Any("deleted", cli.fs.deleted))
		}
	}
}

// Close close client and all worker
func (cli *Client) Close() error {
	cli.log.Info("client starts to close")
	defer cli.log.Info("client closed")

	cli.pool.Release()
	cli.tomb.Kill(nil)
	return cli.tomb.Wait()
}

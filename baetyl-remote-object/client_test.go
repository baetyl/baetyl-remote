package main

//
//import (
//	"io/ioutil"
//	"os"
//	"path"
//	"testing"
//	"time"
//
//	"github.com/baetyl/baetyl/protocol/mqtt"
//	"github.com/stretchr/testify/assert"
//)
//
//func newStorageClient(r report) (*StorageClient, error) {
//	cfg.Kind = Kind("S3")
//	cfg.Region = "us-east-1"
//	storageClient, err := NewStorageClient(*cfg, r)
//	return storageClient, err
//}
//
//func generateTempPath(prefix string) (string, error) {
//	dir, err := ioutil.TempDir("", prefix)
//	if err != nil {
//		return "", err
//	}
//	tmpfile, err := ioutil.TempFile(dir, prefix)
//	if err != nil {
//		return "", err
//	}
//	fpath := tmpfile.Name() + ".yml"
//	return fpath, nil
//}
//
//func TestNewStorageClient(t *testing.T) {
//	// round 1: report is not nil
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//	assert.Equal(t, Kind("S3"), storageClient.cfg.Kind)
//	assert.Equal(t, "test", storageClient.cfg.Name)
//
//	// round 2: report is nil
//	storageClient, err = newStorageClient(nil)
//	assert.Nil(t, err)
//	assert.Equal(t, Kind("S3"), storageClient.cfg.Kind)
//	assert.Equal(t, "test", storageClient.cfg.Name)
//}
//
//// CallAsync and invoke
//func TestCallAsync(t *testing.T) {
//	// create storage client
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//
//	// start storage client
//	tempPath, err := generateTempPath("example")
//	defer os.RemoveAll(path.Dir(tempPath))
//	assert.Nil(t, err)
//	storageClient.cfg.TempPath = tempPath
//	statsPath, err := generateTempPath("test")
//	defer os.RemoveAll(path.Dir(statsPath))
//	assert.Nil(t, err)
//	storageClient.cfg.Limit.Path = statsPath
//	storageClient.cfg.Pool.Worker = 10
//	storageClient.cfg.Pool.Idletime = time.Duration(30000000000)
//	err = storageClient.Start()
//	assert.Nil(t, err)
//	defer storageClient.Close()
//
//	msg := &EventMessage{
//		ID:    1,
//		QOS:   uint32(1),
//		Topic: "t",
//		Event: &Event{
//			Time:    time.Now(),
//			Type:    EventType("UPLOAD"),
//			Content: nil,
//		},
//	}
//	hub := new(mqtt.ClientInfo)
//	ruler := NewRuler(*ru, *hub, Client(storageClient))
//	err = storageClient.CallAsync(msg, ruler.callback)
//	assert.Nil(t, err)
//}
//
//func TestCall(t *testing.T) {
//	// create storage client
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//	// create ruler
//	hub := new(mqtt.ClientInfo)
//	ruler := NewRuler(*ru, *hub, Client(storageClient))
//	task1 := &Task{
//		msg: &EventMessage{
//			ID:    1,
//			QOS:   uint32(1),
//			Topic: "t",
//			Event: &Event{
//				Time:    time.Now(),
//				Type:    EventType("TEST"), // unsupported event type
//				Content: nil,
//			},
//		},
//		cb: ruler.callback,
//	}
//	// start call
//	storageClient.call(task1)
//
//	// unsupported struct when convert to Task struct
//	task2 := map[string]string{}
//	storageClient.call(task2)
//}
//
//func TestUpload(t *testing.T) {
//	// create storage client
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//
//	// round 1: local file is not exist
//	err = storageClient.upload("var/test/file", "", map[string]string{})
//	assert.NotNil(t, err)
//	assert.Equal(t, "open var/test/file: no such file or directory", err.Error())
//
//	// round 2: file exists without limit data
//	storageClient.cfg.Bucket = "Bucket"
//	storageClient.cfg.MultiPart.PartSize = 1048576000
//	storageClient.cfg.MultiPart.Concurrency = 10
//	err = storageClient.upload("./example/test/baetyl/service.yml", "var/file/service.yml", map[string]string{})
//	assert.NotNil(t, err)
//	assert.Equal(t, "EmptyStaticCreds: static credentials are empty", err.Error()) // without AccessKey and SecretKey
//
//	// round 3: file exists with limit data
//	storageClient.cfg.Limit.Enable = true
//	storageClient.cfg.Limit.Data = 1073741824
//	storageClient.cfg.Limit.Path = "var/lib/baetyl/data/stats.yml"
//	storageClient.stats.Months = map[string]*Item{
//		"2019-09": &Item{
//			Bytes: 21234345,
//			Count: 20,
//		}}
//	err = storageClient.upload("./example/test/baetyl/service.yml", "var/file/service.yml", map[string]string{})
//	assert.NotNil(t, err)
//	assert.Equal(t, "EmptyStaticCreds: static credentials are empty", err.Error()) // without AccessKey and SecretKey
//}
//
//func TestHandleUploadEvent(t *testing.T) {
//	// create storage client
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//
//	// contains '..' of local path
//	e := &UploadEvent{
//		RemotePath: "var/file/service.yml",
//		LocalPath:  "../example/test/baetyl/service.yml",
//		Zip:        false,
//		Meta:       make(map[string]string),
//	}
//	err = storageClient.handleUploadEvent(e)
//	assert.Nil(t, err)
//
//	// wrong path
//	e.LocalPath = "./test/baetyl/service.yml"
//	err = storageClient.handleUploadEvent(e)
//	assert.Nil(t, err)
//
//	// real path
//	e.LocalPath = "./example/test/baetyl/service.yml"
//	storageClient.cfg.Bucket = "Bucket"
//	storageClient.cfg.MultiPart.PartSize = 1048576000
//	storageClient.cfg.MultiPart.Concurrency = 10
//	err = storageClient.handleUploadEvent(e)
//	assert.NotNil(t, err)
//	assert.Equal(t, "EmptyStaticCreds: static credentials are empty", err.Error()) // without AccessKey and SecretKey
//
//	// zip is true, upload file
//	e.Zip = true
//	e.RemotePath = "var/file/test.zip"
//	err = storageClient.handleUploadEvent(e)
//	assert.NotNil(t, err)
//	assert.Equal(t, "failed to zip/tar dir: checking extension: filename must have a .zip extension", err.Error())
//
//	// zip is true, upload directory
//	e.LocalPath = "./example"
//	err = storageClient.handleUploadEvent(e)
//	assert.NotNil(t, err)
//	assert.Equal(t, "failed to zip/tar dir: checking extension: filename must have a .zip extension", err.Error())
//
//	// zip is false, tar directory and upload
//	e.Zip = false
//	e.RemotePath = "var/file/test.tar"
//	err = storageClient.handleUploadEvent(e)
//	assert.NotNil(t, err)
//	assert.Equal(t, "failed to zip/tar dir: checking extension: filename must have a .tar extension", err.Error())
//}
//
//func TestCheckFile(t *testing.T) {
//	remotePath := "var/file/service.yml"
//	md5 := "4a0fb0ea68b05a84234e420d1f8cb32b"
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//	rlt := storageClient.checkFile(remotePath, md5)
//	assert.Equal(t, false, rlt)
//}
//
//func TestCheckData(t *testing.T) {
//	// create storage client
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//
//	// round 1: limit data less than 0(Byte)
//	storageClient.cfg.Limit.Enable = true
//	storageClient.cfg.Limit.Data = -1
//	storageClient.cfg.Limit.Path = "var/lib/baetyl/data/stats.yml"
//	err = storageClient.checkData(200, "2019-09")
//	assert.NotNil(t, err)
//	assert.Equal(t, "limit data should be greater than 0(Byte)", err.Error())
//
//	// round 2: limit data greater than 0(Byte), will exceed
//	storageClient.cfg.Limit.Data = 1073741824
//	storageClient.stats.Months = map[string]*Item{
//		"2019-09": &Item{
//			Bytes: 221212425,
//			Count: 20,
//		}}
//	err = storageClient.checkData(2000000000, "2019-09")
//	assert.NotNil(t, err)
//	assert.Equal(t, "exceeds max upload data size of this month，stop to upload and will reset next month", err.Error())
//
//	// round 3: limit data greater than 0(Byte), will not exceed
//	storageClient.cfg.Limit.Data = 107374182400
//	err = storageClient.checkData(2000000000, "2019-09")
//	assert.Nil(t, err)
//}
//
//func TestStartAndClose(t *testing.T) {
//	// create storage client
//	storageClient, err := newStorageClient(r)
//	assert.Nil(t, err)
//
//	// file is not exist
//	storageClient.cfg.TempPath = "var/file/test"
//	defer os.RemoveAll("var/")
//	err = storageClient.Start()
//	assert.NotNil(t, err)
//	assert.Equal(t, "open : no such file or directory", err.Error())
//
//	// cannot make directory of temppath
//	storageClient.cfg.TempPath = "/usr/data.yml"
//	err = storageClient.Start()
//	assert.NotNil(t, err)
//	assert.Equal(t, "mkdir /usr/data.yml: operation not permitted", err.Error())
//
//	// cannot make directory of limit path
//	dir, err := ioutil.TempDir("", "example")
//	assert.Nil(t, err)
//	defer os.RemoveAll(dir)
//	storageClient.cfg.TempPath = dir
//	storageClient.cfg.Limit.Path = "/var/file/service.yml"
//	err = storageClient.Start()
//	assert.NotNil(t, err)
//	assert.Equal(t, "mkdir /var/file: permission denied", err.Error())
//
//	// invalid size for pool
//	tmpfile, err := ioutil.TempFile(dir, "test")
//	storageClient.cfg.Limit.Path = tmpfile.Name() + ".yml"
//	err = storageClient.Start()
//	assert.NotNil(t, err)
//	assert.Equal(t, "invalid size for pool", err.Error())
//
//	// storage client start successfully
//	storageClient.cfg.Pool.Worker = 10
//	storageClient.cfg.Pool.Idletime = time.Duration(30000000000)
//	err = storageClient.Start()
//	assert.Nil(t, err)
//
//	// storage client close
//	err = storageClient.Close()
//	assert.Nil(t, err)
//}

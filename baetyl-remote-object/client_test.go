package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newClient() (*Client, error) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"
	storageClient, err := NewClient(*cfg)
	return storageClient, err
}

func generateTempPath(prefix string) (string, error) {
	dir, err := ioutil.TempDir("", prefix)
	if err != nil {
		return "", err
	}
	tmpfile, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		return "", err
	}
	fpath := tmpfile.Name() + ".yml"
	return fpath, nil
}

func TestNewStorageClient(t *testing.T) {
	// round 1: report is not nil
	_, err := newClient()
	assert.Equal(t, err.Error(), "failed to make dir (): mkdir : no such file or directory")
}

// CallAsync and invoke
func TestCallAsync(t *testing.T) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"

	tempPath, err := generateTempPath("example")
	defer os.RemoveAll(path.Dir(tempPath))
	assert.Nil(t, err)

	cfg.TempPath = tempPath

	statsPath, err := generateTempPath("test")
	defer os.RemoveAll(path.Dir(statsPath))
	assert.Nil(t, err)

	cfg.Limit.Path = statsPath
	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)

	// create storage client
	storageClient, err := newClient()
	assert.Nil(t, err)
	defer storageClient.Close()

	msg := &EventMessage{
		ID:    1,
		QOS:   uint32(1),
		Topic: "t",
		Event: &Event{
			Time:    time.Now(),
			Type:    EventType("UPLOAD"),
			Content: nil,
		},
	}
	callback := func(msg *EventMessage, err error) {
		return
	}
	err = storageClient.CallAsync(msg, callback)
	assert.Nil(t, err)
}

func TestCall(t *testing.T) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"

	tempPath, err := generateTempPath("example")
	defer os.RemoveAll(path.Dir(tempPath))
	assert.Nil(t, err)

	cfg.TempPath = tempPath

	statsPath, err := generateTempPath("test")
	defer os.RemoveAll(path.Dir(statsPath))
	assert.Nil(t, err)

	cfg.Limit.Path = statsPath
	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)

	// create storage client
	storageClient, err := newClient()
	assert.Nil(t, err)
	defer storageClient.Close()

	callback := func(msg *EventMessage, err error) {
		return
	}

	task1 := &Task{
		msg: &EventMessage{
			ID:    1,
			QOS:   uint32(1),
			Topic: "t",
			Event: &Event{
				Time:    time.Now(),
				Type:    EventType("TEST"), // unsupported event type
				Content: nil,
			},
		},
		cb: callback,
	}
	// start call
	storageClient.call(task1)

	// unsupported struct when convert to Task struct
	task2 := map[string]string{}
	storageClient.call(task2)
}

func TestUpload(t *testing.T) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"
	cfg.Bucket = "default"

	tempPath, err := generateTempPath("example")
	defer os.RemoveAll(path.Dir(tempPath))
	assert.Nil(t, err)

	cfg.TempPath = tempPath

	statsPath, err := generateTempPath("test")
	defer os.RemoveAll(path.Dir(statsPath))
	assert.Nil(t, err)

	cfg.Limit.Path = statsPath
	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)

	// create storage client
	storageClient, err := newClient()
	assert.Nil(t, err)
	defer storageClient.Close()

	// round 1: local file is not exist
	err = storageClient.upload("var/test/file", "default", map[string]string{})
	assert.NotNil(t, err)
	assert.Equal(t, "EmptyStaticCreds: static credentials are empty", err.Error())

	// round 2: file exists without limit data
	storageClient.cfg.Bucket = "Bucket"
	storageClient.cfg.MultiPart.PartSize = 1048576000
	storageClient.cfg.MultiPart.Concurrency = 10
	err = storageClient.upload("./example/etc/baetyl/service-bos.yml", "var/file/service.yml", map[string]string{})
	assert.NotNil(t, err)
	assert.Equal(t, "EmptyStaticCreds: static credentials are empty", err.Error()) // without AccessKey and SecretKey

	// round 3: file exists with limit data
	storageClient.cfg.Limit.Enable = true
	storageClient.cfg.Limit.Data = 1073741824
	storageClient.cfg.Limit.Path = "var/lib/baetyl/data/stats.yml"
	storageClient.stats.Months = map[string]*Item{
		"2019-09": &Item{
			Bytes: 21234345,
			Count: 20,
		}}
	err = storageClient.upload("./example/test/baetyl/service.yml", "var/file/service.yml", map[string]string{})
	assert.NotNil(t, err)
	assert.Equal(t, "EmptyStaticCreds: static credentials are empty", err.Error()) // without AccessKey and SecretKey
}

func TestHandleUploadEvent(t *testing.T) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"

	tempPath, err := generateTempPath("example")
	defer os.RemoveAll(path.Dir(tempPath))
	assert.Nil(t, err)

	cfg.TempPath = tempPath

	statsPath, err := generateTempPath("test")
	defer os.RemoveAll(path.Dir(statsPath))
	assert.Nil(t, err)

	cfg.Limit.Path = statsPath
	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)

	// create storage client
	storageClient, err := newClient()
	assert.Nil(t, err)
	defer storageClient.Close()

	// contains '..' of local path
	e := &UploadEvent{
		RemotePath: "var/file/service.yml",
		LocalPath:  "./example/etc/baetyl/service-bos.yml",
		Zip:        false,
		Meta:       make(map[string]string),
	}

	// wrong path
	e.LocalPath = "../example/etc/baetyl/service-bos.yml"
	err = storageClient.handleUploadEvent(e)
	assert.Error(t, err)

	// real path
	e.LocalPath = "./example/etc/baetyl/service-bos.yml"
	storageClient.cfg.Bucket = "Bucket"
	storageClient.cfg.MultiPart.PartSize = 1048576000
	storageClient.cfg.MultiPart.Concurrency = 10
	err = storageClient.handleUploadEvent(e)
	assert.NotNil(t, err)
	assert.Equal(t, "EmptyStaticCreds: static credentials are empty", err.Error()) // without AccessKey and SecretKey

	// zip is true, upload file
	e.Zip = true
	e.RemotePath = "var/file/test.zip"
	err = storageClient.handleUploadEvent(e)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to zip/tar dir: checking extension: filename must have a .zip extension", err.Error())

	// zip is true, upload directory
	e.LocalPath = "./example"
	err = storageClient.handleUploadEvent(e)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to zip/tar dir: checking extension: filename must have a .zip extension", err.Error())

	// zip is false, tar directory and upload
	e.Zip = false
	e.RemotePath = "var/file/test.tar"
	err = storageClient.handleUploadEvent(e)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to zip/tar dir: checking extension: filename must have a .tar extension", err.Error())
}

func TestCheckFile(t *testing.T) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"
	cfg.Bucket = "default"

	tempPath, err := generateTempPath("example")
	defer os.RemoveAll(path.Dir(tempPath))
	assert.Nil(t, err)

	cfg.TempPath = tempPath

	statsPath, err := generateTempPath("test")
	defer os.RemoveAll(path.Dir(statsPath))
	assert.Nil(t, err)

	cfg.Limit.Path = statsPath
	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)

	// create storage client
	storageClient, err := newClient()
	assert.Nil(t, err)
	defer storageClient.Close()

	remotePath := "var/file/service.yml"
	md5 := "4a0fb0ea68b05a84234e420d1f8cb32b"

	rlt, err := storageClient.checkFile(remotePath, md5)
	assert.Equal(t, err.Error(), "EmptyStaticCreds: static credentials are empty")
	assert.Equal(t, false, rlt)
}

func TestCheckData(t *testing.T) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"

	tempPath, err := generateTempPath("example")
	defer os.RemoveAll(path.Dir(tempPath))
	assert.Nil(t, err)

	cfg.TempPath = tempPath

	statsPath, err := generateTempPath("test")
	defer os.RemoveAll(path.Dir(statsPath))
	assert.Nil(t, err)

	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)

	// round 1: limit data less than 0(Byte)
	cfg.Limit.Enable = true
	cfg.Limit.Data = -1
	cfg.Limit.Path = "var/lib/baetyl/data/stats.yml"

	// create storage client
	storageClient, err := newClient()
	assert.Nil(t, err)
	defer storageClient.Close()

	err = storageClient.checkData(200, "2019-09")
	assert.NotNil(t, err)
	assert.Equal(t, "limit data should be greater than 0(Byte)", err.Error())

	// round 2: limit data greater than 0(Byte), will exceed
	storageClient.cfg.Limit.Data = 1073741824
	storageClient.stats.Months = map[string]*Item{
		"2019-09": &Item{
			Bytes: 221212425,
			Count: 20,
		}}
	err = storageClient.checkData(2000000000, "2019-09")
	assert.NotNil(t, err)
	assert.Equal(t, "exceeds max upload data size of this month，stop to upload and will reset next month", err.Error())

	// round 3: limit data greater than 0(Byte), will not exceed
	storageClient.cfg.Limit.Data = 107374182400
	err = storageClient.checkData(2000000000, "2019-09")
	assert.Nil(t, err)
}

func TestStartAndClose(t *testing.T) {
	cfg.Kind = "S3"
	cfg.Region = "us-east-1"

	cfg.TempPath = "/usr/data.yml"
	defer os.RemoveAll("var/")

	statsPath, err := generateTempPath("test")
	defer os.RemoveAll(path.Dir(statsPath))
	assert.Nil(t, err)

	cfg.Limit.Path = statsPath
	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)

	// create storage client
	_, err = newClient()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to make dir (/usr/data.yml)")

	// cannot make directory of limit path
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)
	cfg.TempPath = dir
	cfg.Limit.Path = "/var/file/service.yml"
	// create storage client
	_, err = newClient()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to make dir (/var/file)")

	// invalid size for pool
	cfg.Pool.Worker = 0
	//cfg.Pool.Idletime = time.Duration(30000000000)
	tmpfile, err := ioutil.TempFile(dir, "test")
	cfg.Limit.Path = tmpfile.Name() + ".yml"
	// create storage client
	_, err = newClient()
	assert.NotNil(t, err)
	assert.Equal(t, "failed to create a pool: invalid size for pool", err.Error())

	// storage client start successfully
	cfg.Pool.Worker = 10
	cfg.Pool.Idletime = time.Duration(30000000000)
	// create storage client
	storageClient, err := newClient()
	assert.NoError(t, err)

	err = storageClient.Close()
	assert.NoError(t, err)
}

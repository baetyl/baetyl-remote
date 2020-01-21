package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/awstesting/mock"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/baetyl/baetyl-go/utils"
	"github.com/docker/distribution/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewBosHandler(t *testing.T) {
	// var bosHandler *BosHandler
	// round 1: create bos handler normally with none empty AccessKey and SecretKey
	cfg.Ak = uuid.Generate().String()
	cfg.Sk = uuid.Generate().String()
	_, err := NewBosHandler(*cfg)
	assert.Nil(t, err)

	// round 2: create bos handler normally with empty AccessKey and empty SecretKey
	cfg.Ak = ""
	cfg.Sk = ""
	_, err = NewBosHandler(*cfg)
	assert.Nil(t, err)

	// round 3: create bos handler failed with empty AccessKey and none empty SecretKey
	cfg.Ak = ""
	cfg.Sk = uuid.Generate().String()
	_, err = NewBosHandler(*cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to create bos client (test): accessKeyId should not be empty", err.Error())

	// round 4: create bos handler failed with empty SecretKey and none empty AccessKey
	cfg.Ak = uuid.Generate().String()
	cfg.Sk = ""
	_, err = NewBosHandler(*cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to create bos client (test): secretKey should not be empty", err.Error())
}

func TestPutObjectFromFile(t *testing.T) {
	cfg.Kind = Kind("S3")
	cfg.Region = "us-east-2"
	cfg.MultiPart.PartSize = 1048576000
	cfg.MultiPart.Concurrency = 10
	sc := mock.NewMockClient()
	up := s3manager.NewUploader(unit.Session)
	s3Handler := &S3Handler{
		s3Client: &s3.S3{sc},
		uploader: up,
		cfg:      *cfg,
	}
	err := s3Handler.PutObjectFromFile("Bucket", "Key", "./example/etc/baetyl/service-s3.yml", map[string]string{"name": "hahaha", "location": "Beijing"})
	assert.NotNil(t, err)
	assert.Equal(t, "RequestCanceled: request context canceled\ncaused by: context deadline exceeded", err.Error())
}

func TestFileExists(t *testing.T) {
	cfg.Kind = Kind("S3")
	sc := mock.NewMockClient()
	u := s3manager.NewUploader(unit.Session)
	s3Handler := &S3Handler{
		s3Client: &s3.S3{sc},
		uploader: u,
		cfg:      *cfg,
	}
	md5, err := utils.CalculateFileMD5("example/test/baetyl/service.yml")
	assert.Nil(t, err)
	rlt := s3Handler.FileExists("Bucket", "var/file/service.yml", md5)
	assert.Equal(t, false, rlt)
}

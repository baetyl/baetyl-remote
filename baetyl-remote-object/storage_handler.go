package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/baetyl/baetyl-go/v2/errors"
	"github.com/baetyl/baetyl-go/v2/log"
	"github.com/baidubce/bce-sdk-go/bce"
	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/baidubce/bce-sdk-go/services/bos/api"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// ObjectStorage interface
type StorageHandler interface {
	PutObjectFromFile(Bucket, remotePath, filename string, meta map[string]string) error
	FileExists(Bucket, remotePath, md5 string) (bool, error)
}

// NewObjectStorageHandler NewObjectStorageHandler
func NewObjectStorageHandler(cfg ClientInfo) (StorageHandler, error) {
	switch cfg.Kind {
	case Bos:
		return NewBosHandler(cfg)
	case S3:
		return NewS3Client(cfg)
	default:
		return nil, fmt.Errorf("kind type unexpected")
	}
}

// BosHandler BosHandler
type BosHandler struct {
	bos *bos.Client
	cfg ClientInfo
	log *log.Logger
}

// NewBosHandler creates a new newBosClient
func NewBosHandler(cfg ClientInfo) (StorageHandler, error) {
	cli, err := bos.NewClient(cfg.Ak, cfg.Sk, cfg.Endpoint)
	if err != nil {
		return nil, errors.Errorf("failed to create bos client (%s): %s", cfg.Name, err.Error())
	}
	cli.MultipartSize = cfg.MultiPart.PartSize
	cli.MaxParallel = (int64)(cfg.MultiPart.Concurrency)
	cli.Config.ConnectionTimeoutInMillis = (int)(cfg.Timeout / time.Millisecond)
	cli.Config.Retry = bce.NewBackOffRetryPolicy(cfg.Backoff.Max, (int64)(cfg.Backoff.Delay/time.Millisecond), (int64)(cfg.Backoff.Base/time.Millisecond))
	b := &BosHandler{
		bos: cli,
		cfg: cfg,
		log: log.With(log.Any("storage", "bos")),
	}
	return b, nil
}

// PutObjectFromFile upload file
func (cli *BosHandler) PutObjectFromFile(Bucket, remotePath, filename string, meta map[string]string) error {
	args := new(api.PutObjectArgs)
	args.UserMeta = meta
	_, err := cli.bos.PutObjectFromFile(Bucket, remotePath, filename, args)
	return errors.Trace(err)
}

// FileExists FileExists
func (cli *BosHandler) FileExists(Bucket, remotePath, md5 string) (bool, error) {
	res, err := cli.bos.GetObjectMeta(Bucket, remotePath)
	if err != nil {
		cli.log.Warn("failed to get object meta", log.Error(err))
		return false, nil
	}
	if res.ObjectMeta.ContentMD5 == md5 {
		return true, nil
	}
	return false, nil
}

// S3Handler S3Handler
type S3Handler struct {
	s3Client *s3.S3
	uploader *s3manager.Uploader
	cfg      ClientInfo
	log      *log.Logger
}

// NewS3Client creates a new NewS3Client
func NewS3Client(cfg ClientInfo) (StorageHandler, error) {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(cfg.Ak, cfg.Sk, ""),
		Endpoint:         aws.String(cfg.Endpoint),
		Region:           aws.String(cfg.Region),
		DisableSSL:       aws.Bool(!strings.HasPrefix(cfg.Endpoint, "https")),
		S3ForcePathStyle: aws.Bool(true),
	}
	sessionProvider, err := session.NewSession(s3Config)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &S3Handler{
		s3Client: s3.New(sessionProvider),
		cfg:      cfg,
		uploader: s3manager.NewUploader(sessionProvider),
		log:      log.With(log.Any("storage", "s3")),
	}, nil
}

// PutObjectFromFile upload file
func (cli *S3Handler) PutObjectFromFile(Bucket, remotePath, filename string, meta map[string]string) error {
	Metadata := make(map[string]*string)
	for k, v := range meta {
		Metadata[k] = &v
	}
	f, err := os.Open(filename)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()
	params := &s3manager.UploadInput{
		Bucket:   aws.String(Bucket),     // Required
		Key:      aws.String(remotePath), // Required
		Body:     f,
		Metadata: Metadata,
	}
	ctx, cancel := context.WithTimeout(context.Background(), cli.cfg.Timeout)
	defer cancel()
	_, err = cli.uploader.UploadWithContext(ctx, params, func(u *s3manager.Uploader) {
		u.PartSize = cli.cfg.MultiPart.PartSize
		u.LeavePartsOnError = true
		u.Concurrency = cli.cfg.MultiPart.Concurrency
	}) //并发数
	return errors.Trace(err)
}

// FileExists FileExists
func (cli *S3Handler) FileExists(Bucket, remotePath, md5 string) (bool, error) {
	cparams := &s3.HeadObjectInput{
		Bucket: aws.String(Bucket),
		Key:    aws.String(remotePath),
	}
	ho, err := cli.s3Client.HeadObject(cparams)
	if err != nil {
		return false, errors.Trace(err)
	}
	input, _ := hex.DecodeString(strings.Replace(*ho.ETag, "\"", "", -1))
	encodeString := base64.StdEncoding.EncodeToString(input)
	if encodeString == md5 {
		return true, nil
	}
	return false, nil
}

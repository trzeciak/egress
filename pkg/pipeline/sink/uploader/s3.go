package uploader

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/livekit/egress/pkg/types"
	"github.com/livekit/protocol/livekit"
)

type S3Uploader struct {
	awsConfig *aws.Config
	bucket    *string
	metadata  map[string]*string
	tagging   *string
}

func newS3Uploader(conf *livekit.S3Upload) Uploader {
	awsConfig := &aws.Config{
		MaxRetries:       aws.Int(maxRetries), // Switching to v2 of the aws Go SDK would allow to set a maxDelay as well.
		S3ForcePathStyle: aws.Bool(conf.ForcePathStyle),
	}
	if conf.AccessKey != "" && conf.Secret != "" {
		awsConfig.Credentials = credentials.NewStaticCredentials(conf.AccessKey, conf.Secret, "")
	}
	if conf.Endpoint != "" {
		awsConfig.Endpoint = aws.String(conf.Endpoint)
	}
	if conf.Region != "" {
		awsConfig.Region = aws.String(conf.Region)
	}

	u := &S3Uploader{
		awsConfig: awsConfig,
		bucket:    aws.String(conf.Bucket),
	}

	if len(conf.Metadata) > 0 {
		u.metadata = make(map[string]*string, len(conf.Metadata))
		for k, v := range conf.Metadata {
			v := v
			u.metadata[k] = &v
		}
	}

	if conf.Tagging != "" {
		u.tagging = aws.String(conf.Tagging)
	}

	return u
}

func (u *S3Uploader) Upload(localFilepath, storageFilepath string, outputType types.OutputType) (string, int64, error) {
	sess, err := session.NewSession(u.awsConfig)
	if err != nil {
		return "", 0, err
	}

	file, err := os.Open(localFilepath)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		_ = file.Close()
	}()

	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}

	_, err = s3manager.NewUploader(sess).Upload(&s3manager.UploadInput{
		Body:        file,
		Bucket:      u.bucket,
		ContentType: aws.String(string(outputType)),
		Key:         aws.String(storageFilepath),
		Metadata:    u.metadata,
		Tagging:     u.tagging,
	})
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", *u.bucket, storageFilepath), stat.Size(), nil
}
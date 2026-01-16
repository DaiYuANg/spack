package scanner

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/daiyuang/spack/internal/model"
)

type s3Backend struct {
	client *s3.Client
	bucket string
	prefix string
}

func NewS3Backend(client *s3.Client, bucket, prefix string) Backend {
	return &s3Backend{client: client, bucket: bucket, prefix: prefix}
}

func (b *s3Backend) Walk(walkFn func(obj *model.ObjectInfo) error) error {
	ctx := context.Background()
	paginator := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(b.prefix),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range output.Contents {
			key := *item.Key
			obj := &model.ObjectInfo{
				Key:   key,
				Size:  aws.ToInt64(item.Size),
				IsDir: false,
				Reader: func() (io.ReadCloser, error) {
					out, err := b.client.GetObject(ctx, &s3.GetObjectInput{
						Bucket: aws.String(b.bucket),
						Key:    aws.String(key),
					})
					if err != nil {
						return nil, err
					}
					return out.Body, nil
				},
				Metadata: map[string]string{
					"ETag":    aws.ToString(item.ETag),
					"LastMod": item.LastModified.String(),
				},
			}
			if err := walkFn(obj); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *s3Backend) Stat(key string) (*model.ObjectInfo, error) {
	// 可以用 HeadObject 获取元数据
	return nil, fmt.Errorf("not implemented")
}

package spaces

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Spaces struct {
	BucketName      string
	BaseURL         string
	InputDir        string
	OutputDir       string

	client 			*minio.Client
}

func New(endpoint, accessKeyID, secretAccessKey, bucketName, baseURL, inputDir, outputDir string) (*Spaces, error) {
	client, error := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true,
	})
	if error != nil {
		return nil, error
	}
	spaces := &Spaces{
		BucketName:      bucketName,
		BaseURL:         baseURL,
		InputDir:        inputDir,
		OutputDir:       outputDir,
		client:   		 client,
	}
	return spaces, nil
}

func (s *Spaces) BucketExists() (bool, error) {
	return s.client.BucketExists(context.Background(), s.BucketName)
}

func (s *Spaces) FGetObject(path string, destination string) error {
	return s.client.FGetObject(context.Background(), s.BucketName, path, destination, minio.GetObjectOptions{})
}

func (s *Spaces) GetObject(path string) (*minio.Object, error) {
	return s.client.GetObject(context.Background(), s.BucketName, path, minio.GetObjectOptions{})
}

func (s *Spaces) PutObject(path string, source string) (minio.UploadInfo, error) {
	return s.client.FPutObject(context.Background(), s.BucketName, path, source, minio.PutObjectOptions{})
}

func (s *Spaces) CheckObject(path string) (bool, error) {
	info, err := s.client.StatObject(context.Background(), s.BucketName, path, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, err
	}
	return info.Size > 0, nil
}


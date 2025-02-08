package tests

import (
	"aws-s3-actions/app"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockS3Client struct {
	mock.Mock
}

// GetObject implements app.s3Downloader.
func (m *MockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

// ListObjectVersions implements app.s3Downloader.
func (m *MockS3Client) ListObjectVersions(ctx context.Context, input *s3.ListObjectVersionsInput, opts ...func(*s3.Options)) (*s3.ListObjectVersionsOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.ListObjectVersionsOutput), args.Error(1)
}

// ListObjectsV2 implements app.s3Downloader.
func (m *MockS3Client) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, opts ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func TestDownloadObject_Success(t *testing.T) {
	mockS3 := new(MockS3Client)

	bucket := "test-bucket"
	prefix := "test-folder/test-file.txt"
	local := "/tmp"
	region := "us-east-1"
	fileContent := "conteúdo de teste"
	contentLength := int64(len(fileContent))

	r := io.NopCloser(bytes.NewReader([]byte(fileContent)))

	mockS3.On("GetObject", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{
		Body:          r,
		ContentLength: &contentLength,
	}, nil)

	configGlobal := app.ConfigGlobal{Region: region}
	err := app.DownloadObject(&configGlobal, mockS3, bucket, prefix, "", local, region)
	require.NoError(t, err)

	filePath := local + "/test-file.txt"
	defer os.Remove(filePath)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, fileContent, string(content))

	mockS3.AssertExpectations(t)
}

func TestDownloadObject_Failure(t *testing.T) {
	mockS3 := new(MockS3Client)
	bucket := "test-bucket"
	prefix := "test-folder/test-file.txt"
	local := "/tmp"
	region := "us-east-1"

	mockS3.On("GetObject", mock.Anything, mock.Anything).
		Return((*s3.GetObjectOutput)(nil), errors.New("erro ao baixar do S3"))

	configGlobal := app.ConfigGlobal{Region: region}
	err := app.DownloadObject(&configGlobal, mockS3, bucket, prefix, "", local, region)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Erro ao baixar arquivo do S3")

	mockS3.AssertExpectations(t)
}

func TestListObjects(t *testing.T) {
	mockS3 := new(MockS3Client)
	bucket := "test-bucket"
	prefix := "test-prefix/"
	delimiter := "/"
	showVersion := true

	mockObjects := &s3.ListObjectsV2Output{
		Contents: []s3Types.Object{
			{Key: &prefix},
		},
	}
	mockS3.On("ListObjectsV2", mock.Anything, mock.Anything).Return(mockObjects, nil)

	mockVersions := &s3.ListObjectVersionsOutput{
		Versions: []s3Types.ObjectVersion{
			{VersionId: aws.String("v1"), LastModified: aws.Time(time.Now())},
		},
	}
	mockS3.On("ListObjectVersions", mock.Anything, mock.Anything).Return(mockVersions, nil)

	// Captura a saída
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app.ListObjects(bucket, "us-east-1", prefix, delimiter, showVersion, mockS3)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	assert.Contains(t, output, prefix)
	assert.Contains(t, output, "Versões anteriores:")
	assert.Contains(t, output, "Versão ID: v1")

	mockS3.AssertExpectations(t)
}

package tests

import (
	"aws-s3-actions/app"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Mocks3Client struct {
	mock.Mock
}

// AbortMultipartUpload implements app.S3Uploader.
func (m *Mocks3Client) AbortMultipartUpload(ctx context.Context, input *s3.AbortMultipartUploadInput, opts ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.AbortMultipartUploadOutput), args.Error(1)
}

// CompleteMultipartUpload implements app.S3Uploader.
func (m *Mocks3Client) CompleteMultipartUpload(ctx context.Context, input *s3.CompleteMultipartUploadInput, opts ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.CompleteMultipartUploadOutput), args.Error(1)
}

// CreateMultipartUpload implements app.S3Uploader.
func (m *Mocks3Client) CreateMultipartUpload(ctx context.Context, input *s3.CreateMultipartUploadInput, opts ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.CreateMultipartUploadOutput), args.Error(1)
}

// UploadPart implements app.S3Uploader.
func (m *Mocks3Client) UploadPart(ctx context.Context, input *s3.UploadPartInput, opts ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.UploadPartOutput), args.Error(1)
}

func (m *Mocks3Client) PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func TestUploadSingle_Success(t *testing.T) {
	// Cria um arquivo temporário para testes
	file, err := os.CreateTemp("", "testefile")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	// Escreve um conteúdo no arquivo
	content := []byte("arquivo de teste")
	_, err = file.Write(content)
	assert.NoError(t, err)

	file.Close()

	// Reabre arquivo para leitura
	file, err = os.Open(file.Name())
	assert.NoError(t, err)
	defer file.Close()

	// Configura mock
	mocks3Client := new(Mocks3Client)
	configBackup := app.ConfigBackup{Bucket: "bucket-test"}
	s3Key := "key-test"

	// Debug para verificar valores antes do mock ser chamado
	fmt.Printf("🚀 Iniciando teste: Upload de arquivo %s para o bucket %s com chave %s\n", file.Name(), configBackup.Bucket, s3Key)

	mocks3Client.On("PutObject", mock.Anything, mock.AnythingOfType("*s3.PutObjectInput")).Return(&s3.PutObjectOutput{}, nil)

	// canal de erros
	errChan := make(chan error, 1)
	defer close(errChan)

	app.UploadSingle(file, mocks3Client, configBackup, s3Key, errChan)

	select {
	case err := <-errChan:
		assert.NoError(t, err, "Erro durante o upload")
	case <-time.After(5 * time.Second):
		t.Error("Timeout: upload demorou muito para completar")
	default:

	}

	// Verifica mock
	mocks3Client.AssertExpectations(t)

}

func TestUploadSingle_Failure(t *testing.T) {
	// Criar um arquivo temporário
	file, err := os.CreateTemp("", "testefile")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.Write([]byte("arquivo de teste"))
	assert.NoError(t, err)

	file.Close()

	file, err = os.Open(file.Name())
	assert.NoError(t, err)
	defer file.Close()

	mockS3 := new(Mocks3Client)
	configBackup := app.ConfigBackup{Bucket: "bucket-test"}
	s3Key := "key-test"

	// Configura o mock para falha no upload
	mockS3.On("PutObject", mock.Anything, mock.AnythingOfType("*s3.PutObjectInput")).Return((*s3.PutObjectOutput)(nil), assert.AnError)

	errChan := make(chan error, 1)

	// Executa o upload
	app.UploadSingle(file, mockS3, configBackup, s3Key, errChan)

	// Verifica se um erro foi enviado ao canal
	select {
	case err := <-errChan:
		assert.Error(t, err)
	default:
		assert.Fail(t, "Esperava erro, mas não houve nenhum")
	}

	mockS3.AssertExpectations(t)
}

func TestUploadMultipart_Success(t *testing.T) {
	// Criar um arquivo temporário para simular o upload
	file, err := os.CreateTemp("", "test-multipart")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	// Escrever dados no arquivo para simular um arquivo grande (> 10 MB)
	file.Write(bytes.Repeat([]byte("A"), 15*1024*1024)) // 15MB
	file.Close()
	file, _ = os.Open(file.Name())

	// Mock do S3
	mockS3 := new(Mocks3Client)
	configBackup := app.ConfigBackup{Bucket: "bucket-test"}
	s3Key := "key-test"
	fileSize := int64(15 * 1024 * 1024)

	// Configura o mock para um upload bem-sucedido
	mockS3.On("CreateMultipartUpload", mock.Anything, mock.Anything).Return(&s3.CreateMultipartUploadOutput{UploadId: aws.String("upload-123")}, nil)

	mockS3.On("UploadPart", mock.Anything, mock.Anything).Return(&s3.UploadPartOutput{ETag: aws.String("etag-part")}, nil).Times(2)

	mockS3.On("CompleteMultipartUpload", mock.Anything, mock.Anything).Return(&s3.CompleteMultipartUploadOutput{}, nil)

	errChan := make(chan error, 2)

	// Executa o upload
	app.UploadMultipart(file, fileSize, mockS3, configBackup, s3Key, errChan)

	// Verifica se o upload foi bem-sucedido
	select {
	case err := <-errChan:
		assert.Fail(t, "Erro inesperado: %v", err)
	default:
		t.Log("✅ Multipart Upload realizado com sucesso")
	}

	// Verifica se os mocks foram chamados corretamente
	mockS3.AssertExpectations(t)
}

func TestUploadMultipart_FailCreateUpload(t *testing.T) {
	file, _ := os.CreateTemp("", "test-multipart")
	defer os.Remove(file.Name())

	mockS3 := new(Mocks3Client)
	configBackup := app.ConfigBackup{Bucket: "bucket-test"}
	s3Key := "key-test"
	fileSize := int64(15 * 1024 * 1024)

	// Simula falha ao iniciar o Multipart Upload
	mockS3.On("CreateMultipartUpload", mock.Anything, mock.Anything).Return((*s3.CreateMultipartUploadOutput)(nil), errors.New("erro ao criar upload"))

	errChan := make(chan error, 1)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.UploadMultipart(file, fileSize, mockS3, configBackup, s3Key, errChan)
	}()
	wg.Wait()

	select {
	case err := <-errChan:
		assert.Error(t, err, "erro ao criar upload")
	default:
		assert.Fail(t, "Esperava erro ao criar upload, mas não ocorreu")
	}
}

func TestUploadMultipart_FailUploadPart(t *testing.T) {
	file, _ := os.CreateTemp("", "test-multipart")
	defer os.Remove(file.Name())

	mockS3 := new(Mocks3Client)
	configBackup := app.ConfigBackup{Bucket: "bucket-test"}
	s3Key := "key-test"
	fileSize := int64(15 * 1024 * 1024)

	// Simula sucesso ao iniciar o Multipart Upload
	mockS3.On("CreateMultipartUpload", mock.Anything, mock.Anything).Return(&s3.CreateMultipartUploadOutput{UploadId: aws.String("upload-123")}, nil)

	// Simula erro no envio da primeira parte
	mockS3.On("UploadPart", mock.Anything, mock.Anything).Return((*s3.UploadPartOutput)(nil), errors.New("erro ao enviar parte 1"))

	// Simula abort do upload
	mockS3.On("AbortMultipartUpload", mock.Anything, mock.Anything).Return(&s3.AbortMultipartUploadOutput{}, nil)

	errChan := make(chan error, 1)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.UploadMultipart(file, fileSize, mockS3, configBackup, s3Key, errChan)
	}()
	wg.Wait()

	select {
	case err := <-errChan:
		assert.Error(t, err, "erro ao enviar parte 1")
	default:
		assert.Fail(t, "Esperava erro no envio da parte 1, mas não ocorreu")
	}

	mockS3.AssertCalled(t, "AbortMultipartUpload", mock.Anything, mock.Anything)
}

func TestUploadMultipart_FailCompleteUpload(t *testing.T) {
	file, _ := os.CreateTemp("", "test-multipart")
	defer os.Remove(file.Name())

	mockS3 := new(Mocks3Client)
	configBackup := app.ConfigBackup{Bucket: "bucket-test"}
	s3Key := "key-test"
	fileSize := int64(15 * 1024 * 1024)

	mockS3.On("CreateMultipartUpload", mock.Anything, mock.Anything).Return(&s3.CreateMultipartUploadOutput{UploadId: aws.String("upload-123")}, nil)

	mockS3.On("UploadPart", mock.Anything, mock.Anything).Return(&s3.UploadPartOutput{ETag: aws.String("etag-part")}, nil).Times(2)

	// Simula falha ao completar o upload
	mockS3.On("CompleteMultipartUpload", mock.Anything, mock.Anything).Return((*s3.CompleteMultipartUploadOutput)(nil), errors.New("erro ao finalizar upload"))

	// Simula abort do upload
	mockS3.On("AbortMultipartUpload", mock.Anything, mock.Anything).Return(&s3.AbortMultipartUploadOutput{}, nil)

	errChan := make(chan error, 1)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.UploadMultipart(file, fileSize, mockS3, configBackup, s3Key, errChan)
	}()
	wg.Wait()

	select {
	case err := <-errChan:
		assert.Error(t, err, "erro ao finalizar upload")
	default:
		assert.Fail(t, "Esperava erro ao finalizar upload, mas não ocorreu")
	}

	mockS3.AssertCalled(t, "AbortMultipartUpload", mock.Anything, mock.Anything)
}

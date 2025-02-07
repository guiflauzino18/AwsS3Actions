package tests

// import (
// 	"aws-s3-actions/app"
// 	"context"
// 	"errors"
// 	"os"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/aws/aws-sdk-go-v2/service/s3"
// )

// // Mock do clientS3
// type MockS3Client struct{}

// // Funções do Mock
// func (m *MockS3Client) CreateMultipartUpload(ctx context.Context, input *s3.CreateMultipartUploadInput,
// 	opts ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {

// 	if input.Bucket == nil || input.Key == nil {
// 		return nil, errors.New("Parâmetros inválidos")
// 	}
// 	return &s3.CreateMultipartUploadOutput{UploadId: input.Key}, nil
// }

// func (m *MockS3Client) UploadPart(ctx context.Context, input *s3.UploadPartInput,
// 	opts ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
// 	if input.Body == nil || input.UploadId == nil || input.PartNumber == nil {
// 		return nil, errors.New("dados do upload inválidos")
// 	}
// 	return &s3.UploadPartOutput{ETag: input.UploadId}, nil
// }

// func (m *MockS3Client) CompletedMultipartUpload(ctx context.Context, input *s3.CompleteMultipartUploadInput,
// 	opts ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
// 	if input.UploadId == nil || len(input.MultipartUpload.Parts) == 0 {
// 		return nil, errors.New("upload incompleto")
// 	}
// 	return &s3.CompleteMultipartUploadOutput{}, nil
// }

// func createMockS3Client() *MockS3Client {
// 	return &MockS3Client{}
// }

// // Testa a função Multipart Upload
// func TestUploadMultipart(t *testing.T) {

// 	bucket := "test-bucket"
// 	s3Key := "test-file.txt"

// 	// Criando arquivo temporário
// 	tempFile, err := os.CreateTemp("", "testfile-*.txt")
// 	if err != nil {
// 		t.Fatalf("Erro ao criar arquivo temporário: %v", err)
// 	}
// 	defer os.Remove(tempFile.Name())

// 	// Escrevendo dados no arquivo
// 	content := []byte("Teste de upload multipart")
// 	if _, err := tempFile.Write(content); err != nil {
// 		t.Fatalf("Erro ao escrever no arquivo temporário: %v", err)
// 	}

// 	if err := tempFile.Close(); err != nil {
// 		t.Errorf("Erro ao fechar o arquivo temporário")
// 	}

// 	// Abrindo arquivo para leitura
// 	file, err := os.Open(tempFile.Name())
// 	if err != nil {
// 		t.Fatalf("Erro ao abrir o arquivo temporário: %v", err)
// 	}
// 	defer file.Close()

// 	s3Client := createMockS3Client()
// 	configBackup := app.ConfigBackup{Bucket: bucket}
// 	errCh := make(chan error, 1)

// 	// Goroutine para rodar o teste
// 	var wg sync.WaitGroup
// 	wg.Add(1)

// 	go func() {
// 		defer wg.Done()
// 		app.UploadMultipart(file, configBackup, s3Key)
// 	}()

// 	select {
// 	case err := <-errCh:
// 		if err != nil {
// 			t.Errorf("Upload Multipart Falhou: %v", err)
// 		}

// 	case <-time.After(10 * time.Second):
// 		t.Errorf("Timeout: Upload demorou muito para completar!")
// 	}

// 	wg.Wait()

// }

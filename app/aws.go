package app

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/urfave/cli"
)

func BackupRun(c *cli.Context) {
	// lê arquivo json de perfil de backup
	file, err := os.Open("profile/" + c.String("profile") + ".json")
	if err != nil {
		log.Fatal("Erro ao ler o arquivo de perfil de backup.")
	}
	defer file.Close()

	// Cria um struct de configBackup com os dados do json
	var configBackup ConfigBackup
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&configBackup); err != nil {
		log.Fatalf("Erro ao ler Perfil de Backup:\n%v", err)
	}

	// Recupera s3Client
	s3Client, _, err := CreateS3Client(configBackup.Region)
	if err != nil {
		log.Fatalf("Erro ao criar s3Client: %v", err)
	}

	// Criando um canal para distribuir os arquivos
	fileCh := make(chan string)
	numWorkers := 10 // Define o número de workers
	var wg sync.WaitGroup

	// Inicia os workers antes de enviar os arquivos para o canal
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go workerPool(fileCh, &wg, &configBackup, *s3Client)
	}

	fmt.Println("Adicionando arquivos para envio...")
	err = filepath.Walk(configBackup.SourceFolder, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		//envia o arquivo atual para o canal
		fileCh <- path

		return nil

	})
	if err != nil {
		log.Fatalf("Erro ao fazer backup: %v", err)
	}

	close(fileCh) // Fecha o canal após adicionar os arquivos

	wg.Wait()
}

// Workerpool para envio dos arquivos
func workerPool(fileCh chan string, wg *sync.WaitGroup, configBackup *ConfigBackup, s3Client s3.Client) {
	defer wg.Done()
	errCh := make(chan error, 5) // Canal de erros

	go func() {
		defer wg.Done()
		for file := range fileCh { // Cada worker processa arquivos do canal

			uploadObjetct(file, *configBackup, s3Client, errCh)
		}
	}()

	wg.Wait() // Aguarda todos os uploads terminarem
	defer close(errCh)

	// Exibindo erros, se houver
	for err := range errCh {
		log.Println("❌ Erro:", err)
	}

	fmt.Println("===========================================================================")
	fmt.Println("Envios dos arquivo concluídos!")
	fmt.Println("===========================================================================")
}

// FAz upload de objetos usando concorrencias
func uploadObjetct(path string, configBackup ConfigBackup, s3Client s3.Client, errCh chan error) error {

	realPath, err := filepath.Rel(configBackup.SourceFolder, path)
	if err != nil {
		errCh <- fmt.Errorf("Erro ao pegar caminho completo do arquivo")
	}

	//Cria nome do objeto pegando o prefix passado e o realpath
	s3Key := filepath.Join(configBackup.S3Prefix, realPath)

	//Pega o arquivo e adicionar em file
	file, err := os.Open(path)
	if err != nil {
		errCh <- fmt.Errorf("Erro ao abrir arquivo")
	}
	defer file.Close()

	// Obtém o tamanho do arquivo
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()

	// Decide o método de upload
	if fileSize < 10*1024*1024 { // < 10 MB
		uploadSingle(file, &s3Client, configBackup, s3Key, errCh)
	} else {
		UploadMultipart(file, fileSize, &s3Client, configBackup, s3Key, errCh)
	}

	return nil
}

// Upload normal para arquivos pequenos
func uploadSingle(file *os.File, s3Client *s3.Client, configBackup ConfigBackup, s3Key string, errCh chan error) {

	// Calcula o hash MD5 localmente
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		errCh <- fmt.Errorf("Erro ao cacular o MD5 do arquivo %s", file)
	}

	localMD5 := fmt.Sprintf("\"%x\"", hash.Sum(nil))

	result, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &configBackup.Bucket,
		Key:    &s3Key,
		Body:   file,
	})
	if err != nil {
		errCh <- fmt.Errorf("❌ Erro no upload do arquivo: (%s): %v", s3Key, err)
	} else {
		// Verifica a integridade do upload
		if result.ETag != &localMD5 {
			errCh <- fmt.Errorf("Erro de integridade: Etag do S3 (%s) diferente do MD5 local (%s) para o arquivo %s", *result.ETag, localMD5, file)
		}
		fmt.Printf("✅ Arquivo %s enviado para %s\n", s3Key, configBackup.Bucket)
	}

}

// Multipart Upload para arquivos grandes
func UploadMultipart(file *os.File, fileSize int64, s3Client *s3.Client, configBackup ConfigBackup, s3Key string, errCh chan error) {
	//Inicia o Multipart Upload
	resp, err := s3Client.CreateMultipartUpload(context.TODO(), &s3.CreateMultipartUploadInput{
		Bucket: &configBackup.Bucket,
		Key:    &s3Key,
	})
	if err != nil {
		errCh <- fmt.Errorf("Erro ao iniciar Multipart Upload: %v", err)
		return
	}
	uploadID := *resp.UploadId

	// Define o tamanho das partes
	partSize := int64(10 * 1024 * 1024) // 10 MB
	totalParts := (fileSize + partSize - 1) / partSize

	// Upload das partes em paralelo
	var wg sync.WaitGroup
	var mu sync.Mutex
	var parts []types.CompletedPart

	errChan := make(chan error, totalParts)

	// Máximo de tentativas de enviar uma parte que deu erro.
	const maxRetries = 3

	// Progresso do envio
	progressChan := make(chan int64)
	go func() {
		totalUploaded := int64(0)
		for uploaded := range progressChan {
			totalUploaded += uploaded
			percentage := float64(totalUploaded) / float64(fileSize) * 100
			fmt.Printf("\r🚀 Progresso: %.2f%%", percentage)
		}
	}()

	for partNumber := int64(1); partNumber <= totalParts; partNumber++ {
		wg.Add(1)
		go func(partNumber int64) {
			defer wg.Done()

			// Controle de tentativas de envios com falha
			var partResp *s3.UploadPartOutput
			var err error
			var attempt int

			//Converter Partnumber para int32
			pNum := int32(partNumber)

			for attempt = 0; attempt < maxRetries; attempt++ {

				// Calcula o deslocamento correto da parte
				offset := (partNumber - 1) * partSize
				partBuffer := make([]byte, partSize)

				// Garante que cada parte leia do offset correto
				mu.Lock()
				_, err := file.Seek(offset, io.SeekStart)
				if err != nil {
					mu.Unlock()
					errChan <- fmt.Errorf("Erro ao buscar parte %d: %v", partNumber, err)
					return
				}
				n, err := file.Read(partBuffer)
				mu.Unlock()

				if err != nil && err != io.EOF {
					errChan <- fmt.Errorf("Erro ao ler parte %d: %v", partNumber, err)
					return
				}

				// Envia a parte
				partResp, err = s3Client.UploadPart(context.TODO(), &s3.UploadPartInput{
					Bucket:     &configBackup.Bucket,
					Key:        &s3Key,
					UploadId:   &uploadID,
					PartNumber: &pNum,
					Body:       bytes.NewReader(partBuffer[:n]),
				})
				if err != nil {
					errChan <- fmt.Errorf("Erro ao enviar parte %d: %v", partNumber, err)
					return
				}

				// Cada parte enviada atualiza o progresso
				progressChan <- int64(n)

				if err == nil {
					break
				}

				log.Printf("Erro ao enviar parte %d (tentativa %d): %v", partNumber, attempt+1, err)
				time.Sleep(time.Duration(1<<attempt) * time.Second)

			}

			if err != nil {
				errCh <- fmt.Errorf("Falha ao enviar parte %d após %d tentativas: %v", partNumber, maxRetries, err)
			}
			// Armazena informações da parte
			mu.Lock()
			parts = append(parts, types.CompletedPart{
				ETag:       partResp.ETag,
				PartNumber: &pNum,
			})
			mu.Unlock()

		}(partNumber)
	}

	wg.Wait()
	close(errChan)
	close(progressChan)

	// Verifica se houve erro
	for err := range errChan {
		abortMultipartUpload(s3Client, configBackup, s3Key, uploadID)
		errCh <- fmt.Errorf(err.Error())
		return
	}

	// Ordena as partes antes de finalizar o upload
	sort.Slice(parts, func(i, j int) bool {
		return *parts[i].PartNumber < *parts[j].PartNumber
	})

	// Finaliza o upload
	_, err = s3Client.CompleteMultipartUpload(context.TODO(), &s3.CompleteMultipartUploadInput{
		Bucket:   &configBackup.Bucket,
		Key:      &s3Key,
		UploadId: &uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		abortMultipartUpload(s3Client, configBackup, s3Key, uploadID)
		errCh <- fmt.Errorf("Erro ao finalizar Multipart Upload: %v", err)
		return
	}

	fmt.Printf("✅ Arquivo %s enviado para %s\n!", s3Key, configBackup.Bucket)
}

// Aborta Multipart Upload em caso de erro
func abortMultipartUpload(s3Client *s3.Client, configBackup ConfigBackup, s3Key, uploadID string) {
	_, _ = s3Client.AbortMultipartUpload(context.TODO(), &s3.AbortMultipartUploadInput{
		Bucket:   &configBackup.Bucket,
		Key:      &s3Key,
		UploadId: &uploadID,
	})
	fmt.Printf("❌ Multipart Upload abortado para %s\n", s3Key)
}

// Função para download de objetos
func downloadObject(configGlobal *ConfigGlobal, s3Client s3.Client, bucket, prefix, version, local, region string) error {

	nomeArquivo := strings.Split(prefix, "/")
	fmt.Println("Fazendo download do arquivo: " + nomeArquivo[len(nomeArquivo)-1])

	file, err := os.Create(local + "/" + nomeArquivo[len(nomeArquivo)-1])
	if err != nil {
		return fmt.Errorf("Erro ao criar arquivo local: %v", err)
	}
	defer file.Close()

	// Cria input e caso a versão seja especificada é passada na requisição.
	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &prefix,
	}
	if version != "" {
		input.VersionId = &version
	}

	// Se region não for especificado pega da conf padrao
	if region == "" {
		region = configGlobal.Region
	}

	resp, err := s3Client.GetObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("Erro ao baixar arquivo do S3: %v", err)
	}
	defer resp.Body.Close()

	// Progresso do download
	totalSize := resp.ContentLength
	progressChan := make(chan int64)

	go func() {
		totalBaixado := int64(0)
		for baixado := range progressChan {
			totalBaixado += baixado
			porcent := float64(totalBaixado) / float64(*totalSize) * 100
			fmt.Printf("\r Progresso: %.2f%%", porcent)
		}
	}()

	buf := make([]byte, 1024*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				return fmt.Errorf("Erro ao escrever no arquivo local: %v", err)
			}
			progressChan <- int64(n)
		}
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("Erro ao ler dados do Bucket: %v", err)
		}

	}
	close(progressChan)

	fmt.Printf("\n✅ Download concluído: %s\n", local+nomeArquivo[len(nomeArquivo)-1])

	return nil

}

func ListObjects(bucket, region, prefix, delimiter string, showVersion bool, s3Client s3.Client) {

	input := &s3.ListObjectsV2Input{
		Bucket:    &bucket,
		Prefix:    &prefix,
		Delimiter: &delimiter,
	}

	result, erro := s3Client.ListObjectsV2(context.TODO(), input)
	if erro != nil {
		log.Fatal(erro)
	}

	// Exibe objetos
	for _, objeto := range result.Contents {

		fmt.Println("====================================================================================")
		fmt.Println(*objeto.Key)
		if showVersion {
			fmt.Println("Versões anteriores: ")
			versoes, erro := s3Client.ListObjectVersions(context.TODO(), &s3.ListObjectVersionsInput{
				Bucket:    &bucket,
				Prefix:    objeto.Key,
				Delimiter: &delimiter,
			})
			if erro != nil {
				log.Fatalf("Erro ao listar versões do objeto: %v", erro)
			}

			for index, versao := range versoes.Versions {
				fmt.Printf("Versão: %d\n Versão ID: %s\n Última Modificação: %v\n", index, *versao.VersionId, versao.LastModified)
			}
		}
		fmt.Println("====================================================================================")

	}

}

// Cria Client S3
func CreateS3Client(region string) (*s3.Client, ConfigGlobal, error) {

	var configGlobal *ConfigGlobal

	// Recupera a chave de criptografia
	key, err := os.ReadFile(os.ExpandEnv("conf/.aws_key"))
	if err != nil {
		fmt.Printf("Execute 'aws-s3-actions configure' para definir as configurações padrão.\n", err)
		return nil, *configGlobal, err
	}

	// REcupera o arquivo com os dados
	configGlobal, err = LoadCredentials("conf/config_global.enc", key)
	if err != nil {
		fmt.Println("Execute 'aws-s3-actions configure' para definir as configurações padrão.")
		return nil, *configGlobal, err
	}

	if region == "" {
		region = configGlobal.Region
	}

	cfg, erro := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			configGlobal.AccessKey, configGlobal.SecretKey, "",
		)),
		config.WithRegion(region),
	)

	if erro != nil {
		return nil, *configGlobal, err
	}

	return s3.NewFromConfig(cfg), *configGlobal, nil
}

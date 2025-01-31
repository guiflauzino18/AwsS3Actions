package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/urfave/cli"
)

func BackupRun(c *cli.Context) {
	file, err := os.Open("profile/" + c.String("profile") + ".json")
	if err != nil {
		log.Fatal("Erro ao ler o arquivo de perfil de backup.")
	}
	defer file.Close()

	var configBackup ConfigBackup
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&configBackup); err != nil {
		log.Fatalf("Erro ao ler Perfil de Backup:\n%v", err)
	}

	// Recupera a chave de criptografia
	key, err := os.ReadFile(os.ExpandEnv("$HOME/.aws_key"))
	if err != nil {
		log.Fatal("Execute 'aws-s3-actions configure' para definir as configurações padrão.\n", err)
	}

	// REcupera o arquivo com os dados
	configGlobal, err := LoadCredentials("config_global.enc", key)
	if err != nil {
		fmt.Println("Execute 'aws-s3-actions configure' para definir as configurações padrão.")
		log.Fatal(err)
	}

	// Cria um S3Cliente
	s3Client := createS3Client(configGlobal, configBackup.Region)

	err = filepath.Walk(configBackup.SourceFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		realPath, err := filepath.Rel(configBackup.SourceFolder, path)
		if err != nil {
			return err
		}

		s3Key := filepath.Join(configBackup.S3Prefix, realPath)
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: &configBackup.Bucket,
			Key:    &s3Key,
			Body:   file,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Arquivo %s enviado para %s/%s\n", path, configBackup.Bucket, s3Key)
		return nil
	})

	if err != nil {
		log.Fatalf("Erro ao fazer backup: %v", err)
	}

	fmt.Println("Backup concluído com sucesso!")
}

func ListObjects(c *cli.Context) {
	// Recupera parametros passados
	bucket := c.String("bucket")
	region := c.String("region")
	prefix := c.String("prefix")
	showVersion := c.Bool("show-version")
	var delimiter string

	// Recupera a chave de criptografia
	key, err := os.ReadFile(os.ExpandEnv("$HOME/.aws_key"))
	if err != nil {
		log.Fatal("Execute 'aws-s3-actions configure' para definir as configurações padrão.\n", err)
	}

	// REcupera o arquivo com os dados
	configGlobal, err := LoadCredentials("config_global.enc", key)
	if err != nil {
		fmt.Println("Execute 'aws-s3-actions configure' para definir as configurações padrão.")
		log.Fatal(err)
	}

	// Preenche bucket e region com valor padrão se não definido
	if bucket == "" {
		bucket = configGlobal.Bucket
	}

	if region == "" {
		region = configGlobal.Region
	}

	if prefix != "" {
		delimiter = "/"
	}

	// Cria um S3Cliente
	s3Client := createS3Client(configGlobal, region)

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
func createS3Client(configGlobal *ConfigGlobal, region string) *s3.Client {
	cfg, erro := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			configGlobal.AccessKey, configGlobal.SecretKey, "",
		)),
		config.WithRegion(region),
	)

	if erro != nil {
		log.Fatal(erro)
	}

	return s3.NewFromConfig(cfg)
}

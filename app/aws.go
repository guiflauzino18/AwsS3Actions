package app

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/urfave/cli"
)

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

package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli"
)

type ConfigGlobal struct {
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access-key"`
	SecretKey string `json:"secret-key"`
}

type ConfigBackup struct {
	Nome          string `json:"nome"`
	Region        string `json:"region"`
	Bucket        string `json:"bucket"`
	SourceFolder  string `json:"source_folder"`
	S3Prefix      string `json:"s3_prefix"`
	ArnAssumeRole string `json:"arn_assume_role"`
}

func GlobalConfigure(c *cli.Context) {

	var config ConfigGlobal

	fmt.Print("Região AWS: ")
	fmt.Scanln(&config.Region)

	fmt.Print("Bucket: ")
	fmt.Scanln(&config.Bucket)

	fmt.Print("AWS Access Key ID: ")
	fmt.Scanln(&config.AccessKey)

	fmt.Print("AWS Secret Access Key: ")
	fmt.Scanln(&config.SecretKey)

	// Gera e salva a chave e Criptografia
	SaveChave()

	// Criptografa o arquivo Json
	jsonCrypt(config)

}

func jsonCrypt(configGlobal ConfigGlobal) {
	// Carrega a chave de Criptografia
	key, err := os.ReadFile(os.ExpandEnv("conf/.aws_key"))
	if err != nil {
		log.Fatal("Erro ao carregar a chave de criptografia!")
	}

	data, err := json.Marshal(configGlobal)
	if err != nil {
		log.Fatal("Erro ao converter credenciais")
	}

	// Criptografa e salva o arquivo
	err = EncryptAndSave(data, key, "conf/config_global.enc")
	if err != nil {
		log.Fatal("Erro ao gerar arquivo de configuração!")
	}

	fmt.Println("Arquivo de configuração salvo com sucesso!")

}

func BackupConfigure(c *cli.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	var config ConfigBackup

	// Captura informações para gerar arquivo de conf
	fmt.Print("Nome do Backup (Sem espaços): ")
	scanner.Scan()
	config.Nome = scanner.Text()

	fmt.Print("Região AWS: ")
	scanner.Scan()
	config.Region = scanner.Text()

	fmt.Print("Nome do Bucket: ")
	scanner.Scan()
	config.Bucket = scanner.Text()

	fmt.Print("Pasta de origem: ")
	scanner.Scan()
	config.SourceFolder = scanner.Text()

	fmt.Print("Prefixo no S3 (ex: Backups/): ")
	scanner.Scan()
	config.S3Prefix = scanner.Text()

	// Verifica se a pasta profile existe e cria caso não existir
	if _, err := os.Stat("profile/"); os.IsNotExist(err) {
		err = os.MkdirAll("profile", os.ModePerm)
		if err != nil {
			fmt.Println("Erro ao criar a pasta profile.")
		}
	}

	file, err := os.Create("profile/" + config.Nome + ".json")
	if err != nil {
		log.Fatalf("Erro ao criar arquivo: %v", err)
	}
	defer file.Close()

	//Gera arquivo Json
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		log.Fatalf("Erro ao escrever no arquivo: %v", err)
	}

	fmt.Println("Perfil de Backup salvo com Sucesso!")
}

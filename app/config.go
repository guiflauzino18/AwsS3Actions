package app

import (
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
	key, err := os.ReadFile(os.ExpandEnv("$HOME/.aws_key"))
	if err != nil {
		log.Fatal("Erro ao carregar a chave de criptografia!")
	}

	data, err := json.Marshal(configGlobal)
	if err != nil {
		log.Fatal("Erro ao converter credenciais")
	}

	// Criptografa e salva o arquivo
	err = EncryptAndSave(data, key, "config_global.enc")
	if err != nil {
		log.Fatal("Erro ao gerar arquivo de configuração!")
	}

	fmt.Println("Arquivo de configuração salvo com sucesso!")

}

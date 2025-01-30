package app

import (
	"fmt"

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

	SaveCredentials(config)

}

package main

import (
	"aws-s3-actions/app"
	"log"
	"os"
)

func main() {
	app := app.Run()

	if erro := app.Run(os.Args); erro != nil {
		log.Fatal(erro)
	}
}

package app

/*
	Command Line terá todos as funções relacionados à linha de comando.
	As funções de cada comando ficará em um arquivo à parte relacionado à ação do comando.
*/

import (
	"github.com/urfave/cli"
)

func Run() *cli.App {
	app := cli.NewApp()
	app.Name = "AWS S3 ACTIONS"
	app.Usage = "Facilitando operações no S3 da Amazon"
	app.Version = "2025.1.0"

	app.Commands = []cli.Command{
		{
			Name:   "configure",
			Usage:  "Defina parâmetros básicos para o funcionamento da aplicação",
			Action: GlobalConfigure,
		},
		{
			Name:  "list",
			Usage: "Listar conteúdo de um bucket. Use --prefix para filtrar por pasta ou arquivo",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "bucket",
					Usage: "Nome do Bucket para lista o conteúdo",
				},
				cli.StringFlag{
					Name:  "prefix",
					Usage: "Filtrar objetos pelo começo do caminho. Ex.: Backup/",
				},
				cli.StringFlag{
					Name:  "region",
					Usage: "Região da AWS onde executar a requisição",
				},
				cli.BoolFlag{
					Name:  "show-version",
					Usage: "Exibir versões antesriores de objetos",
				},
			},
			Action: ListObjects,
		},
	}

	return app

}

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
		{
			Name:  "backup",
			Usage: "Configure e faça backup de pastas e arquivos no Amazon S3",
			Subcommands: []cli.Command{
				{
					Name:   "configure",
					Usage:  "Configura um novo perfil de backup.",
					Action: BackupConfigure,
				},
				{
					Name:      "run",
					Usage:     "Executa um backup.",
					UsageText: "aws-s3-action backup run --profile nomeProfile",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:     "profile",
							Usage:    "nome do perfil de backup para executar",
							Required: true,
						},
					},
					Action: BackupRun,
				},
			},
		},
		{
			Name:      "restore",
			Usage:     "Download de arquivos do S3 para o ambiente local",
			UsageText: "aws-s3-actions restore --prefix --local [--version] [--bucket] [--region]\n \n   Ex.: aws-s3-actions restore --prefix Backup/Pasta/arquivo.txt --local /root --version 5ez3MI7Z4vLeEFn7i9SJHSynkEfjx8iG\n \n   Use aws-s3-actions list --prefix --show-version para listar objeto e suas versões",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "prefix",
					Usage:    "Nome do objeto para download",
					Required: true,
				},
				cli.StringFlag{
					Name:  "version",
					Usage: "Opcional, versão do objeto para download. Se não especificado baixa a versão mais recente.",
				},
				cli.StringFlag{
					Name:  "bucket",
					Usage: "Opcional, se não informado será usado o bucket padrão",
				},
				cli.StringFlag{
					Name:     "local",
					Usage:    "Caminho local para salvar o objeto.",
					Required: true,
				},
				cli.StringFlag{
					Name:  "region",
					Usage: "Opcional, se não informado é usado região padrão.",
				},
			},
			Action: RestoreObject,
		},
	}

	return app

}

package app

/*
	Command Line terá todos as funções relacionados à linha de comando.
	As funções de cada comando ficará em um arquivo à parte relacionado à ação do comando.
*/

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/urfave/cli"
)

// Menu do terminal
func Run() *cli.App {
	app := cli.NewApp()
	app.Name = "AWS S3 ACTIONS"
	app.Usage = "Facilitando operações no S3 da Amazon"
	app.Version = "v1.1.15"

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
			Action: ListObjectsCommand,
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
					Action: backupRun,
				},
				{
					Name:   "list",
					Usage:  "Lista perfis de Backup configurados.",
					Action: listBackup,
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

func ListObjectsCommand(c *cli.Context) {
	// Recupera parametros passados
	bucket := c.String("bucket")
	region := c.String("region")
	prefix := c.String("prefix")
	showVersion := c.Bool("show-version")
	var delimiter string

	s3Client, configGlobal, err := CreateS3Client(region)
	if err != nil {
		log.Fatal("Execute 'aws-s3-actions configure' para definir as configurações padrão.")
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

	ListObjects(bucket, region, prefix, delimiter, showVersion, s3Client)
}

// Restore de objetos
func RestoreObject(c *cli.Context) {
	//Recupera parâmetros do comando
	bucket := c.String("bucket")
	prefix := c.String("prefix")
	version := c.String("version")
	local := c.String("local")
	region := c.String("region")

	s3Client, configGlobal, err := CreateS3Client(region)
	if err != nil {
		fmt.Println("Execute 'aws-s3-actions configure' para definir as configurações padrão.")
		log.Fatal(err.Error())
	}

	// Atribui valores padrão se não for passado
	if bucket == "" {
		bucket = configGlobal.Bucket
	}

	err = DownloadObject(&configGlobal, s3Client, bucket, prefix, version, local, region)
	if err != nil {
		fmt.Printf("Erro no Download do Arquivo:\n %v", err)
	}

}

// Upload de Objetos
func backupRun(c *cli.Context) {
	// lê arquivo json de perfil de backup
	file, err := os.Open("/usr/local/aws-s3-actions/profile/" + c.String("profile") + ".json")
	if err != nil {
		log.Fatal("Erro ao ler o arquivo de perfil de backup.\n Execute backup configure para configurar.")
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
		fmt.Println("Execute 'aws-s3-actions configure' para definir as configurações padrão.")
		log.Fatal(err)
	}

	// Criando um canal para distribuir os arquivos
	fileCh := make(chan string)
	numWorkers := 10 // Define o número de workers
	var wg sync.WaitGroup

	// Inicia os workers antes de enviar os arquivos para o canal
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go WorkerPool(fileCh, &wg, &configBackup, *s3Client)
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

	fmt.Println("===========================================================================")
	fmt.Println("Envios dos arquivo concluídos!")
	fmt.Println("===========================================================================")
}

// Lista Backups Configurados
func listBackup(c *cli.Context) {
	type BackupProfile struct {
		Nome         string `json:"nome"`
		Region       string `json:"region"`
		Bucket       string `json:"bucket"`
		SourceFolder string `json:"source_folder"`
		S3Prefix     string `json:"s3_prefix"`
	}

	profileDir := "/usr/local/aws-s3-actions/profile/"

	var backupProfiles []BackupProfile

	err := filepath.WalkDir(profileDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Erro ao abrir o arquivo %s", file.Name())
		}
		defer file.Close()

		var backupProfile BackupProfile
		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&backupProfile); err != nil {
			return fmt.Errorf("Erro ao decodificar json do arquivo %s: \n%v", path, err)
		}

		backupProfiles = append(backupProfiles, backupProfile)

		return nil

	})

	if err != nil {
		log.Fatalf("Erro ao listar backups: \n%v", err)
	}

	fmt.Println("\nPerfis de backup:\n")
	for _, profile := range backupProfiles {
		fmt.Printf("Nome: %s\nBucket: %s\nRegião: %s\nPasta de Origem: %s\nPrefixo de Destino: %s\n=============================================================\n",
			profile.Nome, profile.Bucket, profile.Region, profile.SourceFolder, profile.S3Prefix)
	}

	fmt.Printf("\nPara editar um perfil, edite o arquivo json em %s\n", profileDir)

}

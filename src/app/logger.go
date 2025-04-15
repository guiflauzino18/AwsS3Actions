package app

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var LogGlobal = "aws-s3-actions-global"

func ConfiguraLogger(backupProfileName string) (*logrus.Logger, *os.File, error) {

	//Caminho do log
	logCaminho := "/var/log/aws-s3-actions/"
	logName := fmt.Sprintf("%s.log", backupProfileName)
	logCaminhoCompleto := fmt.Sprintf("%s%s", logCaminho, logName)

	//Criar pastas do caminho de log
	erro := os.MkdirAll(logCaminho, os.ModePerm)
	if erro != nil {
		return nil, nil, fmt.Errorf("Erro ao criar pasta de log: %v", erro)
	}

	// Cria arquivo de log
	logFile, erro := os.OpenFile(logCaminhoCompleto, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if erro != nil {
		return nil, nil, fmt.Errorf("Erro ao criar arquivo de log: %v", erro)
	}

	//Configura Logrus
	logger := logrus.New()
	logger.SetOutput(logFile)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	return logger, logFile, nil

}

package app

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var LogGlobal = "aws-s3-actions-global"

type timestampHook struct {
	loc *time.Location
}

func (h *timestampHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *timestampHook) Fire(entry *logrus.Entry) error {
	entry.Time = entry.Time.In(h.loc)
	return nil
}

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

	//Carrega fuso horário
	loc, erro := time.LoadLocation("Local")
	if erro != nil {
		return nil, nil, fmt.Errorf("Erro ao carregar timezone: %v", erro)
	}

	//Configura Logrus
	logger := logrus.New()
	logger.SetOutput(logFile)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "02-01-2006 15:04:05",
	})
	logger.AddHook(&timestampHook{loc: loc})
	logger.SetLevel(logrus.InfoLevel)

	return logger, logFile, nil

}

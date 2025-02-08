package app

/*

	Aqui ficará todos as funções e dados de credenciais de acessos.

*/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Descriptografa o arquivo para pegar os valores
func LoadCredentials(encFile string, key []byte) (*ConfigGlobal, error) {
	encryptedData, err := os.ReadFile(encFile)
	if err != nil {

		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {

		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {

		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	nonce, cipherText := encryptedData[:nonceSize], encryptedData[nonceSize:]

	plainData, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {

		return nil, err
	}

	var confGlobal ConfigGlobal
	err = json.Unmarshal(plainData, &confGlobal)
	if err != nil {

		return nil, err
	}

	return &confGlobal, nil
}

// Criptografa o arquivo de conf e salva em arquivo
func EncryptAndSave(data, key []byte, outputFile string) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	nonce := make([]byte, 12) //GCM usa nonce de 12 bytes
	_, err = rand.Read(nonce)
	if err != nil {
		return err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	encryptedData := aesGCM.Seal(nonce, nonce, data, nil)
	return os.WriteFile(outputFile, encryptedData, 0600)
}

// salva chave de Criptografia
func SaveChave() {

	key, erro := generateEncryptionKey()
	if erro != nil {
		fmt.Println("Erro ao gerar Chave de Criptografia!", erro)
		log.Fatal()
	}

	// Verifica se a pasta conf existe e cria caso não existir
	if _, err := os.Stat("conf/"); os.IsNotExist(err) {
		err = os.MkdirAll("conf", os.ModePerm)
		if err != nil {
			fmt.Println("Erro ao criar a pasta conf.")
		}
	}

	erro = saveKeyToFile(key, os.ExpandEnv("conf/.aws_key"))
	if erro != nil {
		fmt.Println("Erro ao salvar a chave em arquivo")
		log.Fatal()
	}

	fmt.Println("Chave de Criptografia salva!")
	fmt.Println("chave (hex): ", hex.EncodeToString(key))

}

// Gera chave aleatória de criptografia
func generateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32) //256bits
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// Salva a chave de criptografia gerada em um arquivo
func saveKeyToFile(key []byte, filePath string) error {
	return os.WriteFile(filePath, key, 0600)
}

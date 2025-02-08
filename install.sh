#!/bin/bash

# Nome do repositório no GitHub (troque pelo seu)
REPO="seu-usuario/seu-repo"

# Nome do binário gerado pelo Go
BIN="minha-app-linux"

# URL do último release no GitHub
LATEST_RELEASE=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep "browser_download_url" | cut -d '"' -f 4)

# Verifica se encontrou um release
if [ -z "$LATEST_RELEASE" ]; then
    echo "Erro: Nenhum release encontrado para $REPO"
    exit 1
fi

echo "Baixando a última versão de $REPO..."
curl -L -o $BIN "$LATEST_RELEASE"

echo "Dando permissão de execução..."
chmod +x $BIN

echo "Movendo para /usr/local/bin/..."
sudo mv $BIN /usr/local/bin/$BIN

echo "Instalação concluída! Agora você pode executar '$BIN' de qualquer lugar."

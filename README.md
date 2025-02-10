# AWS S3 ACTIONS
<h3>Facilitando Operações do S3 da Amazon</h3>

AWS S3 Actions tem a função de facilitar as operações de backup e restore de arquivos hospedados no S3 para sistemas Linux.

- Fácil configuração Inicial
- Sem necessidade da AWS CLI ou de arquivos de configuração em ~/.aws/credentials
- Credenciais criptografadas e não salvas em plain-text
- Restore de uma versão específica de um objeto
- Fácil configuração de Backup

# Documentação
<h3>Instalação:</h3>

`curl -s https://raw.githubusercontent.com/guiflauzino18/AwsS3Actions/refs/heads/Main/install.sh | bash`

<h3>Pré Ajustes:</h3>

- Crie uma conta no IAM da AWS com no mínimo as seguintes permissões:
 "s3:PutObject",
 "s3:GetObject",
 "s3:AbortMultipartUpload",
 "s3:DeleteObjectVersion",
 "s3:GetObjectAttributes",
 "s3:DeleteObject",
 "s3:GetObjectVersion",
 "s3:ListMultipartUploadParts",
 "s3:ListBucket",
 "s3:ListBucketVersions"
- Gere um Access Key ID e o Secret Access Key para a conta.
- Execute `aws-s3-actions configure` para configurar a autenticação e definir alguns valores Default. Os dados informados aqui são armazenados em um arquivo criptografado.
- Teste executando `aws-s3-actions list` e se houver objetos armazenados no bucket passado na configuração padrão eles serão listados.

<h3>Listando objetos</h3>

- Execute `aws-s3-actions list` para listar todos os objetos do bucket.<br>

  * --bucket <bucket_name> - Opcional, nome do bucket a listar objetos. Se não informado é usado bucket padrão<br>
  * --prefix <caminho_do_objeto> - Opcional, filtra por nome ou caminho do objeto<br>
  * --show-version <true> - Opcional, exibe versões anterioes dos objetos <br>

<h3>Configurando upload</h3>

- Execute `aws-s3-actions backup configure` e informe as opções de configuração de backup <br>

  * Nome do Backup (Sem espaços): _meu_backup_
  * Região AWS: _Ex.: us-east-2_
  * Bucket: _Ex.: meu_bucket_
  * Pasta de Origem: _Ex.: /home/_
  * Prefixo no S3: _Ex.: Backup/Home_

- Execute `aws-s3-actions backup run --profile <meu_backup>` para executar o backup <br>

<h3>Download de Objetos</h3>

- Execute `aws-s3-actions restore --prefix <caminho_do_objeto> --local <Ex.: /root/>` para realizar o download do objeto para a pasta local.

  * --bucket <bucket_name> - Opcional, especifica o bucket do objeto <br>
  * --version <Version_ID> - Opcional, baixa uma versão específica do objeto <br>
  * --region <regiao> - Opcional, especifica uma região <br>

# Códigos de saída

- Sucesso: 0
- Erro: 1
- Use echo $? para ver o status de saída

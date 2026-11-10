package main

// Convenções compartilhadas pelas stacks do infra-base.
// Todas as stacks (e o CDK dos outros repositórios, como referência) devem
// reaproveitar estes valores em vez de repetir strings.

const (
	// Org é a organização GitHub dona de todos os repositórios.
	Org = "Rossete-Inc"

	// Projeto é o prefixo usado em nomes de stacks e recursos.
	Projeto = "gestor-gastos"

	// Regiao é a única região usada pelo sistema.
	Regiao = "us-east-1"

	// GitHubOIDCHost é o emissor dos tokens OIDC do GitHub Actions.
	GitHubOIDCHost = "token.actions.githubusercontent.com"

	// GitHubOIDCAudience é a audience configurada pelo aws-actions/configure-aws-credentials.
	GitHubOIDCAudience = "sts.amazonaws.com"

	// RepoFrontend é o repositório do app, que também faz deploy no Amplify.
	RepoFrontend = "gestor-gastos-frontend"
)

// Ambientes implantáveis. Cada um corresponde a um GitHub Environment de mesmo nome.
// Por custo, o projeto tem um único ambiente na AWS (decisão de 2026-11-10, ADR 0007).
var Ambientes = []string{"dev"}

// ReposImplantaveis são os repositórios que fazem deploy na AWS e, por isso,
// recebem uma role por ambiente. A go-commons não entra: ela só publica tags.
var ReposImplantaveis = []string{
	"gestor-gastos-infra-base",
	"gestor-gastos-users-service",
	"gestor-gastos-categories-service",
	"gestor-gastos-expenses-service",
	"gestor-gastos-bff",
	RepoFrontend,
}

// ReposComCdk são os repositórios que rodam `cdk diff` nos PRs e, por isso,
// recebem uma role só leitura. O frontend publica no Amplify sem CDK.
var ReposComCdk = []string{
	"gestor-gastos-infra-base",
	"gestor-gastos-users-service",
	"gestor-gastos-categories-service",
	"gestor-gastos-expenses-service",
	"gestor-gastos-bff",
}

// NomeRoleDiff devolve o nome da role só leitura usada pelo `cdk diff` em PRs.
// Ex.: gh-diff-gestor-gastos-bff.
func NomeRoleDiff(repo string) string {
	return "gh-diff-" + repo
}

// SubjectOIDCPullRequest é a claim `sub` que o GitHub emite para jobs de
// pull_request sem `environment:`.
func SubjectOIDCPullRequest(repo string) string {
	return "repo:" + Org + "/" + repo + ":pull_request"
}

// NomeRoleDeploy devolve o nome da role de deploy de um repositório em um ambiente.
// Ex.: gh-deploy-gestor-gastos-bff-dev.
func NomeRoleDeploy(repo, env string) string {
	return "gh-deploy-" + repo + "-" + env
}

// SubjectOIDC devolve o valor exato da claim `sub` aceita na trust policy.
// O GitHub só emite esse formato quando o job declara `environment: <env>`.
func SubjectOIDC(repo, env string) string {
	return "repo:" + Org + "/" + repo + ":environment:" + env
}

// PrefixoSSM devolve o caminho base dos parâmetros de um ambiente no SSM,
// sem barra final. Ex.: /gestor-gastos/dev.
func PrefixoSSM(env string) string {
	return "/" + Projeto + "/" + env
}

const (
	// ChaveContextoAmbiente é a chave de contexto do CDK com o ambiente
	// (`cdk deploy --context env=dev`), a mesma que o go-service.yml passa.
	ChaveContextoAmbiente = "env"

	// AmbientePadrao é usado quando o contexto `env` não é informado.
	AmbientePadrao = "dev"

	// SourceAuth é o `source` dos eventos do trigger Post Confirmation. O
	// users-service filtra a regra do EventBridge por ele.
	SourceAuth = Projeto + ".auth"

	// GrupoAdmin e GrupoCliente são os grupos do User Pool, iguais aos do
	// cognito-local (local/seed/cognito.go).
	GrupoAdmin   = "admin"
	GrupoCliente = "cliente"

	// OrigemLocal é a origem do `expo start --web`, sempre liberada no CORS.
	OrigemLocal = "http://localhost:8081"
)

// Parâmetros publicados no SSM, relativos a PrefixoSSM(env). Os outros
// repositórios leem estes caminhos; mudar um nome quebra quem consome.
const (
	ParamUserPoolID       = "auth/user-pool-id"
	ParamUserPoolArn      = "auth/user-pool-arn"
	ParamUserPoolClientID = "auth/user-pool-client-id"
	ParamIssuerURL        = "auth/issuer-url"
	ParamJwksURL          = "auth/jwks-url"
	ParamBusName          = "events/bus-name"
	ParamBusArn           = "events/bus-arn"
	ParamDlqArn           = "events/dlq-arn"
	ParamDlqURL           = "events/dlq-url"
	ParamAPIID            = "api/id"
	ParamAPIURL           = "api/url"
	ParamAuthorizerID     = "api/authorizer-id"
)

// CaminhoSSM devolve o caminho completo de um parâmetro do ambiente.
// Ex.: CaminhoSSM("dev", ParamAPIURL) = /gestor-gastos/dev/api/url.
func CaminhoSSM(env, param string) string {
	return PrefixoSSM(env) + "/" + param
}

// NomeStack devolve o nome de uma stack por ambiente. Ex.: gestor-gastos-dev-auth.
func NomeStack(env, nome string) string {
	return Projeto + "-" + env + "-" + nome
}

// NomeBarramento devolve o nome do barramento do EventBridge do ambiente.
// Ex.: gestor-gastos-dev.
func NomeBarramento(env string) string {
	return Projeto + "-" + env
}

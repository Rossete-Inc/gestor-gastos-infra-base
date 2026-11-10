package main

// Convenções compartilhadas pelas stacks do infra-base.
// As tarefas 6.4 e 6.5 devem reaproveitar estes valores em vez de repetir strings.

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
var Ambientes = []string{"dev", "prod"}

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

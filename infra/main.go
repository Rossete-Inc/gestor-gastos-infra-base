// Command infra é o app AWS CDK (Go) do gestor-gastos-infra-base.
//
// Stacks da conta (não dependem de ambiente):
//   - gestor-gastos-cicd: OIDC provider do GitHub, roles gh-deploy-<repo>-<env> e
//     roles só leitura gh-diff-<repo> para o cdk diff em PRs.
//   - gestor-gastos-budget: orçamento mensal de US$ 5 com alertas por e-mail.
//
// As stacks por ambiente (auth, events, api) entram na tarefa 6.4, com nomes
// gestor-gastos-<env>-<stack> e parâmetros em /gestor-gastos/<env>/... no SSM.
package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

const (
	StackCicd   = Projeto + "-cicd"
	StackBudget = Projeto + "-budget"
)

// NovoApp monta o app com todas as stacks. Separado de main para os testes.
func NovoApp(app awscdk.App) {
	props := &awscdk.StackProps{Env: ambienteAws()}

	awscdk.Tags_Of(app).Add(jsii.String("projeto"), jsii.String(Projeto), nil)
	awscdk.Tags_Of(app).Add(jsii.String("repositorio"), jsii.String("gestor-gastos-infra-base"), nil)

	NovaStackCicd(app, StackCicd, props)
	NovaStackBudget(app, StackBudget, props)
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	NovoApp(app)
	app.Synth(nil)
}

// ambienteAws fixa a região em us-east-1. A conta vem das credenciais do CLI
// (CDK_DEFAULT_ACCOUNT) quando houver; sem credenciais a stack fica agnóstica de
// conta e as ARNs usam ${AWS::AccountId}, o que permite synth e testes offline.
func ambienteAws() *awscdk.Environment {
	env := &awscdk.Environment{Region: jsii.String(Regiao)}
	if conta := os.Getenv("CDK_DEFAULT_ACCOUNT"); conta != "" {
		env.Account = jsii.String(conta)
	}
	return env
}

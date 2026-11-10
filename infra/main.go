// Command infra é o app AWS CDK (Go) do gestor-gastos-infra-base.
//
// Stacks da conta (não dependem de ambiente):
//   - gestor-gastos-cicd: OIDC provider do GitHub, roles gh-deploy-<repo>-<env> e
//     roles só leitura gh-diff-<repo> para o cdk diff em PRs.
//   - gestor-gastos-budget: orçamento mensal de US$ 5 com alertas por e-mail.
//
// Stacks do ambiente (contexto `env`, padrão dev), na ordem de deploy:
//   - gestor-gastos-<env>-events: barramento do EventBridge e DLQ dos consumidores.
//   - gestor-gastos-<env>-auth: User Pool, grupos, app client e trigger Post Confirmation.
//   - gestor-gastos-<env>-web: app do Amplify Hosting (PWA), sem repositório conectado.
//   - gestor-gastos-<env>-api: HTTP API com authorizer JWT, CORS e throttling.
//
// Elas se comunicam só pelo SSM (/gestor-gastos/<env>/...), nunca por exports.
package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const (
	StackCicd   = Projeto + "-cicd"
	StackBudget = Projeto + "-budget"
)

// NovoApp monta o app com todas as stacks. Separado de main para os testes.
// dirLambda é a pasta do bootstrap do trigger (DirLambdaPostConfirmation).
func NovoApp(app awscdk.App, dirLambda string) {
	props := &awscdk.StackProps{Env: ambienteAws()}

	awscdk.Tags_Of(app).Add(jsii.String("projeto"), jsii.String(Projeto), nil)
	awscdk.Tags_Of(app).Add(jsii.String("repositorio"), jsii.String("gestor-gastos-infra-base"), nil)

	NovaStackCicd(app, StackCicd, props)
	NovaStackBudget(app, StackBudget, props)

	env := Ambiente(app)
	awscdk.Tags_Of(app).Add(jsii.String("ambiente"), jsii.String(env), nil)

	events := NovaStackEvents(app, NomeStack(env, "events"), env, props)
	auth := NovaStackAuth(app, NomeStack(env, "auth"), &AuthProps{
		StackProps:             *props,
		Ambiente:               env,
		CodigoPostConfirmation: CodigoLambda(dirLambda),
	})
	web := NovaStackWeb(app, NomeStack(env, "web"), env, props)
	api := NovaStackApi(app, NomeStack(env, "api"), &ApiProps{
		StackProps:    *props,
		Ambiente:      env,
		OrigemAmplify: OrigemAmplify(app),
	})

	// Só ordem de deploy (o trigger publica no barramento; a api lê o SSM da
	// auth e da web). AddStackDependency não cria exports.
	auth.AddStackDependency(events, jsii.String("o trigger publica no barramento"))
	api.AddStackDependency(auth, jsii.String("o authorizer lê o issuer e o client id do SSM"))
	api.AddStackDependency(web, jsii.String("o CORS lê a URL do site no SSM"))
}

// Ambiente lê o contexto `env` (padrão dev) e aceita só os de Ambientes.
func Ambiente(scope constructs.Construct) string {
	env := AmbientePadrao
	if v, ok := scope.Node().TryGetContext(jsii.String(ChaveContextoAmbiente)).(string); ok && v != "" {
		env = v
	}
	if !slices.Contains(Ambientes, env) {
		panic(fmt.Sprintf("ambiente %q inválido: use um de %v (contexto %s)", env, Ambientes, ChaveContextoAmbiente))
	}
	return env
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	NovoApp(app, DirLambdaPostConfirmation)
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

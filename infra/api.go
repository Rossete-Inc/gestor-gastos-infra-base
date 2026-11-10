package main

import (
	"os"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2authorizers"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const (
	// ChaveContextoOrigemAmplify é a chave de contexto com a origem do site no
	// Amplify (ex.: https://main.d1abc.amplifyapp.com), liberada no CORS. O app
	// do Amplify só existe a partir da tarefa 6.6; até lá, só o localhost.
	// Depois, grave o valor em infra/cdk.json para todo deploy usá-lo.
	ChaveContextoOrigemAmplify = "amplifyOrigin"

	// VarAmbienteOrigemAmplify é a alternativa ao contexto.
	VarAmbienteOrigemAmplify = "AMPLIFY_ORIGIN"

	// MsgOrigemAmplifyInvalida é o erro de synth para uma origem mal formada.
	MsgOrigemAmplifyInvalida = "stack api: " + ChaveContextoOrigemAmplify +
		" precisa ser uma origem https sem caminho nem barra final (ex.: https://main.d1abc.amplifyapp.com)"

	// RotaAuthCheck é a rota protegida que existe só para verificar o authorizer
	// antes de o BFF publicar as rotas dele. Sem token: 401.
	RotaAuthCheck = "/infra/auth-check"

	// ThrottleBurst e ThrottleRate são os limites do stage $default para todas
	// as rotas. Baixos de propósito: protegem o custo contra abuso.
	ThrottleBurst = 20
	ThrottleRate  = 10
)

// ApiProps configura a stack api.
type ApiProps struct {
	awscdk.StackProps
	Ambiente string
	// OrigemAmplify é a origem extra liberada no CORS, além de OrigemLocal. Vazia = só o localhost.
	OrigemAmplify string
}

// OrigemAmplify lê a origem do Amplify do contexto e, se não houver, da
// variável de ambiente AMPLIFY_ORIGIN.
func OrigemAmplify(scope constructs.Construct) string {
	if v, ok := scope.Node().TryGetContext(jsii.String(ChaveContextoOrigemAmplify)).(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(os.Getenv(VarAmbienteOrigemAmplify))
}

func origemValida(o string) bool {
	resto, ok := strings.CutPrefix(o, "https://")
	return ok && resto != "" && !strings.ContainsAny(resto, "/?#* ")
}

// NovaStackApi cria a HTTP API pública do ambiente, com o authorizer JWT do
// Cognito, CORS e throttling. As rotas de negócio são do BFF (tarefa 6.5), que
// as adiciona a esta API lendo o id da API e do authorizer no SSM.
//
// O issuer e o client id vêm do SSM (publicados pela stack auth) e são
// resolvidos no deploy, sem CloudFormation exports.
func NovaStackApi(scope constructs.Construct, id string, props *ApiProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), &props.StackProps)
	env := props.Ambiente

	origens := []string{OrigemLocal}
	if props.OrigemAmplify != "" {
		if !origemValida(props.OrigemAmplify) {
			awscdk.Annotations_Of(stack).AddError(jsii.String(MsgOrigemAmplifyInvalida))
		}
		origens = append(origens, props.OrigemAmplify)
	}

	issuer := lerParametro(stack, env, ParamIssuerURL)
	clientID := lerParametro(stack, env, ParamUserPoolClientID)

	api := awsapigatewayv2.NewHttpApi(stack, jsii.String("HttpApi"), &awsapigatewayv2.HttpApiProps{
		ApiName:     jsii.String(NomeStack(env, "api")),
		Description: jsii.String("API pública do Gestor de Gastos (" + env + "). Rotas publicadas pelo BFF."),
		// O stage $default é criado abaixo, para ter throttling.
		CreateDefaultStage: jsii.Bool(false),
		CorsPreflight: &awsapigatewayv2.CorsPreflightOptions{
			AllowOrigins: jsii.Strings(origens...),
			AllowMethods: &[]awsapigatewayv2.CorsHttpMethod{
				awsapigatewayv2.CorsHttpMethod_GET,
				awsapigatewayv2.CorsHttpMethod_POST,
				awsapigatewayv2.CorsHttpMethod_PUT,
				awsapigatewayv2.CorsHttpMethod_PATCH,
				awsapigatewayv2.CorsHttpMethod_DELETE,
				awsapigatewayv2.CorsHttpMethod_OPTIONS,
			},
			AllowHeaders: jsii.Strings("Authorization", "Content-Type"),
			MaxAge:       awscdk.Duration_Hours(jsii.Number(1)),
		},
	})

	awsapigatewayv2.NewHttpStage(stack, jsii.String("StageDefault"), &awsapigatewayv2.HttpStageProps{
		HttpApi:    api,
		StageName:  jsii.String("$default"),
		AutoDeploy: jsii.Bool(true),
		Throttle: &awsapigatewayv2.ThrottleSettings{
			BurstLimit: jsii.Number(ThrottleBurst),
			RateLimit:  jsii.Number(ThrottleRate),
		},
	})

	// Para access tokens, a HTTP API compara a audience com a claim client_id.
	authorizer := awsapigatewayv2authorizers.NewHttpJwtAuthorizer(jsii.String("CognitoJwt"), jsii.String(issuer),
		&awsapigatewayv2authorizers.HttpJwtAuthorizerProps{
			AuthorizerName: jsii.String("cognito-jwt"),
			JwtAudience:    jsii.Strings(clientID),
			IdentitySource: jsii.Strings("$request.header.Authorization"),
		})

	// Rota de verificação do authorizer. A HTTP API não tem integração mock,
	// e uma API sem rotas responde 404, não 401. Esta rota repassa para o JWKS
	// público do próprio pool (HTTP_PROXY, sem Lambda, sem custo parado):
	// sem token, o authorizer responde 401 antes da integração; com um token
	// válido, devolve as chaves públicas, o que também serve de smoke test do login.
	api.AddRoutes(&awsapigatewayv2.AddRoutesOptions{
		Path:    jsii.String(RotaAuthCheck),
		Methods: &[]awsapigatewayv2.HttpMethod{awsapigatewayv2.HttpMethod_GET},
		Integration: awsapigatewayv2integrations.NewHttpUrlIntegration(jsii.String("JwksDoPool"),
			jsii.String(issuer+"/.well-known/jwks.json"),
			&awsapigatewayv2integrations.HttpUrlIntegrationProps{Method: awsapigatewayv2.HttpMethod_GET}),
		Authorizer: authorizer,
	})

	publicarParametro(stack, env, ParamAPIID, *api.ApiId(), "Id da HTTP API pública")
	publicarParametro(stack, env, ParamAPIURL, *api.ApiEndpoint(), "URL base da HTTP API pública (stage $default)")
	publicarParametro(stack, env, ParamAuthorizerID, *authorizer.AuthorizerId(), "Id do authorizer JWT (Cognito) da HTTP API")

	return stack
}

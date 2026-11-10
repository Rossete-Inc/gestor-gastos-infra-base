package main

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
)

// Testes das stacks do ambiente (events, auth, api), da tarefa 6.4.
// Rodam sem AWS e sem compilar a Lambda: o bootstrap é um arquivo falso.

const envTeste = "dev"

// dirLambdaFake cria uma pasta com um bootstrap falso, para o asset da Lambda
// existir sem `task infra:build`.
func dirLambdaFake(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bootstrap"), []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// appTeste monta o app inteiro com o contexto informado e devolve as stacks.
func appTeste(t *testing.T, contexto map[string]interface{}) awscdk.App {
	t.Helper()
	t.Setenv(VarAmbienteEmailBudget, emailTeste)
	t.Setenv(VarAmbienteOrigemAmplify, "")
	app := awscdk.NewApp(&awscdk.AppProps{Context: &contexto})
	NovoApp(app, dirLambdaFake(t))
	return app
}

func stackDoApp(t *testing.T, app awscdk.App, nome string) awscdk.Stack {
	t.Helper()
	filho := app.Node().TryFindChild(jsii.String(nome))
	if filho == nil {
		t.Fatalf("stack %s não encontrada", nome)
	}
	return filho.(awscdk.Stack)
}

func templateDe(t *testing.T, nome string) assertions.Template {
	t.Helper()
	app := appTeste(t, map[string]interface{}{})
	return assertions.Template_FromStack(stackDoApp(t, app, nome), nil)
}

// logicalID devolve o id lógico do único recurso do tipo informado.
func logicalID(t *testing.T, tpl assertions.Template, tipo string) string {
	t.Helper()
	recursos := *tpl.FindResources(jsii.String(tipo), nil)
	if len(recursos) != 1 {
		t.Fatalf("esperava 1 recurso %s, achei %d", tipo, len(recursos))
	}
	for id := range recursos {
		return id
	}
	return ""
}

func exigirParametros(t *testing.T, tpl assertions.Template, params ...string) {
	t.Helper()
	tpl.ResourceCountIs(jsii.String("AWS::SSM::Parameter"), jsii.Number(len(params)))
	for _, p := range params {
		tpl.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]interface{}{
			"Name": "/gestor-gastos/dev/" + p,
			"Type": "String",
			"Tier": "Standard",
		})
	}
}

func TestNomesDasStacksDoAmbiente(t *testing.T) {
	app := appTeste(t, map[string]interface{}{})
	for _, nome := range []string{"gestor-gastos-dev-events", "gestor-gastos-dev-auth", "gestor-gastos-dev-api"} {
		if got := *stackDoApp(t, app, nome).StackName(); got != nome {
			t.Errorf("stackName = %s, quer %s", got, nome)
		}
	}
}

func TestOrdemDeDeploy(t *testing.T) {
	app := appTeste(t, map[string]interface{}{})
	depende := func(de, para string) bool {
		for _, d := range *stackDoApp(t, app, de).Dependencies() {
			if *d.StackName() == para {
				return true
			}
		}
		return false
	}
	if !depende("gestor-gastos-dev-auth", "gestor-gastos-dev-events") {
		t.Error("auth deve ser implantada depois de events")
	}
	if !depende("gestor-gastos-dev-api", "gestor-gastos-dev-auth") {
		t.Error("api deve ser implantada depois de auth")
	}
}

func TestAmbienteInvalidoFalha(t *testing.T) {
	defer func() {
		if r := recover(); r == nil || !strings.Contains(r.(string), "prod") {
			t.Fatalf("esperava panic citando o ambiente, veio %v", r)
		}
	}()
	appTeste(t, map[string]interface{}{ChaveContextoAmbiente: "prod"})
}

func TestStacksDoAmbienteSemExports(t *testing.T) {
	app := appTeste(t, map[string]interface{}{})
	for _, nome := range []string{"gestor-gastos-dev-events", "gestor-gastos-dev-auth", "gestor-gastos-dev-api"} {
		b, err := json.Marshal(assertions.Template_FromStack(stackDoApp(t, app, nome), nil).ToJSON())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), `"Export"`) || strings.Contains(string(b), "Fn::ImportValue") {
			t.Errorf("%s usa CloudFormation export/import, proibido (usar SSM)", nome)
		}
	}
}

// --- events ---

func TestEventsBarramentoEDlq(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-events")

	tpl.ResourceCountIs(jsii.String("AWS::Events::EventBus"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::Events::EventBus"), map[string]interface{}{"Name": "gestor-gastos-dev"})
	tpl.ResourceCountIs(jsii.String("AWS::Events::Archive"), jsii.Number(0))

	tpl.HasResourceProperties(jsii.String("AWS::SQS::Queue"), map[string]interface{}{
		"QueueName":              "gestor-gastos-dev-events-dlq",
		"MessageRetentionPeriod": 1209600,
		"SqsManagedSseEnabled":   true,
	})
}

func TestEventsDlqAceitaRegrasDoBarramento(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-events")

	tpl.HasResourceProperties(jsii.String("AWS::SQS::QueuePolicy"), map[string]interface{}{
		"PolicyDocument": map[string]interface{}{
			"Statement": assertions.Match_ArrayWith(&[]interface{}{
				assertions.Match_ObjectLike(&map[string]interface{}{
					"Sid":       "RegrasDoBarramentoEnviamParaADlq",
					"Effect":    "Allow",
					"Action":    "sqs:SendMessage",
					"Principal": map[string]interface{}{"Service": "events.amazonaws.com"},
					"Condition": map[string]interface{}{
						"ArnLike": map[string]interface{}{
							"aws:SourceArn": map[string]interface{}{
								"Fn::Join": []interface{}{"", []interface{}{
									"arn:", map[string]interface{}{"Ref": "AWS::Partition"},
									":events:", map[string]interface{}{"Ref": "AWS::Region"},
									":", map[string]interface{}{"Ref": "AWS::AccountId"},
									":rule/gestor-gastos-dev/*",
								}},
							},
						},
					},
				}),
			}),
		},
	})
}

func TestEventsAlarmeNaDlq(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-events")

	tpl.ResourceCountIs(jsii.String("AWS::CloudWatch::Alarm"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::CloudWatch::Alarm"), map[string]interface{}{
		"MetricName":         "ApproximateNumberOfMessagesVisible",
		"Namespace":          "AWS/SQS",
		"Threshold":          1,
		"ComparisonOperator": "GreaterThanOrEqualToThreshold",
		"TreatMissingData":   "notBreaching",
	})
}

func TestEventsParametrosNoSsm(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-events")
	exigirParametros(t, tpl, ParamBusName, ParamBusArn, ParamDlqArn, ParamDlqURL)
}

// --- auth ---

func TestAuthUserPoolIgualAoLocal(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")

	tpl.ResourceCountIs(jsii.String("AWS::Cognito::UserPool"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::Cognito::UserPool"), map[string]interface{}{
		"UserPoolName":           "gestor-gastos-dev-users",
		"UserPoolTier":           "LITE",
		"UsernameAttributes":     []interface{}{"email"},
		"AutoVerifiedAttributes": []interface{}{"email"},
		"AdminCreateUserConfig":  map[string]interface{}{"AllowAdminCreateUserOnly": false},
		"EmailConfiguration":     map[string]interface{}{"EmailSendingAccount": "COGNITO_DEFAULT"},
		"VerificationMessageTemplate": assertions.Match_ObjectLike(&map[string]interface{}{
			"DefaultEmailOption": "CONFIRM_WITH_CODE",
		}),
		"Policies": map[string]interface{}{
			"PasswordPolicy": map[string]interface{}{
				"MinimumLength":    8,
				"RequireUppercase": true,
				"RequireLowercase": true,
				"RequireNumbers":   true,
				"RequireSymbols":   false,
			},
		},
		"Schema": assertions.Match_ArrayEquals(&[]interface{}{
			map[string]interface{}{"Name": "email", "Required": true, "Mutable": true},
			map[string]interface{}{"Name": "name", "Required": true, "Mutable": true},
		}),
	})
}

func TestAuthUserPoolRetido(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")

	tpl.HasResource(jsii.String("AWS::Cognito::UserPool"), map[string]interface{}{
		"DeletionPolicy":      "Retain",
		"UpdateReplacePolicy": "Retain",
	})
}

func TestAuthGrupos(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")

	tpl.ResourceCountIs(jsii.String("AWS::Cognito::UserPoolGroup"), jsii.Number(2))
	for _, g := range []string{"admin", "cliente"} {
		tpl.HasResourceProperties(jsii.String("AWS::Cognito::UserPoolGroup"), map[string]interface{}{"GroupName": g})
	}
}

func TestAuthAppClientPublicoComSrpETokensCurtos(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")

	tpl.ResourceCountIs(jsii.String("AWS::Cognito::UserPoolClient"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::Cognito::UserPoolClient"), map[string]interface{}{
		"ClientName":                      "gestor-gastos-dev-frontend",
		"GenerateSecret":                  false,
		"ExplicitAuthFlows":               assertions.Match_ArrayEquals(&[]interface{}{"ALLOW_USER_SRP_AUTH", "ALLOW_REFRESH_TOKEN_AUTH"}),
		"AllowedOAuthFlowsUserPoolClient": false,
		"PreventUserExistenceErrors":      "ENABLED",
		"EnableTokenRevocation":           true,
		"AccessTokenValidity":             15,
		"IdTokenValidity":                 15,
		"RefreshTokenValidity":            43200, // 30 dias, em minutos
		"TokenValidityUnits": map[string]interface{}{
			"AccessToken":  "minutes",
			"IdToken":      "minutes",
			"RefreshToken": "minutes",
		},
	})
}

func TestAuthTriggerPostConfirmation(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")
	fn := logicalID(t, tpl, "AWS::Lambda::Function")

	tpl.HasResourceProperties(jsii.String("AWS::Cognito::UserPool"), map[string]interface{}{
		"LambdaConfig": map[string]interface{}{
			"PostConfirmation": map[string]interface{}{"Fn::GetAtt": []interface{}{fn, "Arn"}},
		},
	})
	tpl.HasResourceProperties(jsii.String("AWS::Lambda::Permission"), map[string]interface{}{
		"Action":       "lambda:InvokeFunction",
		"Principal":    "cognito-idp.amazonaws.com",
		"FunctionName": map[string]interface{}{"Fn::GetAtt": []interface{}{fn, "Arn"}},
	})
	tpl.HasResourceProperties(jsii.String("AWS::Lambda::Function"), map[string]interface{}{
		"FunctionName":  "gestor-gastos-dev-post-confirmation",
		"Runtime":       "provided.al2023",
		"Architectures": []interface{}{"arm64"},
		"Handler":       "bootstrap",
		"MemorySize":    128,
		"Timeout":       5,
		"Environment": map[string]interface{}{"Variables": map[string]interface{}{
			"RUNTIME":        "aws",
			"ENV":            "dev",
			"EVENT_BUS_NAME": "gestor-gastos-dev",
			"LOG_LEVEL":      "info",
		}},
	})
}

func TestAuthLogsDoTriggerPorSeteDias(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")

	tpl.HasResourceProperties(jsii.String("AWS::Logs::LogGroup"), map[string]interface{}{
		"LogGroupName":    "/aws/lambda/gestor-gastos-dev-post-confirmation",
		"RetentionInDays": 7,
	})
	grupo := logicalID(t, tpl, "AWS::Logs::LogGroup")
	tpl.HasResourceProperties(jsii.String("AWS::Lambda::Function"), map[string]interface{}{
		"LoggingConfig": map[string]interface{}{"LogGroup": map[string]interface{}{"Ref": grupo}},
	})
}

func TestAuthPermissoesDoTriggerSemDependenciaCircular(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")

	arn := func(servico, sufixo string) map[string]interface{} {
		return map[string]interface{}{"Fn::Join": []interface{}{"", []interface{}{
			"arn:", map[string]interface{}{"Ref": "AWS::Partition"},
			":" + servico + ":", map[string]interface{}{"Ref": "AWS::Region"},
			":", map[string]interface{}{"Ref": "AWS::AccountId"},
			sufixo,
		}}}
	}
	tpl.HasResourceProperties(jsii.String("AWS::IAM::Policy"), map[string]interface{}{
		"PolicyDocument": map[string]interface{}{
			"Statement": assertions.Match_ArrayEquals(&[]interface{}{
				map[string]interface{}{
					"Sid": "AdicionarAoGrupo", "Effect": "Allow",
					"Action":   "cognito-idp:AdminAddUserToGroup",
					"Resource": arn("cognito-idp", ":userpool/*"),
				},
				map[string]interface{}{
					"Sid": "PublicarNoBarramento", "Effect": "Allow",
					"Action":   "events:PutEvents",
					"Resource": arn("events", ":event-bus/gestor-gastos-dev"),
				},
			}),
		},
	})

	// A política não pode citar o pool: o pool já depende da Lambda (trigger).
	pool := logicalID(t, tpl, "AWS::Cognito::UserPool")
	b, err := json.Marshal(tpl.FindResources(jsii.String("AWS::IAM::Policy"), nil))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), pool) {
		t.Errorf("a política do trigger referencia o User Pool %s: dependência circular", pool)
	}
}

func TestAuthParametrosNoSsm(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-auth")
	exigirParametros(t, tpl, ParamUserPoolID, ParamUserPoolArn, ParamUserPoolClientID, ParamIssuerURL, ParamJwksURL)

	pool := logicalID(t, tpl, "AWS::Cognito::UserPool")
	tpl.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]interface{}{
		"Name": "/gestor-gastos/dev/auth/issuer-url",
		"Value": map[string]interface{}{"Fn::Join": []interface{}{"", []interface{}{
			"https://cognito-idp.", map[string]interface{}{"Ref": "AWS::Region"},
			".amazonaws.com/", map[string]interface{}{"Ref": pool},
		}}},
	})
}

func TestAuthSemBinarioFalhaComMensagemClara(t *testing.T) {
	app := awscdk.NewApp(nil)
	stack := NovaStackAuth(app, "gestor-gastos-dev-auth", &AuthProps{Ambiente: envTeste})

	assertions.Annotations_FromStack(stack).HasError(jsii.String("/gestor-gastos-dev-auth"), MsgFaltaBuildPostConfirmation)
}

func TestCodigoLambdaSoComBootstrap(t *testing.T) {
	if CodigoLambda(t.TempDir()) != nil {
		t.Error("pasta sem bootstrap deveria devolver nil")
	}
	if CodigoLambda(dirLambdaFake(t)) == nil {
		t.Error("pasta com bootstrap deveria devolver o asset")
	}
}

func TestZipDoBootstrapExecutavelEDeterministico(t *testing.T) {
	dir := dirLambdaFake(t)
	destino, err := zipBootstrap(dir)
	if err != nil {
		t.Fatal(err)
	}
	primeiro, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.OpenReader(destino)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 1 || zr.File[0].Name != "bootstrap" {
		t.Fatalf("conteúdo do zip: %v", zr.File)
	}
	if modo := zr.File[0].Mode().Perm(); modo != 0o755 {
		t.Errorf("modo do bootstrap = %o, quer 755 (a Lambda precisa executá-lo)", modo)
	}

	if _, err := zipBootstrap(dir); err != nil {
		t.Fatal(err)
	}
	segundo, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	if string(primeiro) != string(segundo) {
		t.Error("o zip muda a cada synth: o hash do asset geraria deploys à toa")
	}
}

// --- api ---

func TestApiAuthorizerJwtDoCognito(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-api")

	// Issuer e audience vêm do SSM, resolvidos no deploy.
	tpl.HasParameter(jsii.String("*"), map[string]interface{}{
		"Type":    "AWS::SSM::Parameter::Value<String>",
		"Default": "/gestor-gastos/dev/auth/issuer-url",
	})
	tpl.HasParameter(jsii.String("*"), map[string]interface{}{
		"Type":    "AWS::SSM::Parameter::Value<String>",
		"Default": "/gestor-gastos/dev/auth/user-pool-client-id",
	})

	issuer, client := "", ""
	for id, p := range *tpl.FindParameters(jsii.String("*"), nil) {
		switch (*p)["Default"] {
		case "/gestor-gastos/dev/auth/issuer-url":
			issuer = id
		case "/gestor-gastos/dev/auth/user-pool-client-id":
			client = id
		}
	}

	tpl.ResourceCountIs(jsii.String("AWS::ApiGatewayV2::Authorizer"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::ApiGatewayV2::Authorizer"), map[string]interface{}{
		"AuthorizerType": "JWT",
		"IdentitySource": []interface{}{"$request.header.Authorization"},
		"JwtConfiguration": map[string]interface{}{
			"Issuer":   map[string]interface{}{"Ref": issuer},
			"Audience": []interface{}{map[string]interface{}{"Ref": client}},
		},
	})
}

func TestApiRotaDeVerificacaoProtegida(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-api")
	authorizer := logicalID(t, tpl, "AWS::ApiGatewayV2::Authorizer")

	tpl.ResourceCountIs(jsii.String("AWS::ApiGatewayV2::Route"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::ApiGatewayV2::Route"), map[string]interface{}{
		"RouteKey":          "GET " + RotaAuthCheck,
		"AuthorizationType": "JWT",
		"AuthorizerId":      map[string]interface{}{"Ref": authorizer},
	})
	tpl.HasResourceProperties(jsii.String("AWS::ApiGatewayV2::Integration"), map[string]interface{}{
		"IntegrationType":   "HTTP_PROXY",
		"IntegrationMethod": "GET",
	})
	// Sem Lambda na stack api: a rota não custa nada parada.
	tpl.ResourceCountIs(jsii.String("AWS::Lambda::Function"), jsii.Number(0))
}

func TestApiCorsSoLocalhostPorPadrao(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-api")

	tpl.HasResourceProperties(jsii.String("AWS::ApiGatewayV2::Api"), map[string]interface{}{
		"Name":         "gestor-gastos-dev-api",
		"ProtocolType": "HTTP",
		"CorsConfiguration": map[string]interface{}{
			"AllowOrigins": []interface{}{"http://localhost:8081"},
			"AllowHeaders": []interface{}{"Authorization", "Content-Type"},
			"AllowMethods": []interface{}{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			"MaxAge":       3600,
		},
	})
}

func TestApiCorsComOrigemDoAmplify(t *testing.T) {
	origem := "https://main.d1abc.amplifyapp.com"
	app := appTeste(t, map[string]interface{}{ChaveContextoOrigemAmplify: origem})
	tpl := assertions.Template_FromStack(stackDoApp(t, app, "gestor-gastos-dev-api"), nil)

	tpl.HasResourceProperties(jsii.String("AWS::ApiGatewayV2::Api"), map[string]interface{}{
		"CorsConfiguration": assertions.Match_ObjectLike(&map[string]interface{}{
			"AllowOrigins": []interface{}{"http://localhost:8081", origem},
		}),
	})
}

func TestApiOrigemDoAmplifyPelaVariavelDeAmbiente(t *testing.T) {
	t.Setenv(VarAmbienteOrigemAmplify, "https://main.d9xyz.amplifyapp.com")
	app := awscdk.NewApp(nil)
	if got := OrigemAmplify(app); got != "https://main.d9xyz.amplifyapp.com" {
		t.Errorf("OrigemAmplify = %q", got)
	}
}

func TestApiOrigemInvalidaFalha(t *testing.T) {
	for _, origem := range []string{"http://main.d1abc.amplifyapp.com", "https://main.d1abc.amplifyapp.com/", "https://*.amplifyapp.com", "https://"} {
		t.Run(origem, func(t *testing.T) {
			app := awscdk.NewApp(nil)
			stack := NovaStackApi(app, "gestor-gastos-dev-api", &ApiProps{Ambiente: envTeste, OrigemAmplify: origem})
			assertions.Annotations_FromStack(stack).HasError(jsii.String("/gestor-gastos-dev-api"), MsgOrigemAmplifyInvalida)
		})
	}
}

func TestApiThrottlingNoStageDefault(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-api")

	tpl.ResourceCountIs(jsii.String("AWS::ApiGatewayV2::Stage"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::ApiGatewayV2::Stage"), map[string]interface{}{
		"StageName":  "$default",
		"AutoDeploy": true,
		"DefaultRouteSettings": map[string]interface{}{
			"ThrottlingBurstLimit": ThrottleBurst,
			"ThrottlingRateLimit":  ThrottleRate,
		},
	})
}

func TestApiParametrosNoSsm(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-api")
	exigirParametros(t, tpl, ParamAPIID, ParamAPIURL, ParamAuthorizerID)

	api := logicalID(t, tpl, "AWS::ApiGatewayV2::Api")
	tpl.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]interface{}{
		"Name":  "/gestor-gastos/dev/api/url",
		"Value": map[string]interface{}{"Fn::GetAtt": []interface{}{api, "ApiEndpoint"}},
	})
}

package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscognito"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const (
	// DirLambdaPostConfirmation é onde fica o binário do trigger, a partir de
	// infra/. Segue a convenção do go-service.yml: <dist-dir>/<pasta>/bootstrap,
	// gerado por `task infra:build` a partir de ./cmd/post-confirmation.
	DirLambdaPostConfirmation = "../dist/post-confirmation"

	// MsgFaltaBuildPostConfirmation é o erro de synth quando o binário não existe.
	MsgFaltaBuildPostConfirmation = "stack auth: o binário do trigger Post Confirmation não existe em " +
		DirLambdaPostConfirmation + "/bootstrap. Rode `task infra:build` (ou `task infra:synth`, que já compila)."
)

// AuthProps configura a stack auth.
type AuthProps struct {
	awscdk.StackProps
	Ambiente string
	// CodigoPostConfirmation é o pacote da Lambda do trigger. nil = binário
	// ausente: a stack registra MsgFaltaBuildPostConfirmation e não é sintetizada.
	CodigoPostConfirmation awslambda.Code
}

// CodigoLambda empacota <dir>/bootstrap em <dir>.zip e devolve esse zip como
// asset, ou nil se o bootstrap não existir.
//
// O zip é montado aqui, e não pelo CLI do CDK, por causa do Windows: o CLI
// copia para o zip o modo do arquivo, e no Windows o bootstrap não tem bit de
// execução, então a Lambda falharia com "permission denied". Aqui o modo é
// sempre 0755 e a data é fixa, para o hash do asset não mudar à toa.
func CodigoLambda(dir string) awslambda.Code {
	destino, err := zipBootstrap(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		panic(err)
	}
	return awslambda.Code_FromAsset(jsii.String(destino), nil)
}

func zipBootstrap(dir string) (string, error) {
	bin, err := os.ReadFile(filepath.Join(dir, "bootstrap"))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	cab := &zip.FileHeader{Name: "bootstrap", Method: zip.Deflate, Modified: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)}
	cab.SetMode(0o755)
	w, err := zw.CreateHeader(cab)
	if err == nil {
		_, err = w.Write(bin)
	}
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		return "", fmt.Errorf("empacotar %s: %w", dir, err)
	}

	destino := filepath.Clean(dir) + ".zip"
	if err := os.WriteFile(destino, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("gravar %s: %w", destino, err)
	}
	return destino, nil
}

// NovaStackAuth cria o User Pool (o mesmo do cognito-local: e-mail como login,
// senha de 8 com maiúscula, minúscula e número, grupos admin e cliente), o app
// client público com SRP e o trigger Post Confirmation.
//
// O pool fica com RemovalPolicy RETAIN: apagar a stack não apaga os usuários.
func NovaStackAuth(scope constructs.Construct, id string, props *AuthProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), &props.StackProps)
	env := props.Ambiente

	if props.CodigoPostConfirmation == nil {
		awscdk.Annotations_Of(stack).AddError(jsii.String(MsgFaltaBuildPostConfirmation))
		return stack
	}

	trigger := novoTriggerPostConfirmation(stack, env, props.CodigoPostConfirmation)

	pool := awscognito.NewUserPool(stack, jsii.String("UserPool"), &awscognito.UserPoolProps{
		UserPoolName: jsii.String(NomeStack(env, "users")),
		// Lite: o plano mais barato (sem custo até 10 mil MAUs), com tudo o que o MVP usa
		// (SRP, grupos e triggers). Sem isso, pools novos nascem no Essentials.
		FeaturePlan:         awscognito.FeaturePlan_LITE,
		SignInAliases:       &awscognito.SignInAliases{Email: jsii.Bool(true)},
		SignInCaseSensitive: jsii.Bool(false),
		SelfSignUpEnabled:   jsii.Bool(true),
		AutoVerify:          &awscognito.AutoVerifiedAttrs{Email: jsii.Bool(true)},
		UserVerification: &awscognito.UserVerificationConfig{
			EmailStyle:   awscognito.VerificationEmailStyle_CODE,
			EmailSubject: jsii.String("Gestor de Gastos: código de verificação"),
			EmailBody:    jsii.String("Seu código de verificação é {####}"),
		},
		StandardAttributes: &awscognito.StandardAttributes{
			Email:    &awscognito.StandardAttribute{Required: jsii.Bool(true), Mutable: jsii.Bool(true)},
			Fullname: &awscognito.StandardAttribute{Required: jsii.Bool(true), Mutable: jsii.Bool(true)},
		},
		PasswordPolicy: &awscognito.PasswordPolicy{
			MinLength:        jsii.Number(8),
			RequireUppercase: jsii.Bool(true),
			RequireLowercase: jsii.Bool(true),
			RequireDigits:    jsii.Bool(true),
			RequireSymbols:   jsii.Bool(false),
		},
		AccountRecovery: awscognito.AccountRecovery_EMAIL_ONLY,
		// E-mail padrão do Cognito (limite de ~50/dia), sem SES: suficiente para o MVP.
		Email:          awscognito.UserPoolEmail_WithCognito(nil),
		LambdaTriggers: &awscognito.UserPoolTriggers{PostConfirmation: trigger},
		RemovalPolicy:  awscdk.RemovalPolicy_RETAIN,
	})

	awscognito.NewUserPoolGroup(stack, jsii.String("GrupoAdmin"), &awscognito.UserPoolGroupProps{
		UserPool:    pool,
		GroupName:   jsii.String(GrupoAdmin),
		Description: jsii.String("Administradores: gerenciam usuários"),
	})
	awscognito.NewUserPoolGroup(stack, jsii.String("GrupoCliente"), &awscognito.UserPoolGroupProps{
		UserPool:    pool,
		GroupName:   jsii.String(GrupoCliente),
		Description: jsii.String("Clientes: lançam e consultam os próprios gastos"),
	})

	// Client público (o app roda no navegador): sem secret, só SRP e refresh.
	// Sem OAuth/Hosted UI: o login é feito pelo próprio app.
	client := pool.AddClient(jsii.String("AppClient"), &awscognito.UserPoolClientOptions{
		UserPoolClientName: jsii.String(NomeStack(env, "frontend")),
		GenerateSecret:     jsii.Bool(false),
		// AdminUserPassword só é aceito com credenciais IAM (AdminInitiateAuth):
		// serve ao smoke test da esteira e não abre nada para o app público.
		AuthFlows:                  &awscognito.AuthFlow{UserSrp: jsii.Bool(true), AdminUserPassword: jsii.Bool(true)},
		DisableOAuth:               jsii.Bool(true),
		PreventUserExistenceErrors: jsii.Bool(true),
		EnableTokenRevocation:      jsii.Bool(true),
		AccessTokenValidity:        awscdk.Duration_Minutes(jsii.Number(15)),
		IdTokenValidity:            awscdk.Duration_Minutes(jsii.Number(15)),
		RefreshTokenValidity:       awscdk.Duration_Days(jsii.Number(30)),
	})

	issuer := "https://cognito-idp." + *awscdk.Aws_REGION() + ".amazonaws.com/" + *pool.UserPoolId()

	publicarParametro(stack, env, ParamUserPoolID, *pool.UserPoolId(), "Id do User Pool do Cognito")
	publicarParametro(stack, env, ParamUserPoolArn, *pool.UserPoolArn(), "ARN do User Pool do Cognito")
	publicarParametro(stack, env, ParamUserPoolClientID, *client.UserPoolClientId(), "Id do app client público (SRP) do frontend")
	publicarParametro(stack, env, ParamIssuerURL, issuer, "Issuer dos JWT do Cognito")
	publicarParametro(stack, env, ParamJwksURL, issuer+"/.well-known/jwks.json", "JWKS do Cognito, para validar os JWT")

	return stack
}

// novoTriggerPostConfirmation cria a Lambda que põe o usuário no grupo cliente
// e publica UserConfirmed.
//
// Dependência circular evitada: o pool referencia a Lambda (trigger), e a
// Lambda depende da própria política. Se a política citasse o ARN do pool,
// o CloudFormation teria pool -> Lambda -> política -> pool. Por isso a
// permissão AdminAddUserToGroup usa um ARN montado sem o pool:
// userpool/* da conta e da região.
func novoTriggerPostConfirmation(stack awscdk.Stack, env string, codigo awslambda.Code) awslambda.Function {
	nome := NomeStack(env, "post-confirmation")
	barramento := NomeBarramento(env)
	conta, regiao, particao := *awscdk.Aws_ACCOUNT_ID(), *awscdk.Aws_REGION(), *awscdk.Aws_PARTITION()

	logs := awslogs.NewLogGroup(stack, jsii.String("PostConfirmationLogs"), &awslogs.LogGroupProps{
		LogGroupName:  jsii.String("/aws/lambda/" + nome),
		Retention:     awslogs.RetentionDays_ONE_WEEK,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	fn := awslambda.NewFunction(stack, jsii.String("PostConfirmation"), &awslambda.FunctionProps{
		FunctionName: jsii.String(nome),
		Description:  jsii.String("Trigger Post Confirmation: grupo cliente + evento UserConfirmed"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Architecture: awslambda.Architecture_ARM_64(),
		Handler:      jsii.String("bootstrap"),
		Code:         codigo,
		MemorySize:   jsii.Number(128),
		// O Cognito espera o trigger por no máximo 5 s.
		Timeout:  awscdk.Duration_Seconds(jsii.Number(5)),
		LogGroup: logs,
		Environment: &map[string]*string{
			"RUNTIME":        jsii.String("aws"),
			"ENV":            jsii.String(env),
			"EVENT_BUS_NAME": jsii.String(barramento),
			"LOG_LEVEL":      jsii.String("info"),
		},
	})

	fn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("AdicionarAoGrupo"),
		Actions:   jsii.Strings("cognito-idp:AdminAddUserToGroup"),
		Resources: jsii.Strings("arn:" + particao + ":cognito-idp:" + regiao + ":" + conta + ":userpool/*"),
	}))
	fn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("PublicarNoBarramento"),
		Actions:   jsii.Strings("events:PutEvents"),
		Resources: jsii.Strings("arn:" + particao + ":events:" + regiao + ":" + conta + ":event-bus/" + barramento),
	}))

	return fn
}

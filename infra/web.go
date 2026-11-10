package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsamplify"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const (
	// BranchAmplify é a branch do app no Amplify que recebe os deploys do
	// frontend.yml (input amplify-branch, padrão main). Não tem relação com Git:
	// o app não tem repositório conectado.
	BranchAmplify = "main"

	// RegraSpaOrigem é o rewrite de SPA: toda rota sem extensão de arquivo
	// estático vai para o index.html com status 200. Arquivos com as extensões
	// listadas (bundles, ícones, fontes, manifest, sw.js etc.) são servidos como
	// estão, e os ausentes dão 404 de verdade em vez do index.html, o que mantém
	// o service worker e o manifest da PWA (tarefa 11.1) funcionando.
	RegraSpaOrigem = `</^[^.]+$|\.(?!(css|gif|ico|jpg|jpeg|js|png|txt|svg|webp|woff|woff2|ttf|otf|map|json|webmanifest)$)([^.]+$)/>`

	// RegraSpaDestino é o destino do rewrite de SPA.
	RegraSpaDestino = "/index.html"

	// HeadersAmplify são os headers de cache do site (formato do customHttp.yml).
	// sw.js, manifest e index.html nunca ficam em cache longo, senão o usuário
	// fica preso numa versão velha do app; os bundles têm hash no nome e podem
	// ser imutáveis. Em deploy manual o Amplify não lê o customHttp.yml do
	// artefato, por isso os headers ficam no app.
	HeadersAmplify = `customHeaders:
  - pattern: '/sw.js'
    headers:
      - key: 'Cache-Control'
        value: 'no-cache'
  - pattern: '/manifest.json'
    headers:
      - key: 'Cache-Control'
        value: 'no-cache'
  - pattern: '/index.html'
    headers:
      - key: 'Cache-Control'
        value: 'no-cache'
  - pattern: '/_expo/static/**/*'
    headers:
      - key: 'Cache-Control'
        value: 'public, max-age=31536000, immutable'
`
)

// NovaStackWeb cria o app do Amplify Hosting que serve a PWA do frontend:
// plataforma WEB (estático), sem repositório conectado e sem role de serviço,
// porque o build roda no GitHub Actions e chega por deploy manual
// (create-deployment + start-deployment, no frontend.yml). Assim o Amplify não
// cobra minutos de build; fica só o free tier de hospedagem.
//
// Publica no SSM o id do app (lido pelo frontend.yml), a branch e a URL do site
// (lida pela stack api para liberar o CORS).
func NovaStackWeb(scope constructs.Construct, id string, env string, props *awscdk.StackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), props)

	app := awsamplify.NewCfnApp(stack, jsii.String("App"), &awsamplify.CfnAppProps{
		Name:        jsii.String(NomeStack(env, "web")),
		Description: jsii.String("PWA do Gestor de Gastos (" + env + "). Deploy manual pelo frontend.yml."),
		Platform:    jsii.String("WEB"),
		CustomRules: &[]interface{}{
			&awsamplify.CfnApp_CustomRuleProperty{
				Source: jsii.String(RegraSpaOrigem),
				Target: jsii.String(RegraSpaDestino),
				Status: jsii.String("200"),
			},
		},
		CustomHeaders: jsii.String(HeadersAmplify),
	})

	branch := awsamplify.NewCfnBranch(stack, jsii.String("Branch"), &awsamplify.CfnBranchProps{
		AppId:                    app.AttrAppId(),
		BranchName:               jsii.String(BranchAmplify),
		Description:              jsii.String("Ambiente " + env + " (deploy manual)"),
		Stage:                    jsii.String("DEVELOPMENT"),
		EnableAutoBuild:          jsii.Bool(false),
		EnablePullRequestPreview: jsii.Bool(false),
	})

	// URL padrão do Amplify: https://<branch>.<app-id>.amplifyapp.com, sem barra
	// final (é a origem do CORS). DefaultDomain é "<app-id>.amplifyapp.com".
	url := "https://" + BranchAmplify + "." + *app.AttrDefaultDomain()

	publicarParametro(stack, env, ParamAmplifyAppID, *app.AttrAppId(), "Id do app do Amplify Hosting do frontend")
	publicarParametro(stack, env, ParamAmplifyBranch, *branch.BranchName(), "Branch do app do Amplify que recebe os deploys")
	publicarParametro(stack, env, ParamFrontendURL, url, "URL (origem) do site no Amplify, liberada no CORS da API")

	return stack
}

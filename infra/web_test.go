package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
)

// Testes da stack web (tarefa 6.6): app do Amplify Hosting com deploy manual.

func TestWebAppDoAmplifySemRepositorio(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-web")

	tpl.ResourceCountIs(jsii.String("AWS::Amplify::App"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::Amplify::App"), map[string]interface{}{
		"Name":     "gestor-gastos-dev-web",
		"Platform": "WEB",
		// Deploy manual: sem repositório, token de acesso, role de serviço nem buildspec.
		"Repository":     assertions.Match_Absent(),
		"AccessToken":    assertions.Match_Absent(),
		"OauthToken":     assertions.Match_Absent(),
		"IamServiceRole": assertions.Match_Absent(),
		"BuildSpec":      assertions.Match_Absent(),
	})
	// Nada além do app, da branch e dos parâmetros.
	tpl.ResourceCountIs(jsii.String("AWS::IAM::Role"), jsii.Number(0))
	tpl.ResourceCountIs(jsii.String("AWS::Amplify::Domain"), jsii.Number(0))
}

func TestWebRewriteDeSpa(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-web")

	tpl.HasResourceProperties(jsii.String("AWS::Amplify::App"), map[string]interface{}{
		"CustomRules": []interface{}{
			map[string]interface{}{"Source": RegraSpaOrigem, "Target": "/index.html", "Status": "200"},
		},
	})

	// Formato do Amplify (regex entre </ e />): rotas sem extensão vão para o
	// index.html; as extensões abaixo são arquivos de verdade e não podem ser
	// reescritas, senão o sw.js, o manifest e os bundles viram HTML.
	if !strings.HasPrefix(RegraSpaOrigem, "</^[^.]+$|") || !strings.HasSuffix(RegraSpaOrigem, "/>") {
		t.Fatalf("regra fora do formato esperado: %s", RegraSpaOrigem)
	}
	inicio := strings.Index(RegraSpaOrigem, "(?!(") + len("(?!(")
	fim := strings.Index(RegraSpaOrigem, ")$)")
	extensoes := strings.Split(RegraSpaOrigem[inicio:fim], "|")
	for _, ext := range []string{"js", "css", "json", "webmanifest", "png", "ico", "svg", "ttf", "woff2", "map", "txt"} {
		if !slices.Contains(extensoes, ext) {
			t.Errorf("a extensão %q deveria ser servida como arquivo, não reescrita", ext)
		}
	}
	if slices.Contains(extensoes, "html") {
		t.Error("html não entra na lista: /index.html é o próprio destino")
	}
}

func TestWebHeadersDeCache(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-web")
	tpl.HasResourceProperties(jsii.String("AWS::Amplify::App"), map[string]interface{}{
		"CustomHeaders": HeadersAmplify,
	})
	// sw.js, manifest e index.html sem cache longo; bundles com hash imutáveis.
	blocos := strings.Split(HeadersAmplify, "- pattern: ")
	esperado := map[string]string{
		"'/sw.js'":             "'no-cache'",
		"'/manifest.json'":     "'no-cache'",
		"'/index.html'":        "'no-cache'",
		"'/_expo/static/**/*'": "'public, max-age=31536000, immutable'",
	}
	for padrao, valor := range esperado {
		achou := false
		for _, b := range blocos {
			if strings.HasPrefix(b, padrao) {
				achou = true
				if !strings.Contains(b, "value: "+valor) {
					t.Errorf("%s deveria ter Cache-Control %s", padrao, valor)
				}
			}
		}
		if !achou {
			t.Errorf("headers sem o padrão %s", padrao)
		}
	}
}

func TestWebBranchSemBuildAutomatico(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-web")
	app := logicalID(t, tpl, "AWS::Amplify::App")

	tpl.ResourceCountIs(jsii.String("AWS::Amplify::Branch"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::Amplify::Branch"), map[string]interface{}{
		"AppId":                    map[string]interface{}{"Fn::GetAtt": []interface{}{app, "AppId"}},
		"BranchName":               "main",
		"Stage":                    "DEVELOPMENT",
		"EnableAutoBuild":          false,
		"EnablePullRequestPreview": false,
	})
}

func TestWebParametrosNoSsm(t *testing.T) {
	tpl := templateDe(t, "gestor-gastos-dev-web")
	exigirParametros(t, tpl, ParamAmplifyAppID, ParamAmplifyBranch, ParamFrontendURL)
	app := logicalID(t, tpl, "AWS::Amplify::App")

	// O frontend.yml lê o id do app neste caminho (padrão do input amplify-app-id-ssm).
	tpl.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]interface{}{
		"Name":  "/gestor-gastos/dev/frontend/amplify-app-id",
		"Value": map[string]interface{}{"Fn::GetAtt": []interface{}{app, "AppId"}},
	})
	tpl.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]interface{}{
		"Name":  "/gestor-gastos/dev/frontend/amplify-branch",
		"Value": "main",
	})
	// Origem sem barra final, no formato que o CORS da api exige.
	tpl.HasResourceProperties(jsii.String("AWS::SSM::Parameter"), map[string]interface{}{
		"Name": "/gestor-gastos/dev/frontend/url",
		"Value": map[string]interface{}{"Fn::Join": []interface{}{"", []interface{}{
			"https://main.",
			map[string]interface{}{"Fn::GetAtt": []interface{}{app, "DefaultDomain"}},
		}}},
	})
}

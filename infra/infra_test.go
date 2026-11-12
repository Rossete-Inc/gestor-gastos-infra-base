package main

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
)

const emailTeste = "alertas@exemplo.com"

func novaStackCicdTeste(t *testing.T) assertions.Template {
	t.Helper()
	app := awscdk.NewApp(nil)
	stack := NovaStackCicd(app, StackCicd, &awscdk.StackProps{Env: &awscdk.Environment{Region: jsii.String(Regiao)}})
	return assertions.Template_FromStack(stack, nil)
}

func TestCicdCriaOProviderOidcDoGitHub(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	tpl.ResourceCountIs(jsii.String("AWS::IAM::OIDCProvider"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::IAM::OIDCProvider"), map[string]interface{}{
		"Url":          "https://token.actions.githubusercontent.com",
		"ClientIdList": []interface{}{"sts.amazonaws.com"},
	})
}

func TestCicdCriaUmaRolePorRepositorioEAmbiente(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	// 6 repositórios implantáveis x 1 ambiente + 5 roles de diff.
	tpl.ResourceCountIs(jsii.String("AWS::IAM::Role"), jsii.Number(11))

	for _, repo := range ReposImplantaveis {
		for _, env := range Ambientes {
			tpl.HasResourceProperties(jsii.String("AWS::IAM::Role"), map[string]interface{}{
				"RoleName": NomeRoleDeploy(repo, env),
			})
		}
	}
}

// trustEsperada é a trust policy exata, sem Match.objectLike no documento:
// qualquer condição a mais ou a menos quebra o teste.
func trustEsperada(sub string) *map[string]interface{} {
	return &map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []interface{}{
			map[string]interface{}{
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Federated": map[string]interface{}{
						"Fn::GetAtt": []interface{}{"GitHubOidcProvider", "Arn"},
					},
				},
				"Condition": map[string]interface{}{
					"StringEquals": map[string]interface{}{
						"token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
						"token.actions.githubusercontent.com:sub": sub,
					},
				},
			},
		},
	}
}

func TestCicdTrustDaRoleDeDev(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	tpl.HasResourceProperties(jsii.String("AWS::IAM::Role"), map[string]interface{}{
		"RoleName":                 "gh-deploy-gestor-gastos-users-service-dev",
		"AssumeRolePolicyDocument": assertions.Match_ObjectEquals(trustEsperada("repo:Rossete-Inc@339323229/gestor-gastos-users-service@1447342682:environment:dev")),
	})
}

func TestCicdNaoTemAmbienteProd(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	roles := tpl.FindResources(jsii.String("AWS::IAM::Role"), map[string]interface{}{
		"Properties": map[string]interface{}{"RoleName": "gh-deploy-gestor-gastos-bff-prod"},
	})
	if len(*roles) != 0 {
		t.Fatal("não deveria existir role de prod: o projeto tem um único ambiente")
	}
}

func TestCicdRoleDeDiffPorRepositorioComCdk(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	for _, repo := range ReposComCdk {
		tpl.HasResourceProperties(jsii.String("AWS::IAM::Role"), map[string]interface{}{
			"RoleName":                 NomeRoleDiff(repo),
			"AssumeRolePolicyDocument": assertions.Match_ObjectEquals(trustEsperada("repo:Rossete-Inc@339323229/" + repo + "@" + RepoIDs[repo] + ":pull_request")),
		})
	}

	frontend := tpl.FindResources(jsii.String("AWS::IAM::Role"), map[string]interface{}{
		"Properties": map[string]interface{}{"RoleName": NomeRoleDiff(RepoFrontend)},
	})
	if len(*frontend) != 0 {
		t.Fatal("o frontend não usa CDK e não deveria ter role de diff")
	}
}

func TestCicdRoleDeDiffSoAssumeALookupRole(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	tpl.HasResourceProperties(jsii.String("AWS::IAM::Policy"), map[string]interface{}{
		"PolicyDocument": map[string]interface{}{
			"Statement": assertions.Match_ArrayWith(&[]interface{}{
				map[string]interface{}{
					"Sid":    "AssumirLookupRoleDoBootstrap",
					"Action": []interface{}{"sts:AssumeRole", "sts:TagSession"},
					"Effect": "Allow",
					"Resource": map[string]interface{}{
						"Fn::Join": []interface{}{"", []interface{}{
							"arn:", map[string]interface{}{"Ref": "AWS::Partition"},
							":iam::", map[string]interface{}{"Ref": "AWS::AccountId"},
							":role/cdk-*-lookup-role-*",
						}},
					},
				},
			}),
		},
	})

	// Nenhuma política de role de diff pode alcançar as roles cdk-* de escrita.
	politicas := tpl.FindResources(jsii.String("AWS::IAM::Policy"), map[string]interface{}{})
	diffs := 0
	for nome, v := range *politicas {
		props := (*v)["Properties"].(map[string]interface{})
		ref := props["Roles"].([]interface{})[0].(map[string]interface{})["Ref"].(string)
		if len(ref) < len("ghdiff") || ref[:len("ghdiff")] != "ghdiff" {
			continue
		}
		diffs++
		for _, st := range props["PolicyDocument"].(map[string]interface{})["Statement"].([]interface{}) {
			if st.(map[string]interface{})["Sid"] == "AssumirRolesDoBootstrapCdk" {
				t.Errorf("a política %s da role de diff pode assumir todas as roles cdk-*", nome)
			}
		}
	}
	if diffs != len(ReposComCdk) {
		t.Fatalf("políticas de roles de diff: esperado %d, obtido %d", len(ReposComCdk), diffs)
	}
}

func TestCicdRolesSoAssumemAsRolesDoBootstrap(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	tpl.HasResourceProperties(jsii.String("AWS::IAM::Policy"), map[string]interface{}{
		"PolicyDocument": map[string]interface{}{
			"Statement": assertions.Match_ArrayWith(&[]interface{}{
				map[string]interface{}{
					"Sid":    "AssumirRolesDoBootstrapCdk",
					"Action": []interface{}{"sts:AssumeRole", "sts:TagSession"},
					"Effect": "Allow",
					"Resource": map[string]interface{}{
						"Fn::Join": []interface{}{"", []interface{}{
							"arn:", map[string]interface{}{"Ref": "AWS::Partition"},
							":iam::", map[string]interface{}{"Ref": "AWS::AccountId"},
							":role/cdk-*",
						}},
					},
				},
			}),
		},
	})
}

func TestCicdSoOFrontendTemPermissaoNoAmplify(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	comAmplify := tpl.FindResources(jsii.String("AWS::IAM::Policy"), map[string]interface{}{
		"Properties": map[string]interface{}{
			"PolicyDocument": map[string]interface{}{
				"Statement": assertions.Match_ArrayWith(&[]interface{}{
					assertions.Match_ObjectLike(&map[string]interface{}{
						"Sid": "DeployNoAmplifyHosting",
						"Action": assertions.Match_ArrayWith(&[]interface{}{
							"amplify:CreateDeployment", "amplify:StartDeployment",
						}),
					}),
				}),
			},
		},
	})

	// Uma política por role do frontend (só dev).
	if got := len(*comAmplify); got != 1 {
		t.Fatalf("políticas com permissão no Amplify: esperado 1 (frontend dev), obtido %d", got)
	}
	for _, v := range *comAmplify {
		roles := (*v)["Properties"].(map[string]interface{})["Roles"].([]interface{})
		ref := roles[0].(map[string]interface{})["Ref"].(string)
		if len(ref) < len("ghdeploygestorgastosfrontend") || ref[:len("ghdeploygestorgastosfrontend")] != "ghdeploygestorgastosfrontend" {
			t.Errorf("permissão no Amplify anexada a uma role que não é do frontend: %s", ref)
		}
	}
}

func TestCicdE2EDoFrontendNoCognito(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	comE2E := tpl.FindResources(jsii.String("AWS::IAM::Policy"), map[string]interface{}{
		"Properties": map[string]interface{}{
			"PolicyDocument": map[string]interface{}{
				"Statement": assertions.Match_ArrayWith(&[]interface{}{
					assertions.Match_ObjectLike(&map[string]interface{}{
						"Sid": "E2ENoCognito",
						"Action": assertions.Match_ArrayWith(&[]interface{}{
							"cognito-idp:AdminInitiateAuth", "cognito-idp:AdminAddUserToGroup",
							"cognito-idp:AdminConfirmSignUp", "cognito-idp:AdminDeleteUser",
						}),
					}),
				}),
			},
		},
	})

	// Só a role de deploy do frontend (dev) recebe as permissões do E2E.
	if got := len(*comE2E); got != 1 {
		t.Fatalf("políticas com E2ENoCognito: esperado 1 (frontend dev), obtido %d", got)
	}
	for _, v := range *comE2E {
		roles := (*v)["Properties"].(map[string]interface{})["Roles"].([]interface{})
		ref := roles[0].(map[string]interface{})["Ref"].(string)
		if len(ref) < len("ghdeploygestorgastosfrontend") || ref[:len("ghdeploygestorgastosfrontend")] != "ghdeploygestorgastosfrontend" {
			t.Errorf("permissão do E2E anexada a uma role que não é do frontend: %s", ref)
		}
	}
}

func TestCicdSemCloudFormationExports(t *testing.T) {
	tpl := novaStackCicdTeste(t)

	for nome, saida := range *tpl.ToJSON() {
		if nome != "Outputs" {
			continue
		}
		for id, o := range saida.(map[string]interface{}) {
			if _, tem := o.(map[string]interface{})["Export"]; tem {
				t.Errorf("saída %s usa CloudFormation export, proibido (usar SSM)", id)
			}
		}
	}
}

func TestBudgetDeCincoDolaresComDoisAlertas(t *testing.T) {
	t.Setenv(VarAmbienteEmailBudget, "")
	app := awscdk.NewApp(&awscdk.AppProps{
		Context: &map[string]interface{}{ChaveContextoEmailBudget: emailTeste},
	})
	stack := NovaStackBudget(app, StackBudget, nil)
	tpl := assertions.Template_FromStack(stack, nil)

	assinante := []interface{}{map[string]interface{}{"SubscriptionType": "EMAIL", "Address": emailTeste}}

	tpl.ResourceCountIs(jsii.String("AWS::Budgets::Budget"), jsii.Number(1))
	tpl.HasResourceProperties(jsii.String("AWS::Budgets::Budget"), map[string]interface{}{
		"Budget": map[string]interface{}{
			"BudgetType":  "COST",
			"TimeUnit":    "MONTHLY",
			"BudgetLimit": map[string]interface{}{"Amount": 5, "Unit": "USD"},
		},
		"NotificationsWithSubscribers": assertions.Match_ArrayEquals(&[]interface{}{
			map[string]interface{}{
				"Notification": map[string]interface{}{
					"NotificationType":   "ACTUAL",
					"ComparisonOperator": "GREATER_THAN",
					"Threshold":          80,
					"ThresholdType":      "PERCENTAGE",
				},
				"Subscribers": assinante,
			},
			map[string]interface{}{
				"Notification": map[string]interface{}{
					"NotificationType":   "FORECASTED",
					"ComparisonOperator": "GREATER_THAN",
					"Threshold":          100,
					"ThresholdType":      "PERCENTAGE",
				},
				"Subscribers": assinante,
			},
		}),
	})
}

func TestBudgetLeEmailDaVariavelDeAmbiente(t *testing.T) {
	t.Setenv(VarAmbienteEmailBudget, "outro@exemplo.com")
	app := awscdk.NewApp(nil)
	stack := NovaStackBudget(app, StackBudget, nil)
	tpl := assertions.Template_FromStack(stack, nil)

	tpl.HasResourceProperties(jsii.String("AWS::Budgets::Budget"), map[string]interface{}{
		"NotificationsWithSubscribers": assertions.Match_ArrayWith(&[]interface{}{
			assertions.Match_ObjectLike(&map[string]interface{}{
				"Subscribers": []interface{}{map[string]interface{}{"SubscriptionType": "EMAIL", "Address": "outro@exemplo.com"}},
			}),
		}),
	})
}

func TestBudgetSemEmailFalhaComMensagemClara(t *testing.T) {
	t.Setenv(VarAmbienteEmailBudget, "")
	app := awscdk.NewApp(nil)
	stack := NovaStackBudget(app, StackBudget, nil)

	assertions.Annotations_FromStack(stack).HasError(jsii.String("/"+StackBudget), MsgFaltaEmailBudget)
	assertions.Template_FromStack(stack, nil).ResourceCountIs(jsii.String("AWS::Budgets::Budget"), jsii.Number(0))
}

func TestAppTemTodasAsStacks(t *testing.T) {
	t.Setenv(VarAmbienteEmailBudget, emailTeste)
	app := awscdk.NewApp(nil)
	NovoApp(app, dirLambdaFake(t))

	for _, nome := range []string{StackCicd, StackBudget, "gestor-gastos-dev-events", "gestor-gastos-dev-auth", "gestor-gastos-dev-api"} {
		if app.Node().TryFindChild(jsii.String(nome)) == nil {
			t.Errorf("stack %s não encontrada no app", nome)
		}
	}
}

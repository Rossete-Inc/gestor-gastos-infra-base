package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// NovaStackCicd cria o OIDC provider do GitHub, uma role de deploy por
// repositório implantável e ambiente e uma role só leitura por repositório com
// CDK, usada pelo `cdk diff` nos PRs (ADR 0007).
//
// A stack é da conta, não de um ambiente: é implantada uma única vez, à mão,
// logo depois do `cdk bootstrap` (ver infra/README.md). Depois disso, o próprio
// infra-base a atualiza pela esteira usando a role gh-deploy-gestor-gastos-infra-base-<env>.
func NovaStackCicd(scope constructs.Construct, id string, props *awscdk.StackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), props)

	// L1 nativo do CloudFormation: dispensa a Lambda de custom resource do L2 antigo.
	// A AWS valida o certificado do GitHub pela própria CA; os thumbprints ficam por
	// compatibilidade.
	provider := awsiam.NewCfnOIDCProvider(stack, jsii.String("GitHubOidcProvider"), &awsiam.CfnOIDCProviderProps{
		Url:          jsii.String("https://" + GitHubOIDCHost),
		ClientIdList: jsii.Strings(GitHubOIDCAudience),
		ThumbprintList: jsii.Strings(
			"6938fd4d98bab03faadb97b34396831e3780aea1",
			"1c58a3a8518e8759bf075b76b750d4f2df264fcd",
		),
	})

	for _, repo := range ReposImplantaveis {
		for _, env := range Ambientes {
			novaRoleDeploy(stack, provider, repo, env)
		}
	}
	for _, repo := range ReposComCdk {
		novaRoleDiff(stack, provider, repo)
	}

	return stack
}

func novaRoleDeploy(stack awscdk.Stack, provider awsiam.CfnOIDCProvider, repo, env string) awsiam.Role {
	nome := NomeRoleDeploy(repo, env)

	role := awsiam.NewRole(stack, jsii.String(nome), &awsiam.RoleProps{
		RoleName:           jsii.String(nome),
		AssumedBy:          principalGitHub(provider, SubjectOIDC(repo, env)),
		Description:        jsii.String("Deploy do " + repo + " em " + env + " pelo GitHub Actions (OIDC)"),
		MaxSessionDuration: awscdk.Duration_Hours(jsii.Number(1)),
	})

	// O `cdk deploy` só precisa assumir as roles criadas pelo `cdk bootstrap`
	// (deploy, file-publishing, image-publishing e lookup). Quem cria os recursos
	// é a cloudformation-execution-role, assumida pelo próprio CloudFormation.
	role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:     jsii.String("AssumirRolesDoBootstrapCdk"),
		Actions: jsii.Strings("sts:AssumeRole", "sts:TagSession"),
		Resources: jsii.Strings(
			"arn:" + *awscdk.Aws_PARTITION() + ":iam::" + *awscdk.Aws_ACCOUNT_ID() + ":role/cdk-*",
		),
	}))

	// Leitura dos parâmetros do próprio ambiente, para a esteira checar os
	// pré-requisitos e falhar com mensagem clara antes do deploy.
	role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:     jsii.String("LerParametrosDoAmbiente"),
		Actions: jsii.Strings("ssm:GetParameter", "ssm:GetParameters", "ssm:GetParametersByPath"),
		Resources: jsii.Strings(
			"arn:"+*awscdk.Aws_PARTITION()+":ssm:"+*awscdk.Aws_REGION()+":"+*awscdk.Aws_ACCOUNT_ID()+":parameter"+PrefixoSSM(env),
			"arn:"+*awscdk.Aws_PARTITION()+":ssm:"+*awscdk.Aws_REGION()+":"+*awscdk.Aws_ACCOUNT_ID()+":parameter"+PrefixoSSM(env)+"/*",
		),
	}))

	// O frontend não usa CDK para publicar o site: ele envia o build do
	// `expo export -p web` direto ao Amplify Hosting (`aws amplify create-deployment`
	// seguido de `start-deployment`). Por isso precisa de permissões próprias no
	// Amplify. O app id ainda não existe (é criado na 6.6), então o escopo é
	// "apps/*" da conta e região, não um app específico.
	if repo == RepoFrontend {
		role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
			Sid: jsii.String("DeployNoAmplifyHosting"),
			Actions: jsii.Strings(
				"amplify:CreateDeployment",
				"amplify:StartDeployment",
				"amplify:GetApp",
				"amplify:GetBranch",
				"amplify:GetJob",
				"amplify:ListJobs",
				"amplify:StopJob",
			),
			Resources: jsii.Strings(
				"arn:" + *awscdk.Aws_PARTITION() + ":amplify:" + *awscdk.Aws_REGION() + ":" + *awscdk.Aws_ACCOUNT_ID() + ":apps/*",
			),
		}))
	}

	// Saída informativa (sem exportName: nunca usamos CloudFormation exports).
	awscdk.NewCfnOutput(stack, jsii.String(nome+"-arn"), &awscdk.CfnOutputProps{
		Value:       role.RoleArn(),
		Description: jsii.String("ARN da role " + nome),
	})

	return role
}

// principalGitHub aceita só tokens do GitHub Actions com a audience do
// configure-aws-credentials e a claim `sub` exata informada.
func principalGitHub(provider awsiam.CfnOIDCProvider, sub string) awsiam.IPrincipal {
	return awsiam.NewWebIdentityPrincipal(provider.AttrArn(), &map[string]interface{}{
		"StringEquals": map[string]interface{}{
			GitHubOIDCHost + ":aud": GitHubOIDCAudience,
			GitHubOIDCHost + ":sub": sub,
		},
	})
}

// novaRoleDiff cria a role que o job de `cdk diff` assume em PRs. Ela não passa
// por GitHub Environment (a trust aceita o `sub` de pull_request, de qualquer
// branch do próprio repositório) e, por isso, não tem nenhuma permissão de escrita:
// só assume a lookup role do bootstrap, que é ReadOnlyAccess, e lê templates
// e parâmetros. O diff roda com `--method=template`, sem change set.
func novaRoleDiff(stack awscdk.Stack, provider awsiam.CfnOIDCProvider, repo string) awsiam.Role {
	nome := NomeRoleDiff(repo)
	conta := *awscdk.Aws_ACCOUNT_ID()
	particao := *awscdk.Aws_PARTITION()
	regiao := *awscdk.Aws_REGION()

	role := awsiam.NewRole(stack, jsii.String(nome), &awsiam.RoleProps{
		RoleName:           jsii.String(nome),
		AssumedBy:          principalGitHub(provider, SubjectOIDCPullRequest(repo)),
		Description:        jsii.String("cdk diff (so leitura) do " + repo + " em PRs pelo GitHub Actions (OIDC)"),
		MaxSessionDuration: awscdk.Duration_Hours(jsii.Number(1)),
	})

	role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("AssumirLookupRoleDoBootstrap"),
		Actions:   jsii.Strings("sts:AssumeRole", "sts:TagSession"),
		Resources: jsii.Strings("arn:" + particao + ":iam::" + conta + ":role/cdk-*-lookup-role-*"),
	}))

	role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:       jsii.String("LerStacks"),
		Actions:   jsii.Strings("cloudformation:DescribeStacks", "cloudformation:GetTemplate"),
		Resources: jsii.Strings("arn:" + particao + ":cloudformation:" + regiao + ":" + conta + ":stack/" + Projeto + "-*/*"),
	}))

	role.AddToPolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:     jsii.String("LerParametros"),
		Actions: jsii.Strings("ssm:GetParameter", "ssm:GetParameters", "ssm:GetParametersByPath"),
		Resources: jsii.Strings(
			"arn:"+particao+":ssm:"+regiao+":"+conta+":parameter/"+Projeto+"/*",
			"arn:"+particao+":ssm:"+regiao+":"+conta+":parameter/cdk-bootstrap/*",
		),
	}))

	awscdk.NewCfnOutput(stack, jsii.String(nome+"-arn"), &awscdk.CfnOutputProps{
		Value:       role.RoleArn(),
		Description: jsii.String("ARN da role " + nome),
	})

	return role
}

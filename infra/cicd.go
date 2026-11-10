package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// NovaStackCicd cria o OIDC provider do GitHub e uma role de deploy por
// repositório implantável e ambiente (ADR 0007).
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

	return stack
}

func novaRoleDeploy(stack awscdk.Stack, provider awsiam.CfnOIDCProvider, repo, env string) awsiam.Role {
	nome := NomeRoleDeploy(repo, env)

	principal := awsiam.NewWebIdentityPrincipal(provider.AttrArn(), &map[string]interface{}{
		"StringEquals": map[string]interface{}{
			GitHubOIDCHost + ":aud": GitHubOIDCAudience,
			GitHubOIDCHost + ":sub": SubjectOIDC(repo, env),
		},
	})

	role := awsiam.NewRole(stack, jsii.String(nome), &awsiam.RoleProps{
		RoleName:           jsii.String(nome),
		AssumedBy:          principal,
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

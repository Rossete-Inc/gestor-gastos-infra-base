package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// publicarParametro grava um valor no SSM em /gestor-gastos/<env>/<param>.
// É assim que os outros repositórios descobrem os recursos do infra-base, sem
// CloudFormation exports. Parâmetros Standard não têm custo.
func publicarParametro(scope constructs.Construct, env, param, valor, descricao string) awsssm.StringParameter {
	return awsssm.NewStringParameter(scope, jsii.String("Param-"+param), &awsssm.StringParameterProps{
		ParameterName: jsii.String(CaminhoSSM(env, param)),
		StringValue:   jsii.String(valor),
		Description:   jsii.String(descricao),
		Tier:          awsssm.ParameterTier_STANDARD,
	})
}

// lerParametro lê, no momento do deploy, um parâmetro publicado por outra stack.
// Vira um parâmetro do CloudFormation do tipo AWS::SSM::Parameter::Value<String>,
// então a stack que publica precisa ser implantada antes (ver AddStackDependency em main.go).
func lerParametro(scope constructs.Construct, env, param string) string {
	return *awsssm.StringParameter_ValueForStringParameter(scope, jsii.String(CaminhoSSM(env, param)), nil)
}

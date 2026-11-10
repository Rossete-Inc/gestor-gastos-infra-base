package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudwatch"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// NovaStackEvents cria o barramento do EventBridge do ambiente e a DLQ que os
// consumidores (users-service e os outros serviços) usam nos targets das suas
// regras. As regras ficam no CDK de cada serviço (tarefa 6.5).
//
// Sem arquivo de eventos nem replay, por custo.
func NovaStackEvents(scope constructs.Construct, id string, env string, props *awscdk.StackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), props)

	nomeBarramento := NomeBarramento(env)
	barramento := awsevents.NewEventBus(stack, jsii.String("Barramento"), &awsevents.EventBusProps{
		EventBusName: jsii.String(nomeBarramento),
		Description:  jsii.String("Eventos assíncronos do Gestor de Gastos (" + env + ")"),
	})

	dlq := awssqs.NewQueue(stack, jsii.String("Dlq"), &awssqs.QueueProps{
		QueueName:       jsii.String(NomeStack(env, "events-dlq")),
		RetentionPeriod: awscdk.Duration_Days(jsii.Number(14)),
		Encryption:      awssqs.QueueEncryption_SQS_MANAGED,
		EnforceSSL:      jsii.Bool(true),
	})

	// As regras são criadas depois, em outras stacks, então a permissão não
	// pode apontar para uma regra específica: vale qualquer regra do barramento
	// do ambiente (arn:...:rule/<barramento>/<regra>).
	dlq.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:        jsii.String("RegrasDoBarramentoEnviamParaADlq"),
		Principals: &[]awsiam.IPrincipal{awsiam.NewServicePrincipal(jsii.String("events.amazonaws.com"), nil)},
		Actions:    jsii.Strings("sqs:SendMessage"),
		Resources:  jsii.Strings(*dlq.QueueArn()),
		Conditions: &map[string]interface{}{
			"ArnLike": map[string]interface{}{
				"aws:SourceArn": "arn:" + *awscdk.Aws_PARTITION() + ":events:" + *awscdk.Aws_REGION() + ":" +
					*awscdk.Aws_ACCOUNT_ID() + ":rule/" + nomeBarramento + "/*",
			},
		},
	}))

	// Alarme sem ação (sem SNS, para não pôr e-mail no código público): fica
	// vermelho no console do CloudWatch. Está dentro dos 10 alarmes gratuitos.
	awscloudwatch.NewAlarm(stack, jsii.String("DlqComMensagens"), &awscloudwatch.AlarmProps{
		AlarmName:          jsii.String(NomeStack(env, "events-dlq-com-mensagens")),
		AlarmDescription:   jsii.String("Há eventos que nenhum consumidor conseguiu processar na DLQ do barramento " + nomeBarramento),
		Metric:             dlq.MetricApproximateNumberOfMessagesVisible(&awscloudwatch.MetricOptions{Period: awscdk.Duration_Minutes(jsii.Number(5))}),
		Threshold:          jsii.Number(1),
		EvaluationPeriods:  jsii.Number(1),
		ComparisonOperator: awscloudwatch.ComparisonOperator_GREATER_THAN_OR_EQUAL_TO_THRESHOLD,
		TreatMissingData:   awscloudwatch.TreatMissingData_NOT_BREACHING,
	})

	publicarParametro(stack, env, ParamBusName, *barramento.EventBusName(), "Nome do barramento do EventBridge")
	publicarParametro(stack, env, ParamBusArn, *barramento.EventBusArn(), "ARN do barramento do EventBridge")
	publicarParametro(stack, env, ParamDlqArn, *dlq.QueueArn(), "ARN da DLQ dos targets das regras do barramento")
	publicarParametro(stack, env, ParamDlqURL, *dlq.QueueUrl(), "URL da DLQ dos targets das regras do barramento")

	return stack
}

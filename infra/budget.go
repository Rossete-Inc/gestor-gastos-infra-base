package main

import (
	"os"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsbudgets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

const (
	// LimiteBudgetUSD é o teto mensal da conta (spec controle-custos).
	LimiteBudgetUSD = 5

	// ChaveContextoEmailBudget é a chave de contexto do CDK com o e-mail dos alertas
	// (`cdk synth -c budgetEmail=...`).
	ChaveContextoEmailBudget = "budgetEmail"

	// VarAmbienteEmailBudget é a alternativa ao contexto, útil na esteira.
	VarAmbienteEmailBudget = "BUDGET_EMAIL"

	// MsgFaltaEmailBudget é o erro mostrado quando o e-mail não foi informado.
	MsgFaltaEmailBudget = "stack budget: informe o e-mail dos alertas com -c " + ChaveContextoEmailBudget +
		"=voce@exemplo.com ou com a variável de ambiente " + VarAmbienteEmailBudget +
		". O e-mail não fica no código porque o repositório é público."
)

// EmailBudget lê o e-mail dos alertas do contexto do CDK e, se não houver,
// da variável de ambiente BUDGET_EMAIL. Devolve "" quando nenhum foi informado.
func EmailBudget(scope constructs.Construct) string {
	if v, ok := scope.Node().TryGetContext(jsii.String(ChaveContextoEmailBudget)).(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(os.Getenv(VarAmbienteEmailBudget))
}

// NovaStackBudget cria o orçamento mensal de US$ 5 da conta, com alerta por
// e-mail a 80% do custo real e a 100% do custo previsto.
//
// Sem e-mail, a stack não cria o orçamento e registra um erro de synth. O CDK CLI
// só reprova o synth/deploy das stacks selecionadas, então `cdk deploy
// gestor-gastos-cicd` continua funcionando sem o e-mail, e `cdk deploy
// gestor-gastos-budget` falha com MsgFaltaEmailBudget.
func NovaStackBudget(scope constructs.Construct, id string, props *awscdk.StackProps) awscdk.Stack {
	stack := awscdk.NewStack(scope, jsii.String(id), props)

	email := EmailBudget(stack)
	if email == "" || !strings.Contains(email, "@") {
		awscdk.Annotations_Of(stack).AddError(jsii.String(MsgFaltaEmailBudget))
		return stack
	}

	assinantes := &[]interface{}{
		&awsbudgets.CfnBudget_SubscriberProperty{
			SubscriptionType: jsii.String("EMAIL"),
			Address:          jsii.String(email),
		},
	}

	awsbudgets.NewCfnBudget(stack, jsii.String("BudgetMensal"), &awsbudgets.CfnBudgetProps{
		Budget: &awsbudgets.CfnBudget_BudgetDataProperty{
			BudgetName: jsii.String(Projeto + "-mensal"),
			BudgetType: jsii.String("COST"),
			TimeUnit:   jsii.String("MONTHLY"),
			BudgetLimit: &awsbudgets.CfnBudget_SpendProperty{
				Amount: jsii.Number(LimiteBudgetUSD),
				Unit:   jsii.String("USD"),
			},
		},
		NotificationsWithSubscribers: &[]interface{}{
			&awsbudgets.CfnBudget_NotificationWithSubscribersProperty{
				Notification: &awsbudgets.CfnBudget_NotificationProperty{
					NotificationType:   jsii.String("ACTUAL"),
					ComparisonOperator: jsii.String("GREATER_THAN"),
					Threshold:          jsii.Number(80),
					ThresholdType:      jsii.String("PERCENTAGE"),
				},
				Subscribers: assinantes,
			},
			&awsbudgets.CfnBudget_NotificationWithSubscribersProperty{
				Notification: &awsbudgets.CfnBudget_NotificationProperty{
					NotificationType:   jsii.String("FORECASTED"),
					ComparisonOperator: jsii.String("GREATER_THAN"),
					Threshold:          jsii.Number(100),
					ThresholdType:      jsii.String("PERCENTAGE"),
				},
				Subscribers: assinantes,
			},
		},
	})

	return stack
}

// Package postconfirmation implementa o trigger Post Confirmation do Cognito.
//
// Quando uma pessoa confirma o cadastro, o trigger:
//  1. adiciona o usuário ao grupo `cliente` (AdminAddUserToGroup);
//  2. publica `UserConfirmed` no barramento do EventBridge, consumido pelo
//     users-service, que cria o perfil (design.md, "Post Confirmation").
//
// Localmente o trigger não existe: o seed coloca os usuários de teste nos
// grupos e o `/me` do BFF cria o perfil sob demanda.
package postconfirmation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Rossete-Inc/gestor-gastos-go-commons/events"
	lambdaevents "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

const (
	// GrupoCliente é o papel que toda conta recebe ao ser confirmada.
	GrupoCliente = "cliente"

	// Source é o `source` dos eventos publicados pelo trigger. O users-service
	// filtra a regra do EventBridge por ele.
	Source = "gestor-gastos.auth"

	// GatilhoCadastro é o triggerSource da confirmação de cadastro. O mesmo
	// trigger também dispara em PostConfirmation_ConfirmForgotPassword (troca de
	// senha esquecida), que não deve repetir grupo nem evento.
	GatilhoCadastro = "PostConfirmation_ConfirmSignUp"
)

// CognitoAPI é o subconjunto do client Cognito usado aqui;
// *cognitoidentityprovider.Client o satisfaz, e os testes usam um fake.
type CognitoAPI interface {
	AdminAddUserToGroup(ctx context.Context, params *cip.AdminAddUserToGroupInput, optFns ...func(*cip.Options)) (*cip.AdminAddUserToGroupOutput, error)
}

// Evento é o payload que o Cognito entrega ao trigger.
type Evento = lambdaevents.CognitoEventUserPoolsPostConfirmation

// Handler trata o evento do Cognito.
type Handler struct {
	Cognito CognitoAPI
	Eventos events.Publisher
	Log     *slog.Logger
}

// Handle devolve o evento sem alteração, como o Cognito exige.
//
// Falha ao adicionar o grupo devolve erro: sem o papel, a pessoa não acessa
// nada, e o erro aparece no log e na resposta do ConfirmSignUp. Falha ao
// publicar o evento só gera log de erro: o perfil também é criado sob demanda
// pelo `/me` do BFF (design.md, riscos), então a confirmação não precisa falhar.
func (h *Handler) Handle(ctx context.Context, ev Evento) (Evento, error) {
	if ev.TriggerSource != GatilhoCadastro {
		h.Log.InfoContext(ctx, "gatilho ignorado", "triggerSource", ev.TriggerSource)
		return ev, nil
	}

	userID := ev.Request.UserAttributes["sub"]
	if userID == "" || ev.UserName == "" || ev.UserPoolID == "" {
		return ev, errors.New("post confirmation: evento sem sub, userName ou userPoolId")
	}

	_, err := h.Cognito.AdminAddUserToGroup(ctx, &cip.AdminAddUserToGroupInput{
		UserPoolId: aws.String(ev.UserPoolID),
		Username:   aws.String(ev.UserName),
		GroupName:  aws.String(GrupoCliente),
	})
	if err != nil {
		return ev, fmt.Errorf("adicionar %s ao grupo %s: %w", userID, GrupoCliente, err)
	}

	detalhe := events.UserConfirmedDetail{
		UserID: userID,
		Email:  ev.Request.UserAttributes["email"],
		Nome:   ev.Request.UserAttributes["name"],
	}
	if err := h.Eventos.Publish(ctx, events.UserConfirmed, detalhe); err != nil {
		h.Log.ErrorContext(ctx, "falha ao publicar UserConfirmed; o perfil será criado pelo /me",
			"userId", userID, "erro", err.Error())
		return ev, nil
	}

	h.Log.InfoContext(ctx, "cadastro confirmado", "userId", userID, "grupo", GrupoCliente)
	return ev, nil
}

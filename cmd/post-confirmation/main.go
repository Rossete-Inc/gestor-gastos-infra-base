// Command post-confirmation é a Lambda do trigger Post Confirmation do Cognito
// (stack gestor-gastos-<env>-auth). A lógica fica em internal/postconfirmation.
//
// Variáveis de ambiente (definidas pelo CDK):
//
//	RUNTIME=aws          o trigger só existe na AWS
//	ENV=dev              ambiente
//	EVENT_BUS_NAME=...   barramento onde UserConfirmed é publicado
//	LOG_LEVEL=info       opcional
//
// Build (o mesmo do go-service.yml):
//
//	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -trimpath -ldflags="-s -w" \
//	  -o dist/post-confirmation/bootstrap ./cmd/post-confirmation
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/Rossete-Inc/gestor-gastos-go-commons/config"
	"github.com/Rossete-Inc/gestor-gastos-go-commons/events"
	"github.com/Rossete-Inc/gestor-gastos-go-commons/logging"
	"github.com/Rossete-Inc/gestor-gastos-infra-base/internal/postconfirmation"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

const servico = "post-confirmation"

func main() {
	h, err := novoHandler(context.Background(), os.LookupEnv)
	if err != nil {
		slog.Error("configuração inválida", "erro", err.Error())
		os.Exit(1)
	}
	lambda.Start(h.Handle)
}

func novoHandler(ctx context.Context, lookup config.LookupFunc) (*postconfirmation.Handler, error) {
	base, err := config.LoadFrom(lookup, servico)
	if err != nil {
		return nil, err
	}
	if base.IsLocal() {
		return nil, errors.New("o trigger Post Confirmation só roda com RUNTIME=aws")
	}
	r := config.NewReader(lookup)
	bus := r.Require("EVENT_BUS_NAME")
	if err := r.Err(); err != nil {
		return nil, err
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(base.Region))
	if err != nil {
		return nil, fmt.Errorf("carregar configuração da AWS: %w", err)
	}
	pub, err := events.NewPublisher(events.Options{
		Runtime: base.Runtime,
		Source:  postconfirmation.Source,
		BusName: bus,
		API:     eventbridge.NewFromConfig(cfg),
	})
	if err != nil {
		return nil, err
	}

	return &postconfirmation.Handler{
		Cognito: cip.NewFromConfig(cfg),
		Eventos: pub,
		Log:     logging.New(logging.Options{Service: servico, Env: base.Env, Level: base.LogLevel}),
	}, nil
}

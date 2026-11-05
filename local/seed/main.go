// Comando seed prepara o ambiente local do Gestor de Gastos.
//
//	seed cognito  cria o User Pool, os grupos e o app client no cognito-local
//	seed data     garante o Cognito e cria as tabelas, as categorias padrão
//	              e os usuários de teste (all é sinônimo)
//
// Todas as etapas são idempotentes. O comando só aceita endpoints locais,
// para nunca tocar a AWS por engano.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"
)

const region = "us-east-1"

type config struct {
	dynamoEndpoint  string
	cognitoEndpoint string
	envFile         string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "uso: seed cognito|data|all")
		os.Exit(2)
	}

	cfg := config{
		dynamoEndpoint:  envOr("DYNAMODB_ENDPOINT", "http://localhost:8000"),
		cognitoEndpoint: envOr("COGNITO_ENDPOINT", "http://localhost:9229"),
		envFile:         envOr("SEED_ENV_FILE", ".env.cognito"),
	}
	for _, e := range []string{cfg.dynamoEndpoint, cfg.cognitoEndpoint} {
		if err := exigirLocal(e); err != nil {
			fatal(err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var err error
	switch os.Args[1] {
	case "cognito":
		_, err = seedCognito(ctx, cfg)
	case "data", "all":
		// data depende do pool, então sempre garante o Cognito antes.
		err = seedData(ctx, cfg)
	default:
		err = fmt.Errorf("etapa desconhecida %q (use cognito, data ou all)", os.Args[1])
	}
	if err != nil {
		fatal(err)
	}
}

func seedData(ctx context.Context, cfg config) error {
	pool, err := seedCognito(ctx, cfg)
	if err != nil {
		return err
	}
	if err := seedTabelas(ctx, cfg); err != nil {
		return err
	}
	if err := seedCategorias(ctx, cfg); err != nil {
		return err
	}
	return seedUsuarios(ctx, cfg, pool)
}

// exigirLocal recusa qualquer endpoint que não seja a própria máquina ou um
// container do compose.
func exigirLocal(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("endpoint inválido %q: %w", endpoint, err)
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1", "dynamodb", "cognito", "host.docker.internal":
		return nil
	}
	return fmt.Errorf("endpoint %q não é local; o seed só roda contra o ambiente local", endpoint)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func logf(format string, a ...any) { fmt.Printf("  "+format+"\n", a...) }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}

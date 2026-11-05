package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// categoriasPadrao segue a spec `categorias`. Os IDs são estáveis: o
// categories-service usa os mesmos no seed da nuvem (tarefa 7.2).
// As cores devem bater com os tokens category/* do Figma.
var categoriasPadrao = []struct{ id, grupo, nome, cor string }{
	{"alimentacao", "CARTAO", "Alimentação", "#F97316"},
	{"transporte", "CARTAO", "Transporte", "#3B82F6"},
	{"lazer", "CARTAO", "Lazer", "#A855F7"},
	{"saude", "CARTAO", "Saúde", "#EF4444"},
	{"compras", "CARTAO", "Compras", "#EC4899"},
	{"outros", "CARTAO", "Outros", "#64748B"},
	{"aluguel", "FIXO", "Aluguel", "#0EA5E9"},
	{"luz", "FIXO", "Luz", "#EAB308"},
	{"agua", "FIXO", "Água", "#06B6D4"},
	{"internet", "FIXO", "Internet", "#6366F1"},
	{"telefone", "FIXO", "Telefone", "#14B8A6"},
	{"condominio", "FIXO", "Condomínio", "#84CC16"},
}

// SenhaTeste é a senha dos usuários de teste. Só existe no ambiente local.
const SenhaTeste = "Local1234"

var usuariosTeste = []struct{ email, nome, grupo string }{
	{"admin@local.test", "Admin Local", "admin"},
	{"cliente@local.test", "Cliente Local", "cliente"},
}

func seedUsuarios(ctx context.Context, cfg config, pool poolInfo) error {
	fmt.Println("usuários de teste:")
	c := cognitoClient(cfg)
	for _, u := range usuariosTeste {
		_, err := c.AdminGetUser(ctx, &cip.AdminGetUserInput{UserPoolId: &pool.poolID, Username: aws.String(u.email)})
		var notFound *types.UserNotFoundException
		switch {
		case err == nil:
			logf("%s já existe", u.email)
		case errors.As(err, &notFound):
			if err := criarUsuario(ctx, c, pool.poolID, u.email, u.nome); err != nil {
				return err
			}
			logf("%s criado", u.email)
		default:
			return fmt.Errorf("consultar %s: %w", u.email, err)
		}

		// Reaplicar grupo e senha deixa o usuário sempre no estado esperado.
		if _, err := c.AdminSetUserPassword(ctx, &cip.AdminSetUserPasswordInput{
			UserPoolId: &pool.poolID, Username: aws.String(u.email),
			Password: aws.String(SenhaTeste), Permanent: true,
		}); err != nil {
			return fmt.Errorf("definir senha de %s: %w", u.email, err)
		}
		if _, err := c.AdminAddUserToGroup(ctx, &cip.AdminAddUserToGroupInput{
			UserPoolId: &pool.poolID, Username: aws.String(u.email), GroupName: aws.String(u.grupo),
		}); err != nil {
			return fmt.Errorf("adicionar %s ao grupo %s: %w", u.email, u.grupo, err)
		}
		logf("%s no grupo %s, senha %s", u.email, u.grupo, SenhaTeste)
	}
	return nil
}

func criarUsuario(ctx context.Context, c *cip.Client, poolID, email, nome string) error {
	_, err := c.AdminCreateUser(ctx, &cip.AdminCreateUserInput{
		UserPoolId:        &poolID,
		Username:          aws.String(email),
		TemporaryPassword: aws.String(SenhaTeste),
		MessageAction:     types.MessageActionTypeSuppress,
		UserAttributes: []types.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
			{Name: aws.String("email_verified"), Value: aws.String("true")},
			{Name: aws.String("name"), Value: aws.String(nome)},
		},
	})
	if err != nil {
		return fmt.Errorf("criar %s: %w", email, err)
	}
	return nil
}

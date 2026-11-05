package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

const (
	poolName   = "gestor-gastos-local"
	clientName = "gestor-gastos-frontend-local"
)

var grupos = []struct{ nome, descricao string }{
	{"admin", "Administradores: gerenciam usuários"},
	{"cliente", "Clientes: lançam e consultam os próprios gastos"},
}

type poolInfo struct {
	poolID   string
	clientID string
}

func cognitoClient(cfg config) *cip.Client {
	return cip.New(cip.Options{
		Region:       region,
		BaseEndpoint: aws.String(cfg.cognitoEndpoint),
		Credentials:  credentials.NewStaticCredentialsProvider("local", "local", ""),
	})
}

// seedCognito garante o User Pool, os grupos e o app client, e grava os IDs
// em cfg.envFile para o BFF e o frontend.
func seedCognito(ctx context.Context, cfg config) (poolInfo, error) {
	fmt.Println("cognito-local:")
	c := cognitoClient(cfg)

	poolID, err := garantirPool(ctx, c)
	if err != nil {
		return poolInfo{}, err
	}
	// O cognito-local não devolve GroupExistsException, então consulta antes.
	existentes, err := c.ListGroups(ctx, &cip.ListGroupsInput{UserPoolId: &poolID})
	if err != nil {
		return poolInfo{}, fmt.Errorf("listar grupos: %w", err)
	}
	jaExiste := map[string]bool{}
	for _, g := range existentes.Groups {
		jaExiste[aws.ToString(g.GroupName)] = true
	}
	for _, g := range grupos {
		if jaExiste[g.nome] {
			logf("grupo %s já existe", g.nome)
			continue
		}
		_, err := c.CreateGroup(ctx, &cip.CreateGroupInput{
			UserPoolId:  &poolID,
			GroupName:   aws.String(g.nome),
			Description: aws.String(g.descricao),
		})
		var exists *types.GroupExistsException
		if err != nil && !errors.As(err, &exists) {
			return poolInfo{}, fmt.Errorf("criar grupo %s: %w", g.nome, err)
		}
		logf("grupo %s criado", g.nome)
	}
	clientID, err := garantirClient(ctx, c, poolID)
	if err != nil {
		return poolInfo{}, err
	}

	info := poolInfo{poolID: poolID, clientID: clientID}
	if err := gravarEnv(cfg, info); err != nil {
		return poolInfo{}, err
	}
	return info, nil
}

func garantirPool(ctx context.Context, c *cip.Client) (string, error) {
	out, err := c.ListUserPools(ctx, &cip.ListUserPoolsInput{MaxResults: aws.Int32(60)})
	if err != nil {
		return "", fmt.Errorf("listar user pools: %w", err)
	}
	for _, p := range out.UserPools {
		if aws.ToString(p.Name) == poolName {
			logf("user pool %s já existe (%s)", poolName, aws.ToString(p.Id))
			return aws.ToString(p.Id), nil
		}
	}

	created, err := c.CreateUserPool(ctx, &cip.CreateUserPoolInput{
		PoolName:               aws.String(poolName),
		UsernameAttributes:     []types.UsernameAttributeType{types.UsernameAttributeTypeEmail},
		AutoVerifiedAttributes: []types.VerifiedAttributeType{types.VerifiedAttributeTypeEmail},
		Policies: &types.UserPoolPolicyType{PasswordPolicy: &types.PasswordPolicyType{
			MinimumLength:    aws.Int32(8),
			RequireUppercase: true,
			RequireLowercase: true,
			RequireNumbers:   true,
			RequireSymbols:   false,
		}},
		Schema: []types.SchemaAttributeType{{
			Name:              aws.String("name"),
			AttributeDataType: types.AttributeDataTypeString,
			Required:          aws.Bool(true),
			Mutable:           aws.Bool(true),
		}},
	})
	if err != nil {
		return "", fmt.Errorf("criar user pool: %w", err)
	}
	id := aws.ToString(created.UserPool.Id)
	logf("user pool %s criado (%s)", poolName, id)
	return id, nil
}

func garantirClient(ctx context.Context, c *cip.Client, poolID string) (string, error) {
	out, err := c.ListUserPoolClients(ctx, &cip.ListUserPoolClientsInput{UserPoolId: &poolID, MaxResults: aws.Int32(60)})
	if err != nil {
		return "", fmt.Errorf("listar app clients: %w", err)
	}
	for _, cl := range out.UserPoolClients {
		if aws.ToString(cl.ClientName) == clientName {
			logf("app client %s já existe (%s)", clientName, aws.ToString(cl.ClientId))
			return aws.ToString(cl.ClientId), nil
		}
	}

	created, err := c.CreateUserPoolClient(ctx, &cip.CreateUserPoolClientInput{
		UserPoolId:     &poolID,
		ClientName:     aws.String(clientName),
		GenerateSecret: false,
		ExplicitAuthFlows: []types.ExplicitAuthFlowsType{
			types.ExplicitAuthFlowsTypeAllowUserPasswordAuth,
			types.ExplicitAuthFlowsTypeAllowUserSrpAuth,
			types.ExplicitAuthFlowsTypeAllowRefreshTokenAuth,
		},
		AccessTokenValidity: aws.Int32(15),
		IdTokenValidity:     aws.Int32(15),
		TokenValidityUnits: &types.TokenValidityUnitsType{
			AccessToken: types.TimeUnitsTypeMinutes,
			IdToken:     types.TimeUnitsTypeMinutes,
		},
	})
	if err != nil {
		return "", fmt.Errorf("criar app client: %w", err)
	}
	id := aws.ToString(created.UserPoolClient.ClientId)
	logf("app client %s criado (%s)", clientName, id)
	return id, nil
}

func gravarEnv(cfg config, info poolInfo) error {
	conteudo := strings.Join([]string{
		"# Gerado por `task local:up` / `task local:seed`. Não edite.",
		"COGNITO_ENDPOINT=" + cfg.cognitoEndpoint,
		"COGNITO_USER_POOL_ID=" + info.poolID,
		"COGNITO_CLIENT_ID=" + info.clientID,
		"COGNITO_JWKS_URL=" + cfg.cognitoEndpoint + "/" + info.poolID + "/.well-known/jwks.json",
		"COGNITO_ISSUER=" + cfg.cognitoEndpoint + "/" + info.poolID,
		"",
	}, "\n")
	if err := os.WriteFile(cfg.envFile, []byte(conteudo), 0o644); err != nil {
		return fmt.Errorf("gravar %s: %w", cfg.envFile, err)
	}
	logf("IDs gravados em %s", cfg.envFile)
	return nil
}

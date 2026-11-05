package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Tabelas de cada serviço. Estas chaves são o contrato com o CDK de cada
// repositório: uma mudança aqui exige a mesma mudança lá, e vice-versa.
var tabelas = []struct {
	nome string
	gsis []string // nome de cada GSI; as chaves são <nome>PK e <nome>SK
}{
	// users-service: PK=USER#<id>, SK=PROFILE; GSI1 lista todos para o admin
	// (GSI1PK=USERS, GSI1SK=<criadoEm>#<id>).
	{nome: "gestor-gastos-users", gsis: []string{"GSI1"}},
	// categories-service: PK=SYSTEM|USER#<id>, SK=CAT#<grupo>#<id>.
	{nome: "gestor-gastos-categories"},
	// expenses-service: PK=USER#<id>, SK=EXP#<AAAA-MM-DD>#<ulid>.
	{nome: "gestor-gastos-expenses"},
}

func dynamoClient(cfg config) *dynamodb.Client {
	return dynamodb.New(dynamodb.Options{
		Region:       region,
		BaseEndpoint: aws.String(cfg.dynamoEndpoint),
		Credentials:  credentials.NewStaticCredentialsProvider("local", "local", ""),
	})
}

func seedTabelas(ctx context.Context, cfg config) error {
	fmt.Println("DynamoDB Local:")
	c := dynamoClient(cfg)
	for _, t := range tabelas {
		in := &dynamodb.CreateTableInput{
			TableName:   aws.String(t.nome),
			BillingMode: types.BillingModePayPerRequest,
			AttributeDefinitions: []types.AttributeDefinition{
				{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
				{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
			},
			KeySchema: []types.KeySchemaElement{
				{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
				{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
			},
		}
		for _, g := range t.gsis {
			pk, sk := g+"PK", g+"SK"
			in.AttributeDefinitions = append(in.AttributeDefinitions,
				types.AttributeDefinition{AttributeName: aws.String(pk), AttributeType: types.ScalarAttributeTypeS},
				types.AttributeDefinition{AttributeName: aws.String(sk), AttributeType: types.ScalarAttributeTypeS},
			)
			in.GlobalSecondaryIndexes = append(in.GlobalSecondaryIndexes, types.GlobalSecondaryIndex{
				IndexName: aws.String(g),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String(pk), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String(sk), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			})
		}

		_, err := c.CreateTable(ctx, in)
		var inUse *types.ResourceInUseException
		switch {
		case err == nil:
			logf("tabela %s criada", t.nome)
		case errors.As(err, &inUse):
			logf("tabela %s já existe", t.nome)
		default:
			return fmt.Errorf("criar tabela %s: %w", t.nome, err)
		}
	}
	return nil
}

func seedCategorias(ctx context.Context, cfg config) error {
	c := dynamoClient(cfg)
	for _, cat := range categoriasPadrao {
		_, err := c.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String("gestor-gastos-categories"),
			Item: map[string]types.AttributeValue{
				"PK":          &types.AttributeValueMemberS{Value: "SYSTEM"},
				"SK":          &types.AttributeValueMemberS{Value: "CAT#" + cat.grupo + "#" + cat.id},
				"categoriaId": &types.AttributeValueMemberS{Value: cat.id},
				"dono":        &types.AttributeValueMemberS{Value: "SYSTEM"},
				"grupo":       &types.AttributeValueMemberS{Value: cat.grupo},
				"nome":        &types.AttributeValueMemberS{Value: cat.nome},
				"cor":         &types.AttributeValueMemberS{Value: cat.cor},
			},
		})
		if err != nil {
			return fmt.Errorf("gravar categoria %s: %w", cat.id, err)
		}
	}
	logf("%d categorias padrão gravadas", len(categoriasPadrao))
	return nil
}

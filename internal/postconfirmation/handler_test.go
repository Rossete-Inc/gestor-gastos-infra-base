package postconfirmation

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Rossete-Inc/gestor-gastos-go-commons/events"
	"github.com/Rossete-Inc/gestor-gastos-go-commons/logging"
	lambdaevents "github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	cip "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
)

type cognitoFake struct {
	chamadas []*cip.AdminAddUserToGroupInput
	err      error
}

func (f *cognitoFake) AdminAddUserToGroup(_ context.Context, in *cip.AdminAddUserToGroupInput, _ ...func(*cip.Options)) (*cip.AdminAddUserToGroupOutput, error) {
	f.chamadas = append(f.chamadas, in)
	if f.err != nil {
		return nil, f.err
	}
	return &cip.AdminAddUserToGroupOutput{}, nil
}

type publicado struct {
	detailType string
	detail     any
}

type publisherFake struct {
	publicados []publicado
	err        error
}

func (p *publisherFake) Publish(_ context.Context, detailType string, detail any) error {
	p.publicados = append(p.publicados, publicado{detailType, detail})
	return p.err
}

func novoHandler(t *testing.T) (*Handler, *cognitoFake, *publisherFake, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	c, p := &cognitoFake{}, &publisherFake{}
	h := &Handler{
		Cognito: c,
		Eventos: p,
		Log:     logging.New(logging.Options{Service: "post-confirmation", Env: "dev", Output: &logs}),
	}
	return h, c, p, &logs
}

func eventoCadastro() Evento {
	var ev Evento
	ev.TriggerSource = GatilhoCadastro
	ev.UserPoolID = "us-east-1_abc123"
	ev.UserName = "8f0c-uuid"
	ev.Request = lambdaevents.CognitoEventUserPoolsPostConfirmationRequest{
		UserAttributes: map[string]string{
			"sub":   "8f0c-uuid",
			"email": "maria@exemplo.com",
			"name":  "Maria Silva",
		},
	}
	return ev
}

func TestConfirmacaoDeCadastroAdicionaAoGrupoEPublica(t *testing.T) {
	h, c, p, logs := novoHandler(t)
	ev := eventoCadastro()

	got, err := h.Handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, ev) {
		t.Errorf("o evento deve voltar sem alteração: %+v", got)
	}

	if len(c.chamadas) != 1 {
		t.Fatalf("AdminAddUserToGroup chamado %d vezes", len(c.chamadas))
	}
	in := c.chamadas[0]
	if aws.ToString(in.UserPoolId) != "us-east-1_abc123" || aws.ToString(in.Username) != "8f0c-uuid" || aws.ToString(in.GroupName) != "cliente" {
		t.Errorf("entrada = pool %q, user %q, grupo %q", aws.ToString(in.UserPoolId), aws.ToString(in.Username), aws.ToString(in.GroupName))
	}

	want := []publicado{{events.UserConfirmed, events.UserConfirmedDetail{UserID: "8f0c-uuid", Email: "maria@exemplo.com", Nome: "Maria Silva"}}}
	if !reflect.DeepEqual(p.publicados, want) {
		t.Errorf("publicados = %+v, quer %+v", p.publicados, want)
	}

	if strings.Contains(logs.String(), "maria@exemplo.com") || strings.Contains(logs.String(), "Maria Silva") {
		t.Errorf("log com dado pessoal: %s", logs.String())
	}
	if !strings.Contains(logs.String(), "8f0c-uuid") {
		t.Errorf("o log deve identificar o usuário pelo userId: %s", logs.String())
	}
}

func TestTrocaDeSenhaEsquecidaEIgnorada(t *testing.T) {
	h, c, p, _ := novoHandler(t)
	ev := eventoCadastro()
	ev.TriggerSource = "PostConfirmation_ConfirmForgotPassword"

	got, err := h.Handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, ev) {
		t.Errorf("evento alterado: %+v", got)
	}
	if len(c.chamadas) != 0 || len(p.publicados) != 0 {
		t.Errorf("não deveria mexer em grupo nem publicar: %d chamadas, %d eventos", len(c.chamadas), len(p.publicados))
	}
}

func TestEventoIncompletoFalha(t *testing.T) {
	for _, caso := range []struct {
		nome   string
		altera func(*Evento)
	}{
		{"sem sub", func(e *Evento) { delete(e.Request.UserAttributes, "sub") }},
		{"sem userName", func(e *Evento) { e.UserName = "" }},
		{"sem userPoolId", func(e *Evento) { e.UserPoolID = "" }},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			h, c, p, _ := novoHandler(t)
			ev := eventoCadastro()
			caso.altera(&ev)

			if _, err := h.Handle(context.Background(), ev); err == nil {
				t.Fatal("esperava erro")
			}
			if len(c.chamadas) != 0 || len(p.publicados) != 0 {
				t.Error("não deveria chamar o Cognito nem publicar")
			}
		})
	}
}

func TestFalhaNoGrupoDevolveErroENaoPublica(t *testing.T) {
	h, c, p, _ := novoHandler(t)
	c.err = errors.New("TooManyRequestsException")

	_, err := h.Handle(context.Background(), eventoCadastro())
	if err == nil || !strings.Contains(err.Error(), "TooManyRequestsException") || !strings.Contains(err.Error(), "cliente") {
		t.Fatalf("erro = %v", err)
	}
	if len(p.publicados) != 0 {
		t.Error("sem o grupo, o evento não deveria ser publicado")
	}
}

func TestFalhaAoPublicarNaoImpedeAConfirmacao(t *testing.T) {
	h, c, p, logs := novoHandler(t)
	p.err = errors.New("publicar UserConfirmed em gestor-gastos-dev: AccessDenied")

	if _, err := h.Handle(context.Background(), eventoCadastro()); err != nil {
		t.Fatalf("a falha ao publicar não deve falhar a confirmação: %v", err)
	}
	if len(c.chamadas) != 1 {
		t.Error("o grupo deveria ter sido adicionado")
	}
	if !strings.Contains(logs.String(), `"level":"ERROR"`) || !strings.Contains(logs.String(), "AccessDenied") {
		t.Errorf("esperava log de erro: %s", logs.String())
	}
}

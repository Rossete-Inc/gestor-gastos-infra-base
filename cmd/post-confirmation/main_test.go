package main

import (
	"context"
	"strings"
	"testing"
)

func lookup(vars map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := vars[k]
		return v, ok
	}
}

func TestNovoHandlerExigeConfiguracao(t *testing.T) {
	for _, caso := range []struct {
		nome string
		vars map[string]string
		erro string
	}{
		{"sem RUNTIME", map[string]string{"ENV": "dev", "EVENT_BUS_NAME": "gestor-gastos-dev"}, "RUNTIME"},
		{"local", map[string]string{"RUNTIME": "local", "EVENT_BUS_NAME": "gestor-gastos-dev"}, "RUNTIME=aws"},
		{"sem barramento", map[string]string{"RUNTIME": "aws", "ENV": "dev"}, "EVENT_BUS_NAME"},
	} {
		t.Run(caso.nome, func(t *testing.T) {
			_, err := novoHandler(context.Background(), lookup(caso.vars))
			if err == nil || !strings.Contains(err.Error(), caso.erro) {
				t.Fatalf("erro = %v, quer algo com %q", err, caso.erro)
			}
		})
	}
}

func TestNovoHandlerNaAws(t *testing.T) {
	// Sem chamadas de rede: só monta os clients.
	t.Setenv("AWS_ACCESS_KEY_ID", "teste")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "teste")
	h, err := novoHandler(context.Background(), lookup(map[string]string{
		"RUNTIME": "aws", "ENV": "dev", "EVENT_BUS_NAME": "gestor-gastos-dev",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if h.Cognito == nil || h.Eventos == nil || h.Log == nil {
		t.Errorf("handler incompleto: %+v", h)
	}
}

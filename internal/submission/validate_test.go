package submission

import (
	"testing"

	"github.com/arthursa-source/forms/internal/store"
)

// Definição usada nos casos: um campo de cada tipo.
func fields() []store.Field {
	return []store.Field{
		{ID: 1, Position: 0, Label: "Nome", Type: store.FieldText, Required: true},
		{ID: 2, Position: 1, Label: "Idade", Type: store.FieldNumber},
		{ID: 3, Position: 2, Label: "Canal", Type: store.FieldSelect, Required: true,
			Options: []string{"Google", "Indicação"}},
		{ID: 4, Position: 3, Label: "Termos", Type: store.FieldCheckbox, Required: true},
	}
}

func TestValidateAceitaRespostaCompleta(t *testing.T) {
	answers, errs := Validate(fields(), map[string]any{
		"1": "  Maria  ", "2": 34.0, "3": "Google", "4": true,
	})
	if len(errs) > 0 {
		t.Fatalf("não esperava erros, recebi %v", errs)
	}
	want := []string{"Maria", "34", "Google", "true"}
	for i, w := range want {
		if answers[i].Value != w {
			t.Errorf("campo %d: valor %q, esperado %q", i, answers[i].Value, w)
		}
	}
}

func TestValidateRecusaValoresInvalidos(t *testing.T) {
	casos := map[string]map[string]any{
		"obrigatório ausente":   {"3": "Google", "4": true},
		"obrigatório em branco": {"1": "   ", "3": "Google", "4": true},
		"número inválido":       {"1": "X", "2": "abc", "3": "Google", "4": true},
		"opção fora da lista":   {"1": "X", "3": "Facebook", "4": true},
		"checkbox desmarcado":   {"1": "X", "3": "Google", "4": false},
		"tipo errado no texto":  {"1": 42.0, "3": "Google", "4": true},
		"campo inexistente":     {"1": "X", "3": "Google", "4": true, "77": "intruso"},
	}
	for nome, answers := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, errs := Validate(fields(), answers); len(errs) == 0 {
				t.Fatalf("esperava erro de validação")
			}
		})
	}
}

func TestValidateCamposOpcionaisPodemFaltar(t *testing.T) {
	// Idade é opcional: ausente vira string vazia, sem erro.
	answers, errs := Validate(fields(), map[string]any{
		"1": "Ana", "3": "Indicação", "4": true,
	})
	if len(errs) > 0 {
		t.Fatalf("não esperava erros, recebi %v", errs)
	}
	if answers[1].Value != "" {
		t.Errorf("idade ausente deveria virar vazio, veio %q", answers[1].Value)
	}
}

func TestValidateNumeroAceitaTexto(t *testing.T) {
	// <input type="number"> no HTML entrega string.
	answers, errs := Validate(fields(), map[string]any{
		"1": "Ana", "2": "42", "3": "Google", "4": true,
	})
	if len(errs) > 0 {
		t.Fatalf("não esperava erros, recebi %v", errs)
	}
	if answers[1].Value != "42" {
		t.Errorf("valor %q, esperado \"42\"", answers[1].Value)
	}
}

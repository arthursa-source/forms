// Package submission valida uma resposta contra a definição do formulário.
//
// Esta é a regra central do requisito "o backend valida a submissão de
// acordo com a definição do formulário": campos obrigatórios presentes,
// valores do tipo certo e, no select, apenas opções válidas. Fica num
// pacote separado (sem dependência de HTTP ou de banco) para ser fácil de
// ler e de testar.
package submission

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/arthursa-source/forms/internal/store"
)

// FieldError aponta o problema em um campo específico.
type FieldError struct {
	FieldID int64
	Label   string
	Message string
}

func (e FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Label, e.Message)
}

// Validate recebe os valores enviados (chave = id do campo, em texto, como
// chega no JSON) e devolve as respostas normalizadas na ordem da definição.
// Se algo estiver errado, devolve TODOS os erros de uma vez — é mais útil
// para o formulário do que parar no primeiro.
func Validate(fields []store.Field, answers map[string]any) ([]store.Answer, []FieldError) {
	var errs []FieldError
	out := make([]store.Answer, 0, len(fields))

	// Chaves enviadas que não correspondem a nenhum campo: recusamos, para
	// não guardar dado que o formulário não pediu.
	known := map[string]bool{}
	for _, f := range fields {
		known[strconv.FormatInt(f.ID, 10)] = true
	}
	for key := range answers {
		if !known[key] {
			errs = append(errs, FieldError{
				Label:   key,
				Message: "campo não pertence a este formulário",
			})
		}
	}

	for _, field := range fields {
		raw, sent := answers[strconv.FormatInt(field.ID, 10)]

		value, err := normalize(field, raw, sent)
		if err != nil {
			errs = append(errs, FieldError{FieldID: field.ID, Label: field.Label, Message: err.Error()})
			continue
		}

		out = append(out, store.Answer{
			Position: field.Position,
			Label:    field.Label,
			Type:     field.Type,
			Value:    value,
		})
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return out, nil
}

// normalize converte o valor recebido para a forma de texto que vai ao
// banco, aplicando as regras do tipo do campo.
func normalize(field store.Field, raw any, sent bool) (string, error) {
	// Ausente, nulo ou string vazia contam como "não respondido".
	empty := !sent || raw == nil
	if s, ok := raw.(string); ok && strings.TrimSpace(s) == "" {
		empty = true
	}

	switch field.Type {
	case store.FieldText:
		if empty {
			if field.Required {
				return "", fmt.Errorf("campo obrigatório")
			}
			return "", nil
		}
		s, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("esperado texto")
		}
		return strings.TrimSpace(s), nil

	case store.FieldNumber:
		if empty {
			if field.Required {
				return "", fmt.Errorf("campo obrigatório")
			}
			return "", nil
		}
		n, err := toNumber(raw)
		if err != nil {
			return "", err
		}
		// strconv com -1 preserva a precisão sem notação científica curta.
		return strconv.FormatFloat(n, 'f', -1, 64), nil

	case store.FieldSelect:
		if empty {
			if field.Required {
				return "", fmt.Errorf("campo obrigatório")
			}
			return "", nil
		}
		s, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("esperado uma das opções (texto)")
		}
		if !slices.Contains(field.Options, s) {
			return "", fmt.Errorf("opção inválida; use uma de: %s", strings.Join(field.Options, ", "))
		}
		return s, nil

	case store.FieldCheckbox:
		// Checkbox ausente equivale a não marcado.
		if empty {
			if field.Required {
				return "", fmt.Errorf("é preciso marcar esta opção")
			}
			return "false", nil
		}
		b, ok := raw.(bool)
		if !ok {
			return "", fmt.Errorf("esperado true ou false")
		}
		if field.Required && !b {
			return "", fmt.Errorf("é preciso marcar esta opção")
		}
		return strconv.FormatBool(b), nil

	default:
		return "", fmt.Errorf("tipo de campo desconhecido: %s", field.Type)
	}
}

// toNumber aceita número JSON e também número em texto ("42"), porque um
// <input type="number"> em HTML entrega string.
func toNumber(raw any) (float64, error) {
	switch v := raw.(type) {
	case float64:
		return v, nil
	case json.Number:
		return v.Float64()
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, fmt.Errorf("esperado número")
		}
		return n, nil
	default:
		return 0, fmt.Errorf("esperado número")
	}
}

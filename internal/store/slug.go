package store

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
	"unicode"
)

// NewSlug monta a parte pública da URL: o título "achatado" mais um
// sufixo aleatório. O sufixo evita colisão entre títulos iguais e impede
// que alguém adivinhe os formulários de outra pessoa pelo nome.
func NewSlug(title string) string {
	base := slugify(title)
	if base == "" {
		base = "formulario"
	}
	if len(base) > 40 {
		base = strings.Trim(base[:40], "-")
	}
	return base + "-" + randomSuffix()
}

func slugify(s string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		// Acentos comuns do português viram a letra base.
		case strings.ContainsRune("áàâãä", r):
			b.WriteRune('a')
			lastDash = false
		case strings.ContainsRune("éèêë", r):
			b.WriteRune('e')
			lastDash = false
		case strings.ContainsRune("íìîï", r):
			b.WriteRune('i')
			lastDash = false
		case strings.ContainsRune("óòôõö", r):
			b.WriteRune('o')
			lastDash = false
		case strings.ContainsRune("úùûü", r):
			b.WriteRune('u')
			lastDash = false
		case r == 'ç':
			b.WriteRune('c')
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func randomSuffix() string {
	buf := make([]byte, 5) // 5 bytes -> 8 caracteres em base32
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand não falha na prática; se falhar, não publique.
		panic("não foi possível gerar aleatoriedade: " + err.Error())
	}
	enc := base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)
	return enc.EncodeToString(buf)
}

// Package auth concentra hash de senha e emissão/validação de JWT.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword gera o hash bcrypt da senha. O "cost" default do pacote
// (10) é um equilíbrio razoável entre segurança e tempo de login.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword devolve true se a senha corresponde ao hash.
// O bcrypt guarda o salt dentro do próprio hash, então não precisamos
// armazenar nada além dele.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

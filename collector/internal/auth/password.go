package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword gera o hash bcrypt de uma senha
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPassword verifica se a senha bate com o hash
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

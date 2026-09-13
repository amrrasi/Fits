package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

func HashPassword(plain string) (string, error) {
	if len(plain) < 8 {
		return "", fmt.Errorf("پژوهشگر عزیز رمز عبور شما باید حداقل 8 کاراکتر باشد")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(b), nil
}

func CheckPassword(plain, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return fmt.Errorf("متاسفانه پسوورد شما صحیح نمیباشد! ")
	}
	return nil
}

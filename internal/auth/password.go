package auth

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"github.com/amrrasi/fits/internal/api"
)

const (
	bcryptCost     = 12
	MinPasswordLen = 10
	maxPasswordLen = 72 // bcrypt silently ignores anything beyond 72 bytes
)

var commonPasswords = map[string]bool{
	"password": true, "password1": true, "1234567890": true, "12345678910": true, "qwertyuiop": true,
	"admin12345": true, "admin@1234": true, "admin@12345": true, "letmein123": true, "iloveyou12": true,
	"1q2w3e4r5t": true, "abcd123456": true, "0123456789": true, "welcome123": true, "changeme123": true,
}

// ValidatePassword enforces the password policy. email may be empty.
func ValidatePassword(pw, email string) error {
	if len(pw) < MinPasswordLen {
		return api.Invalid(fmt.Sprintf("رمز عبور باید حداقل %d کاراکتر باشد", MinPasswordLen))
	}
	if len(pw) > maxPasswordLen {
		return api.Invalid("رمز عبور نباید بیشتر از ۷۲ بایت باشد")
	}
	low := strings.ToLower(pw)
	if commonPasswords[low] {
		return api.Invalid("این رمز عبور بسیار رایج است؛ رمز قوی‌تری انتخاب کنید")
	}
	same := true
	for i := 1; i < len(pw); i++ {
		if pw[i] != pw[0] {
			same = false
			break
		}
	}
	if same {
		return api.Invalid("رمز عبور نباید فقط از یک کاراکتر تکراری تشکیل شده باشد")
	}
	if local, _, ok := strings.Cut(strings.ToLower(email), "@"); ok && len(local) >= 4 && strings.Contains(low, local) {
		return api.Invalid("رمز عبور نباید شامل بخش نام‌کاربری ایمیل شما باشد")
	}
	letters, digits := false, false
	for _, c := range pw {
		if c >= '0' && c <= '9' {
			digits = true
		} else {
			letters = true
		}
	}
	if !letters || !digits {
		return api.Invalid("رمز عبور باید ترکیبی از حروف و اعداد باشد")
	}
	return nil
}

func HashPassword(plain string) (string, error) {
	if len(plain) > maxPasswordLen {
		return "", api.Invalid("رمز عبور نباید بیشتر از ۷۲ بایت باشد")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(b), nil
}

func CheckPassword(plain, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return fmt.Errorf("رمز عبور نادرست است")
	}
	return nil
}

var (
	dummyOnce sync.Once
	dummyHash string
)

// burnPasswordCheck spends the same CPU as a real check (timing equalisation for unknown users).
func burnPasswordCheck(plain string) {
	dummyOnce.Do(func() {
		b, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
		dummyHash = string(b)
	})
	_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(plain))
}

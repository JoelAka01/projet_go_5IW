package auth

import "testing"

func TestHashPasswordAndCheckPassword(t *testing.T) {
	password := "S3cur3P@ssw0rd!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword a retourné une erreur inattendue: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword a retourné un hash vide")
	}
	if hash == password {
		t.Fatal("HashPassword ne doit pas retourner le mot de passe en clair")
	}

	if !CheckPassword(hash, password) {
		t.Error("CheckPassword devrait valider le bon mot de passe")
	}
	if CheckPassword(hash, "mauvais-mot-de-passe") {
		t.Error("CheckPassword ne devrait pas valider un mauvais mot de passe")
	}
}

func TestGenerateConfirmationCode(t *testing.T) {
	code, err := GenerateConfirmationCode()
	if err != nil {
		t.Fatalf("GenerateConfirmationCode a retourné une erreur inattendue: %v", err)
	}
	if len(code) != 8 {
		t.Fatalf("longueur attendue 8, obtenu %d (%q)", len(code), code)
	}
	for _, c := range code {
		if !isAlphanumeric(byte(c)) {
			t.Fatalf("caractère non alphanumérique dans le code: %q", code)
		}
	}

	other, err := GenerateConfirmationCode()
	if err != nil {
		t.Fatalf("GenerateConfirmationCode a retourné une erreur inattendue: %v", err)
	}
	if code == other {
		t.Error("deux appels ne devraient pas générer le même code")
	}
}

func TestGenerateSessionToken(t *testing.T) {
	token, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken a retourné une erreur inattendue: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateSessionToken a retourné un token vide")
	}

	other, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken a retourné une erreur inattendue: %v", err)
	}
	if token == other {
		t.Error("deux appels ne devraient pas générer le même token")
	}
}

func isAlphanumeric(b byte) bool {
	for i := 0; i < len(alphanumericChars); i++ {
		if alphanumericChars[i] == b {
			return true
		}
	}
	return false
}

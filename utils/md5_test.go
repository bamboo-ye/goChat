package utils

import "testing"

func TestPasswordCompatibility(t *testing.T) {
	plain := "correct horse battery staple"
	legacySalt := "123456"
	legacyHash := Md5Encode(plain + legacySalt)

	if !ValidPassword(plain, legacySalt, legacyHash) {
		t.Fatal("legacy password should remain valid")
	}
	if !ValidPassword(plain, "", MakePassword(plain, "")) {
		t.Fatal("bcrypt password should be valid")
	}
}

func TestNewTokenIsRandom(t *testing.T) {
	first, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 || first == second {
		t.Fatalf("unexpected tokens: %q %q", first, second)
	}
}

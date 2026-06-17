package models

import (
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	request := httptest.NewRequest("GET", "http://example.com/chat", nil)
	request.Host = "example.com"
	request.Header.Set("Origin", "http://example.com")
	if !sameOrigin(request) {
		t.Fatal("same origin should be accepted")
	}

	request.Header.Set("Origin", "http://attacker.example")
	if sameOrigin(request) {
		t.Fatal("cross origin should be rejected")
	}
}

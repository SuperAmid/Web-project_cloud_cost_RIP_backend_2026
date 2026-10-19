package auth

import (
	"testing"
	"time"
)

func TestIssueAndParse(t *testing.T) {
	m := NewManager("secret", time.Hour)
	token, expires, err := m.Issue(7, "a@b.ru")
	if err != nil || time.Until(expires) < 59*time.Minute {
		t.Fatalf("issue: %v %v", err, expires)
	}
	claims, err := m.Parse(token)
	if err != nil || claims.UserID != 7 || claims.Email != "a@b.ru" {
		t.Fatalf("parse: %+v %v", claims, err)
	}
	if _, err := NewManager("other", time.Hour).Parse(token); err == nil {
		t.Fatal("токен с чужой подписью должен отклоняться")
	}
	expired, _, _ := NewManager("secret", -time.Minute).Issue(7, "a@b.ru")
	if _, err := m.Parse(expired); err == nil {
		t.Fatal("просроченный токен должен отклоняться")
	}
}

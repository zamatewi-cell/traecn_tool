package auth

import (
	"strings"
	"testing"
)

func TestStorageCandidates(t *testing.T) {
	cands := StorageCandidates()
	if len(cands) == 0 {
		t.Fatal("StorageCandidates() returned no paths")
	}
	if !strings.Contains(cands[0], "Trae CN") {
		t.Errorf("first candidate = %q, want Trae CN to have priority", cands[0])
	}
	if DefaultStoragePath() != cands[0] {
		t.Errorf("DefaultStoragePath() = %q, want %q", DefaultStoragePath(), cands[0])
	}
}

func TestSniffAccounts_EnvVar(t *testing.T) {
	t.Setenv("TRAE_CN_TOKEN", "env_tok_123")

	found := SniffAccounts()
	var envAcc *SniffedAccount
	for i := range found {
		if found[i].Source.Type == SourceEnv {
			envAcc = &found[i]
			break
		}
	}
	if envAcc == nil {
		t.Fatal("SniffAccounts() did not pick up TRAE_CN_TOKEN")
	}
	if envAcc.Source.EnvVar != "TRAE_CN_TOKEN" {
		t.Errorf("env account var = %q, want TRAE_CN_TOKEN", envAcc.Source.EnvVar)
	}

	tok, err := envAcc.Source.Load()
	if err != nil {
		t.Fatalf("env source Load() error = %v", err)
	}
	if tok.AccessToken != "env_tok_123" {
		t.Errorf("env token = %q, want env_tok_123", tok.AccessToken)
	}
}

func TestCredentialSource_EnvEmpty(t *testing.T) {
	src := CredentialSource{Type: SourceEnv, EnvVar: "DEFINITELY_NOT_SET_VAR_XYZ"}
	if _, err := src.Load(); err == nil {
		t.Error("Load() error = nil, want error for empty env var")
	}
}

func TestCredentialSource_StaticTokenNotReloadable(t *testing.T) {
	src := CredentialSource{Type: SourceToken}
	if _, err := src.Load(); err == nil {
		t.Error("Load() error = nil, want error for static token source")
	}
}

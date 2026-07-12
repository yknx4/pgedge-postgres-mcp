/*-------------------------------------------------------------------------
 *
 * pgEdge Natural Language Agent
 *
 * Copyright (c) 2025 - 2026, pgEdge, Inc.
 * This software is released under The PostgreSQL License
 *
 *-------------------------------------------------------------------------
 */

package tools

import (
	"strings"
	"testing"

	"pgedge-postgres-mcp/internal/config"
)

func boolPointer(value bool) *bool {
	return &value
}

func TestPIIMaskerDisabledByDefault(t *testing.T) {
	masker, enabled := newPIIMasker(config.PIIConfig{}, nil)
	if enabled || masker != nil {
		t.Fatal("PII masking should be disabled by default")
	}
}

func TestPIIMaskerUsesColumnNamesOnly(t *testing.T) {
	masker, enabled := newPIIMasker(config.PIIConfig{Enabled: boolPointer(true)}, nil)
	if !enabled {
		t.Fatal("PII masking should be enabled")
	}

	rows := [][]any{{"person@example.com", "person@example.com"}}
	masked := masker.mask([]string{"email", "notes"}, rows)

	if masked != 1 {
		t.Fatalf("masked %d values, want 1", masked)
	}
	if rows[0][0] == "person@example.com" {
		t.Error("email column was not masked")
	}
	if rows[0][1] != "person@example.com" {
		t.Error("unrecognized notes column was masked based on its value")
	}
}

func TestPIIMaskerProducesConsistentRealisticValues(t *testing.T) {
	masker, _ := newPIIMasker(config.PIIConfig{Enabled: boolPointer(true)}, nil)
	rows := [][]any{
		{"person@example.com", "555-0100"},
		{"person@example.com", "555-0100"},
	}

	masker.mask([]string{"email", "phone_number"}, rows)

	if rows[0][0] != rows[1][0] || rows[0][1] != rows[1][1] {
		t.Error("equal PII values should receive equal replacements within a response")
	}
	if !strings.Contains(rows[0][0].(string), "@") {
		t.Errorf("email replacement %q is not realistic", rows[0][0])
	}
}

func TestPIIMaskerConfigurationAndDatabaseOverride(t *testing.T) {
	global := config.PIIConfig{
		Enabled: boolPointer(true),
		Columns: map[string][]string{"email": {"primary_contact"}},
	}
	disabled := config.PIIConfig{Enabled: boolPointer(false)}
	if _, enabled := newPIIMasker(global, &disabled); enabled {
		t.Error("database override should disable global PII masking")
	}

	masker, enabled := newPIIMasker(global, nil)
	if !enabled {
		t.Fatal("global PII masking should be enabled")
	}
	rows := [][]any{{"person@example.com"}}
	if masked := masker.mask([]string{"primary_contact"}, rows); masked != 1 {
		t.Errorf("configured column masked %d values, want 1", masked)
	}
}

func TestPIIMaskerMasksTokensAndPasswords(t *testing.T) {
	masker, _ := newPIIMasker(config.PIIConfig{Enabled: boolPointer(true)}, nil)
	rows := [][]any{{"live-token-value", "real-password"}}

	if masked := masker.mask([]string{"access_token", "password"}, rows); masked != 2 {
		t.Fatalf("masked %d values, want 2", masked)
	}
	if rows[0][0] == "live-token-value" || rows[0][1] == "real-password" {
		t.Error("token or password was not replaced")
	}
	if !strings.Contains(rows[0][0].(string), "-") {
		t.Errorf("token replacement %q is not a realistic opaque token", rows[0][0])
	}
}

func TestPIIMaskerMasksAuthenticationColumns(t *testing.T) {
	masker, enabled := newPIIMasker(config.PIIConfig{Enabled: boolPointer(true)}, nil)
	if !enabled {
		t.Fatal("PII masking should be enabled")
	}

	rows := [][]any{{
		"encrypted-password",
		"reset-password-token",
		"serialized-tokens",
		"otp-secret",
	}}
	columns := []string{"encrypted_password", "reset_password_token", "tokens", "otp_secret"}

	if masked := masker.mask(columns, rows); masked != len(columns) {
		t.Fatalf("masked %d authentication values, want %d", masked, len(columns))
	}
	for index, original := range []string{
		"encrypted-password",
		"reset-password-token",
		"serialized-tokens",
		"otp-secret",
	} {
		if rows[0][index] == original {
			t.Errorf("%s was not masked", columns[index])
		}
	}
}

func TestPIIMaskerGenericMaskPreservesEdgesAndNil(t *testing.T) {
	masker, _ := newPIIMasker(config.PIIConfig{
		Enabled: boolPointer(true),
		Columns: map[string][]string{"generic": {"private_value"}},
	}, nil)
	rows := [][]any{
		{"sk_live_abc123xyz"},
		{nil},
		{"ab"},
	}

	if masked := masker.mask([]string{"private_value"}, rows); masked != 2 {
		t.Fatalf("masked %d values, want 2", masked)
	}
	if rows[0][0] != "s***************z" {
		t.Errorf("generic mask = %q, want edge-preserving mask", rows[0][0])
	}
	if rows[1][0] != nil {
		t.Errorf("nil value = %v, want nil", rows[1][0])
	}
	if rows[2][0] != "**" {
		t.Errorf("short generic mask = %q, want fully masked", rows[2][0])
	}
}

func TestPerformanceQueriesBypassPIIHandling(t *testing.T) {
	for _, query := range []string{
		"EXPLAIN SELECT * FROM users",
		" explain analyze SELECT * FROM users",
		"ANALYZE users",
		"-- performance check\nEXPLAIN SELECT * FROM users",
		"/* performance check */ EXPLAIN SELECT * FROM users",
	} {
		if !isPerformanceQuery(query) {
			t.Errorf("isPerformanceQuery(%q) = false, want true", query)
		}
	}
	if isPerformanceQuery("SELECT * FROM users") {
		t.Error("ordinary SELECT was classified as a performance query")
	}
}

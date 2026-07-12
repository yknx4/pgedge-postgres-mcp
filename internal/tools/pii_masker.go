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
	"fmt"
	"strings"
	"unicode"

	"github.com/brianvoe/gofakeit/v7"

	"pgedge-postgres-mcp/internal/config"
)

var defaultPIIColumns = map[string][]string{
	"address":     {"address", "address_1", "address_2", "mailing_address", "street_address"},
	"city":        {"city"},
	"credit_card": {"card_number", "credit_card", "credit_card_number"},
	"email":       {"email", "email_address", "user_email"},
	"first_name":  {"first_name", "given_name"},
	"generic":     {"otp_secret", "tokens"},
	"ip_address":  {"ip_address", "last_ip", "last_sign_in_ip", "remote_ip"},
	"last_name":   {"family_name", "last_name", "surname"},
	"name":        {"contact_name", "customer_name", "full_name", "person_name"},
	"password":    {"encrypted_password", "password", "password_digest", "password_hash"},
	"phone":       {"cell_phone", "mobile", "mobile_number", "phone", "phone_number", "telephone"},
	"postal_code": {"postal_code", "postcode", "zip", "zip_code"},
	"ssn":         {"social_security_number", "ssn"},
	"state":       {"province", "state"},
	"token":       {"access_token", "api_key", "api_token", "auth_token", "bearer_token", "confirmation_token", "refresh_token", "reset_password_token", "secret_token", "session_token", "token"},
	"username":    {"login", "user_name", "username"},
}

type piiMasker struct {
	columnTypes  map[string]string
	faker        *gofakeit.Faker
	replacements map[string]map[string]string
}

func newPIIMasker(global config.PIIConfig, databaseOverride *config.PIIConfig) (*piiMasker, bool) {
	effective := global
	if databaseOverride != nil {
		if databaseOverride.Enabled != nil {
			effective.Enabled = databaseOverride.Enabled
		}
		if databaseOverride.Columns != nil {
			effective.Columns = databaseOverride.Columns
		}
	}

	if effective.Enabled == nil || !*effective.Enabled {
		return nil, false
	}

	columns := make(map[string]string)
	addPIIColumns(columns, defaultPIIColumns)
	addPIIColumns(columns, effective.Columns)

	return &piiMasker{
		columnTypes:  columns,
		faker:        gofakeit.New(0),
		replacements: make(map[string]map[string]string),
	}, true
}

func addPIIColumns(target map[string]string, configured map[string][]string) {
	for piiType, names := range configured {
		for _, name := range names {
			target[normalizeColumnName(name)] = strings.ToLower(piiType)
		}
	}
}

func normalizeColumnName(name string) string {
	var normalized strings.Builder
	for i, r := range strings.TrimSpace(name) {
		if unicode.IsUpper(r) && i > 0 {
			normalized.WriteByte('_')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(unicode.ToLower(r))
		} else if normalized.Len() > 0 {
			normalized.WriteByte('_')
		}
	}
	return strings.Trim(normalized.String(), "_")
}

func (m *piiMasker) mask(columnNames []string, rows [][]any) int {
	masked := 0
	for columnIndex, columnName := range columnNames {
		piiType, ok := m.columnTypes[normalizeColumnName(columnName)]
		if !ok {
			continue
		}
		for rowIndex := range rows {
			if columnIndex >= len(rows[rowIndex]) || rows[rowIndex][columnIndex] == nil {
				continue
			}
			original := fmt.Sprint(rows[rowIndex][columnIndex])
			rows[rowIndex][columnIndex] = m.replacement(piiType, original)
			masked++
		}
	}
	return masked
}

func (m *piiMasker) replacement(piiType, original string) string {
	if m.replacements[piiType] == nil {
		m.replacements[piiType] = make(map[string]string)
	}
	if replacement, ok := m.replacements[piiType][original]; ok {
		return replacement
	}

	var replacement string
	switch piiType {
	case "address":
		replacement = m.faker.Street()
	case "city":
		replacement = m.faker.City()
	case "credit_card":
		replacement = m.faker.CreditCardNumber(nil)
	case "email":
		replacement = m.faker.Email()
	case "first_name":
		replacement = m.faker.FirstName()
	case "ip_address":
		replacement = m.faker.IPv4Address()
	case "last_name":
		replacement = m.faker.LastName()
	case "name":
		replacement = m.faker.Name()
	case "password":
		replacement = m.faker.Password(true, true, true, true, false, 20)
	case "phone":
		replacement = m.faker.PhoneFormatted()
	case "postal_code":
		replacement = m.faker.Zip()
	case "ssn":
		replacement = m.faker.SSN()
	case "state":
		replacement = m.faker.State()
	case "token":
		replacement = m.faker.UUID()
	case "username":
		replacement = m.faker.Username()
	case "generic":
		replacement = genericMask(original)
	default:
		replacement = genericMask(original)
	}

	m.replacements[piiType][original] = replacement
	return replacement
}

func genericMask(value string) string {
	characters := []rune(value)
	if len(characters) < 3 {
		return strings.Repeat("*", len(characters))
	}
	return string(characters[0]) + strings.Repeat("*", len(characters)-2) + string(characters[len(characters)-1])
}

func isPerformanceQuery(query string) bool {
	upper := strings.ToUpper(stripLeadingSQLComments(query))
	return strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "ANALYZE")
}

func stripLeadingSQLComments(query string) string {
	remaining := strings.TrimSpace(query)
	for {
		switch {
		case strings.HasPrefix(remaining, "--"):
			newline := strings.IndexByte(remaining, '\n')
			if newline == -1 {
				return ""
			}
			remaining = strings.TrimSpace(remaining[newline+1:])
		case strings.HasPrefix(remaining, "/*"):
			commentEnd := strings.Index(remaining[2:], "*/")
			if commentEnd == -1 {
				return remaining
			}
			remaining = strings.TrimSpace(remaining[commentEnd+4:])
		default:
			return remaining
		}
	}
}

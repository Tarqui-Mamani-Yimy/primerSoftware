// Bean-equivalent input validation and timestamp formatting (GOBE-02).
//
// The Java backend validates request bodies with Jakarta bean validation
// (@Valid: @NotBlank, @NotNull, @Email) before the service layer runs. This
// file ports those constraints so invalid payloads fail with 400 and the
// uniform {"message"} envelope instead of Spring's default problem detail.
package domain

import (
	"strings"
	"time"
)

// ValidateLoginInput ports AuthController.LoginInput (@NotBlank @Email email,
// @NotBlank password).
func ValidateLoginInput(email, password string) []string {
	var errors []string
	if strings.TrimSpace(email) == "" {
		errors = append(errors, "email must not be blank")
	} else if !strings.Contains(email, "@") {
		errors = append(errors, "email must be a well-formed email address")
	}
	if strings.TrimSpace(password) == "" {
		errors = append(errors, "password must not be blank")
	}
	return errors
}

// ValidateProjectInput ports the bean constraints on project creation: the
// name is @NotBlank and fits projects.name VARCHAR(150).
func ValidateProjectInput(name string) []string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return []string{"name must not be blank"}
	}
	if len([]rune(trimmed)) > 150 {
		return []string{"name must be at most 150 characters"}
	}
	return nil
}

// ValidateAccessCodeInput ports the @NotBlank constraint on the join input.
func ValidateAccessCodeInput(accessCode string) []string {
	if strings.TrimSpace(accessCode) == "" {
		return []string{"accessCode must not be blank"}
	}
	return nil
}

// NormalizeAccessCode canonicalizes a classroom code for storage and lookup:
// surrounding whitespace is ignored and case does not matter, so a code typed
// as "x7k2p9" matches the stored "X7K2P9".
func NormalizeAccessCode(accessCode string) string {
	return strings.ToUpper(strings.TrimSpace(accessCode))
}

// ValidateDiagramInput ports the bean constraints on DiagramDocument and its
// nested records (@NotBlank ids/names/types, @NotNull classes/relationships).
// Semantic rules live in ValidateDocument and run after this passes.
func ValidateDiagramInput(doc DiagramDocument) []string {
	var errors []string
	if strings.TrimSpace(doc.Name) == "" {
		errors = append(errors, "name must not be blank")
	}
	if doc.Classes == nil {
		errors = append(errors, "classes must not be null")
	}
	if doc.Relationships == nil {
		errors = append(errors, "relationships must not be null")
	}
	for i, class := range doc.Classes {
		prefix := "classes[" + itoa(i) + "]."
		if strings.TrimSpace(class.ID) == "" {
			errors = append(errors, prefix+"id must not be blank")
		}
		if strings.TrimSpace(class.Name) == "" {
			errors = append(errors, prefix+"name must not be blank")
		}
		for j, attr := range class.Attributes {
			ap := prefix + "attributes[" + itoa(j) + "]."
			if strings.TrimSpace(attr.ID) == "" {
				errors = append(errors, ap+"id must not be blank")
			}
			if strings.TrimSpace(attr.Name) == "" {
				errors = append(errors, ap+"name must not be blank")
			}
			if strings.TrimSpace(attr.Type) == "" {
				errors = append(errors, ap+"type must not be blank")
			}
		}
		for j, method := range class.Methods {
			mp := prefix + "methods[" + itoa(j) + "]."
			if strings.TrimSpace(method.ID) == "" {
				errors = append(errors, mp+"id must not be blank")
			}
			if strings.TrimSpace(method.Name) == "" {
				errors = append(errors, mp+"name must not be blank")
			}
			if strings.TrimSpace(method.ReturnType) == "" {
				errors = append(errors, mp+"returnType must not be blank")
			}
		}
	}
	for i, relationship := range doc.Relationships {
		prefix := "relationships[" + itoa(i) + "]."
		if strings.TrimSpace(relationship.ID) == "" {
			errors = append(errors, prefix+"id must not be blank")
		}
		if strings.TrimSpace(relationship.SourceID) == "" {
			errors = append(errors, prefix+"sourceId must not be blank")
		}
		if strings.TrimSpace(relationship.TargetID) == "" {
			errors = append(errors, prefix+"targetId must not be blank")
		}
		if strings.TrimSpace(relationship.Type) == "" {
			errors = append(errors, prefix+"type must not be blank")
		}
	}
	return errors
}

// FormatInstant renders a timestamp exactly like Jackson's ISO-8601 encoding of
// java.time.Instant (Spring Boot disables WRITE_DATES_AS_TIMESTAMPS): UTC,
// RFC 3339 with nanoseconds only when present.
func FormatInstant(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	for n := i; n > 0; n /= 10 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
	}
	return string(digits)
}

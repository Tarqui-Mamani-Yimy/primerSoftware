package sqlgen

import "strings"

// postgresqlReservedKeywords is the verbatim PostgreSQL reserved-keyword list
// generator-jhipster 9.4.0 checks before naming a table or column, pinned
// from:
//
//	generator-jhipster@9.4.0/dist/generators/spring-boot/generators/data-relational/support/postgresql-reserved-keywords.js
//
// (re-exported unchanged as ReservedWords.POSTGRESQL in
// dist/lib/jhipster/reserved-keywords.js). isReservedTableName(keyword,
// 'postgresql') in that same file (lines 56-60) calls
// isReserved(keyword, 'postgresql'); since 'postgresql'.toUpperCase() !==
// 'SQL', only this list is consulted — NOT the union with MySQL/Oracle/MSSQL
// that isReservedTableName(keyword, 'sql') would use. The check is
// case-insensitive (keyword.toUpperCase()).
//
// Notably DATE is NOT in this list (only CURRENT_DATE is), even though DATE
// is a SQL type name — PostgreSQL itself classifies DATE as non-reserved.
// USER and ORDER, however, are both reserved.
var postgresqlReservedKeywords = map[string]bool{
	"ALL": true, "ANALYSE": true, "ANALYZE": true, "AND": true, "ANY": true,
	"ARRAY": true, "AS": true, "ASC": true, "ASYMMETRIC": true, "AUTHORIZATION": true,
	"BINARY": true, "BOTH": true, "CASE": true, "CAST": true, "CHECK": true,
	"COLLATE": true, "COLLATION": true, "COLUMN": true, "CONCURRENTLY": true, "CONSTRAINT": true,
	"CREATE": true, "CROSS": true, "CURRENT_CATALOG": true, "CURRENT_DATE": true, "CURRENT_ROLE": true,
	"CURRENT_SCHEMA": true, "CURRENT_TIME": true, "CURRENT_TIMESTAMP": true, "CURRENT_USER": true, "DEFAULT": true,
	"DEFERRABLE": true, "DESC": true, "DISTINCT": true, "DO": true, "ELSE": true,
	"END": true, "EXCEPT": true, "FALSE": true, "FETCH": true, "FOR": true,
	"FOREIGN": true, "FROM": true, "FULL": true, "GRANT": true, "GROUP": true,
	"HAVING": true, "ILIKE": true, "IN": true, "INITIALLY": true, "INNER": true,
	"INTERSECT": true, "INTO": true, "IS": true, "ISNULL": true, "JOIN": true,
	"LATERAL": true, "LEADING": true, "LEFT": true, "LIKE": true, "LIMIT": true,
	"LOCALTIME": true, "LOCALTIMESTAMP": true, "NATURAL": true, "NOT": true, "NOTNULL": true,
	"NULL": true, "OFFSET": true, "ON": true, "ONLY": true, "OR": true,
	"ORDER": true, "OUTER": true, "OVERLAPS": true, "PLACING": true, "PRIMARY": true,
	"REFERENCES": true, "RETURNING": true, "RIGHT": true, "SELECT": true, "SESSION_USER": true,
	"SIMILAR": true, "SOME": true, "SYMMETRIC": true, "TABLE": true, "THEN": true,
	"TO": true, "TRAILING": true, "TRUE": true, "UNION": true, "UNIQUE": true,
	"USER": true, "USING": true, "VARIADIC": true, "VERBOSE": true, "WHEN": true,
	"WHERE": true, "WINDOW": true, "WITH": true,
}

// isReservedPostgresqlName reports whether name (already the candidate
// table/column identifier, e.g. "order" or "current_date") collides with a
// PostgreSQL reserved keyword, case-insensitively.
func isReservedPostgresqlName(name string) bool {
	return postgresqlReservedKeywords[strings.ToUpper(name)]
}

// jhiReservedPrefix is JHipster's default jhiPrefix option ('jhi'); this
// project never exposes a way to change it, so it is pinned here rather than
// threaded through as configuration.
const jhiReservedPrefix = "jhi"

// tableName derives the SQL table name for an entity exactly like
// configureEntityTable in generator-jhipster 9.4.0
// (dist/generators/server/generator.js:117-129): hibernateSnakeCase the
// entity name, then prefix with "jhi_" if that collides with a PostgreSQL
// reserved keyword ("Order" -> "jhi_order").
func tableName(entityName string) string {
	t := hibernateSnakeCase(entityName)
	if isReservedPostgresqlName(t) {
		return jhiReservedPrefix + "_" + t
	}
	return t
}

// columnName derives the SQL column name for a scalar field exactly like
// prepareField in generator-jhipster 9.4.0
// (dist/generators/server/support/prepare-field.js:90-100): lodash
// snakeCase the field name, then prefix with "jhi_" if that collides with a
// PostgreSQL reserved keyword ("user" -> "jhi_user").
func columnName(fieldName string) string {
	c := lodashSnakeCase(fieldName)
	if isReservedPostgresqlName(c) {
		return jhiReservedPrefix + "_" + c
	}
	return c
}

// relationshipColumnBase derives the base name of a relationship's FK
// column, before the "_id" suffix that the referenced entity's primary key
// field contributes, exactly like prepareRelationshipForDatabase in
// generator-jhipster 9.4.0
// (dist/generators/server/support/prepare-relationship.js:24-26):
// hibernateSnakeCase the relationship name. Unlike tableName/columnName, no
// reserved-keyword prefixing is ever applied here — prepareRelationshipForDatabase
// never calls isReservedTableName, so a relationship named e.g. "order" still
// yields the column "order_id", not "jhi_order_id".
func relationshipColumnBase(relationshipName string) string {
	return hibernateSnakeCase(relationshipName)
}

// hibernateSnakeCase ports generator-jhipster 9.4.0's Hibernate naming
// strategy helper verbatim
// (dist/generators/server/support/string.js:26-48), used for table names,
// relationship/join names, and constraint-name components. It inserts an
// underscore before an uppercase letter only when the PRECEDING character is
// not itself uppercase and the FOLLOWING character is not uppercase either —
// so a run of uppercase letters (an acronym) never gets split from a
// following capitalized word ("UMLClass" -> "umlclass", not "uml_class";
// "XMLHttpRequest" -> "xmlhttp_request", not "xml_http_request") and digits
// never trigger a split ("Order2" -> "order2"). This deliberately differs
// from lodashSnakeCase, which JHipster uses for field columns instead.
func hibernateSnakeCase(value string) string {
	if value == "" {
		return ""
	}
	r := []rune(value)
	if len(r) == 1 {
		return strings.ToLower(string(r))
	}
	// value.replace('.', '_') in the original replaces only the first '.'.
	if idx := strings.IndexRune(value, '.'); idx >= 0 {
		r = []rune(value[:idx] + "_" + value[idx+1:])
	}
	var b strings.Builder
	b.WriteRune(r[0])
	for i := 1; i < len(r)-1; i++ {
		prevUpper := isASCIIUpper(r[i-1])
		curUpper := isASCIIUpper(r[i])
		nextUpper := isASCIIUpper(r[i+1])
		if !prevUpper && curUpper && !nextUpper {
			b.WriteRune('_')
		}
		b.WriteRune(r[i])
	}
	b.WriteRune(r[len(r)-1])
	return strings.ToLower(b.String())
}

func isASCIIUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
func isASCIILower(r rune) bool { return r >= 'a' && r <= 'z' }
func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }

// lodashSnakeCase ports lodash's snakeCase(word-splitting via words()/
// unicodeWords()) closely enough for Java-identifier-shaped input (ASCII
// letters and digits, no separators): it is used by generator-jhipster
// 9.4.0's prepareField (dist/generators/server/support/prepare-field.js:91,
// `snakeCase(field.fieldName)` from the 'lodash-es' import) for every field
// column name. Unlike hibernateSnakeCase, it splits a trailing acronym off
// the capitalized word that follows it ("userID" -> "user_id") and always
// splits letters from digits ("address2" -> "address_2"), because lodash
// always emits digit runs as their own "word".
//
// This is a faithful re-implementation of lodash's word-splitting decision
// table (lodash-es@4.18.1 words.js / _unicodeWords.js), not a call into the
// real regex (Go's RE2 has no lookahead, which the original leans on): when
// an uppercase run of length > 1 is immediately followed by a lowercase
// letter, the run's last letter is peeled off to start the next word
// ("IDCard" -> "ID", "Card"); a run of exactly one uppercase letter attaches
// directly to the lowercase run that follows it ("Order" -> "Order"); and
// digit runs are always their own word.
func lodashSnakeCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = strings.ToLower(w)
	}
	return strings.Join(parts, "_")
}

// splitWords implements the word-splitting decision table described on
// lodashSnakeCase above.
func splitWords(s string) []string {
	r := []rune(s)
	n := len(r)
	var words []string
	i := 0
	for i < n {
		switch {
		case isASCIIDigit(r[i]):
			j := i
			for j < n && isASCIIDigit(r[j]) {
				j++
			}
			words = append(words, string(r[i:j]))
			i = j
		case isASCIIUpper(r[i]):
			j := i
			for j < n && isASCIIUpper(r[j]) {
				j++
			}
			// A multi-letter acronym immediately followed by a lowercase
			// letter loses its last letter to the next (capitalized) word.
			if j-i > 1 && j < n && isASCIILower(r[j]) {
				j--
			}
			wordEnd := j
			if wordEnd < n && isASCIILower(r[wordEnd]) {
				k := wordEnd
				for k < n && isASCIILower(r[k]) {
					k++
				}
				wordEnd = k
			}
			words = append(words, string(r[i:wordEnd]))
			i = wordEnd
		case isASCIILower(r[i]):
			j := i
			for j < n && isASCIILower(r[j]) {
				j++
			}
			words = append(words, string(r[i:j]))
			i = j
		default:
			// Non-alphanumeric separator (lodash treats it as a "break"):
			// skip it, it never becomes part of a word.
			i++
		}
	}
	return words
}

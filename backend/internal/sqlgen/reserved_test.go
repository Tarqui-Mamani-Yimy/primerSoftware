package sqlgen

import "testing"

func TestTableName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Order", "jhi_order"},   // ORDER is PostgreSQL-reserved
		{"User", "jhi_user"},     // USER is PostgreSQL-reserved (naming rule only; JHipster
		{"Customer", "customer"}, // separately forbids a user-defined User entity outright)
		{"OrderLine", "order_line"},
		{"UMLClass", "umlclass"}, // hibernateSnakeCase quirk: an acronym run swallows the
		{"Order2", "order2"},     // capitalized word right after it; digits never split.
	}
	for _, tc := range cases {
		if got := tableName(tc.in); got != tc.want {
			t.Errorf("tableName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestColumnName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user", "jhi_user"},   // USER is PostgreSQL-reserved
		{"limit", "jhi_limit"}, // LIMIT is PostgreSQL-reserved
		{"date", "date"},       // DATE is NOT in JHipster's PostgreSQL reserved list
		{"label", "label"},
		{"address2", "address_2"}, // lodash snakeCase always splits letters from digits
		{"userID", "user_id"},     // lodash's words() peels a trailing acronym off
		{"whenAt", "when_at"},
	}
	for _, tc := range cases {
		if got := columnName(tc.in); got != tc.want {
			t.Errorf("columnName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRelationshipColumnBaseNeverPrefixed(t *testing.T) {
	// prepareRelationshipForDatabase never checks reserved keywords, so a
	// relationship literally named "order" still yields "order", not
	// "jhi_order" (the caller appends "_id").
	cases := []struct{ in, want string }{
		{"order", "order"},
		{"user", "user"},
		{"orderLine", "order_line"},
	}
	for _, tc := range cases {
		if got := relationshipColumnBase(tc.in); got != tc.want {
			t.Errorf("relationshipColumnBase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHibernateSnakeCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Order", "order"},
		{"OrderLine", "order_line"},
		{"UMLClass", "umlclass"},
		{"XMLHttpRequest", "xmlhttp_request"},
		{"userID", "userid"},
		{"Order2", "order2"},
		{"a", "a"},
	}
	for _, tc := range cases {
		if got := hibernateSnakeCase(tc.in); got != tc.want {
			t.Errorf("hibernateSnakeCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLodashSnakeCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"fooBar", "foo_bar"},
		{"UMLClass", "uml_class"},
		{"userID", "user_id"},
		{"address2", "address_2"},
		{"IDCard", "id_card"},
		{"whenAt", "when_at"},
		{"a", "a"},
	}
	for _, tc := range cases {
		if got := lodashSnakeCase(tc.in); got != tc.want {
			t.Errorf("lodashSnakeCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

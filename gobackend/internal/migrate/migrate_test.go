package migrate_test

import (
	"testing"

	"github.com/ai-uml-architect/gobackend/internal/migrate"
)

func TestMigrationsAreOrderedFlywayStyle(t *testing.T) {
	migs := migrate.Ordered()
	if len(migs) < 2 {
		t.Fatalf("expected at least V1+V2 migrations, got %d", len(migs))
	}
	if migs[0].Version != "1" || migs[1].Version != "2" {
		t.Fatalf("migrations must apply in Flyway version order, got %v", migs)
	}
	for i := 1; i < len(migs); i++ {
		if migs[i].Version <= migs[i-1].Version {
			t.Fatalf("migrations out of order: %v", migs)
		}
	}
}

func TestV1ReplicatesSchemaAuthority(t *testing.T) {
	migs := migrate.Ordered()
	v1 := migs[0].SQL
	for _, table := range []string{
		"CREATE TABLE users",
		"CREATE TABLE projects",
		"CREATE TABLE project_memberships",
		"CREATE TABLE diagrams",
		"CREATE TABLE diagram_versions",
		"CREATE TABLE refresh_tokens",
	} {
		if !contains(v1, table) {
			t.Errorf("V1 must replicate %q from the schema authority", table)
		}
	}
	if !contains(v1, "ana@example.com") || !contains(v1, "Password123!") {
		t.Errorf("V1 must replicate the development seed identities")
	}
}

func TestV2ReplicatesConditionalSeedSemantics(t *testing.T) {
	migs := migrate.Ordered()
	v2 := migs[1].SQL
	if !contains(v2, "updated_diagram") || !contains(v2, "version_number, document, created_by") {
		t.Errorf("V2 must replicate the conditional seed + version-2 insert semantics")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

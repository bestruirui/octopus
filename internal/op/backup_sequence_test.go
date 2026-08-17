package op

import "testing"

func TestPostgresSequenceResetStatement(t *testing.T) {
	const want = `SELECT setval(pg_get_serial_sequence('groups', 'id'), COALESCE(MAX("id"), 1), MAX("id") IS NOT NULL) FROM "groups"`
	if got := postgresSequenceResetStatement("groups"); got != want {
		t.Fatalf("postgresSequenceResetStatement() = %q, want %q", got, want)
	}
}

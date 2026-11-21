package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestCellValueUUID(t *testing.T) {
	var b [16]byte
	copy(b[:], []byte{
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
		0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef,
	})
	got := cellValue(pgtype.UUIDOID, b)
	want := pgtype.UUID{Bytes: b, Valid: true}.String()
	if got != want {
		t.Fatalf("uuid=%v want %s", got, want)
	}
}

func TestNumericTypeName(t *testing.T) {
	const numericOID = uint32(1700)
	// ((12 << 16) | 2) + 4
	if got := typeName(numericOID, (12<<16)|2+4); got != "numeric(12,2)" {
		t.Fatalf("numeric(12,2)=%q", got)
	}
	if got := typeName(numericOID, -1); got != "numeric" {
		t.Fatalf("plain numeric=%q", got)
	}
}

func TestCastSuffix(t *testing.T) {
	suffix, ok := castSuffix("numeric(12,2)")
	if !ok || suffix != "::numeric(12,2)" {
		t.Fatalf("suffix=%q ok=%v", suffix, ok)
	}
	if _, ok := castSuffix("timestamp with time zone"); !ok {
		t.Fatal("timestamptz should be safe")
	}
	if _, ok := castSuffix("int; drop table users"); ok {
		t.Fatal("injection should be rejected")
	}
}

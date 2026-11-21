package driver_test

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/mysql"
	"lotustab/internal/drivers/postgres"
)

func TestRegistryRejectsDuplicates(t *testing.T) {
	r := driver.NewRegistry()
	if err := r.Register(postgres.New()); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(postgres.New()); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
	if err := r.Register(mysql.New()); err != nil {
		t.Fatal(err)
	}
	if got := len(r.All()); got != 2 {
		t.Fatalf("got %d drivers, want 2", got)
	}
}

func TestNormalizeValue(t *testing.T) {
	if driver.NormalizeValue(nil) != nil {
		t.Fatal("nil should stay nil")
	}
	if driver.NormalizeValue([]byte("abc")) != "abc" {
		t.Fatal("bytes should become string")
	}
	if driver.NormalizeValue(int32(7)) != int64(7) {
		t.Fatal("int32 should become int64")
	}
	numeric := pgtype.Numeric{Int: big.NewInt(8036), Exp: -2, Valid: true}
	if got := driver.NormalizeValue(numeric); got != "80.36" {
		t.Fatalf("numeric=%v", got)
	}
	gotSlice := driver.NormalizeValue([]pgtype.Numeric{numeric})
	if !reflect.DeepEqual(gotSlice, []any{"80.36"}) {
		t.Fatalf("numeric slice=%#v", gotSlice)
	}
	stamp := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := driver.NormalizeValue(stamp); got != stamp.Format(time.RFC3339Nano) {
		t.Fatalf("time=%v", got)
	}

	settings := map[string]any{
		"locale":   "pt",
		"theme":    "dark",
		"timezone": "Europe/Moscow",
	}
	gotSettings := driver.NormalizeValue(settings)
	if _, ok := gotSettings.(string); ok {
		t.Fatalf("json object became string %v", gotSettings)
	}
	raw, err := json.Marshal(gotSettings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "map[") {
		t.Fatalf("json object dumped as map: %s", raw)
	}
	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["locale"] != "pt" || decoded["theme"] != "dark" || decoded["timezone"] != "Europe/Moscow" {
		t.Fatalf("decoded=%v", decoded)
	}
}

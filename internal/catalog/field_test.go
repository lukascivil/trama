package catalog_test

import (
	"testing"

	"github.com/lukascivil/trama/internal/catalog"
)

func TestValidateFieldPath(t *testing.T) {
	for _, path := range []string{"id", "customer.email", "items[].sku", "a.b.c", "user_id", "userId"} {
		if err := catalog.ValidateField(path, "string"); err != nil {
			t.Errorf("ValidateField(%q) = %v, want nil", path, err)
		}
	}
	for _, path := range []string{"", ".email", "email.", "a..b", "1id", "items[].", "tags[]", "a b", "customer.*"} {
		if err := catalog.ValidateField(path, "string"); err == nil {
			t.Errorf("ValidateField(%q) = nil, want an error", path)
		}
	}
}

func TestValidateFieldType(t *testing.T) {
	for _, typ := range catalog.FieldTypes {
		if err := catalog.ValidateField("id", typ); err != nil {
			t.Errorf("type %q = %v, want nil", typ, err)
		}
	}
	for _, typ := range []string{"str", "String", "string?", "[]", "[]foo"} {
		if err := catalog.ValidateField("id", typ); err == nil {
			t.Errorf("type %q = nil, want an error", typ)
		}
	}
}

func TestValidateFieldSkipsEmptyType(t *testing.T) {
	if err := catalog.ValidateField("customer.email", ""); err != nil {
		t.Fatalf("ValidateField with no type = %v, want nil", err)
	}
}

func TestValidateTable(t *testing.T) {
	for _, name := range []string{"users", "public.users", "user_roles"} {
		if err := catalog.ValidateTable(name); err != nil {
			t.Errorf("ValidateTable(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"", "a.b.c", "1users", "users;", ".users"} {
		if err := catalog.ValidateTable(name); err == nil {
			t.Errorf("ValidateTable(%q) = nil, want an error", name)
		}
	}
}

func TestValidateColumn(t *testing.T) {
	for _, name := range []string{"id", "email", "created_at"} {
		if err := catalog.ValidateColumn(name); err != nil {
			t.Errorf("ValidateColumn(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"", "a.b", "1a", "a b"} {
		if err := catalog.ValidateColumn(name); err == nil {
			t.Errorf("ValidateColumn(%q) = nil, want an error", name)
		}
	}
}

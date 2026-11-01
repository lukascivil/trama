package catalog

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// FieldTypes are the types a request or response field may have. The list is
// closed on purpose: a type outside it is one nothing can search on.
var FieldTypes = []string{"string", "number", "boolean", "object", "[]string", "[]number", "[]boolean", "[]object"}

var fieldSegment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(\[\])?$`)

// ValidateField checks one request or response field: the dotted path always,
// and the type only when typ is given, so a consumer's fields_used can reuse
// the path rule without naming a type.
func ValidateField(path, typ string) error {
	segments := strings.Split(path, ".")
	for i, seg := range segments {
		if !fieldSegment.MatchString(seg) {
			return fmt.Errorf("field path %q must be dotted names, e.g. customer.email or items[].sku", path)
		}
		if i == len(segments)-1 && strings.HasSuffix(seg, "[]") {
			return fmt.Errorf("field path %q ends in []: a list of values is the type (e.g. []string), not the path", path)
		}
	}
	if typ != "" && !slices.Contains(FieldTypes, typ) {
		return fmt.Errorf("field %q: type %q is not one of %s", path, typ, strings.Join(FieldTypes, ", "))
	}
	return nil
}

var ident = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateColumn checks a column name: a plain name, no dots.
func ValidateColumn(name string) error {
	if !ident.MatchString(name) {
		return fmt.Errorf("column %q must be a plain name, e.g. created_at", name)
	}
	return nil
}

// ValidateTable checks a table name: a plain name, or one qualified by its
// schema, as in public.users.
func ValidateTable(name string) error {
	parts := strings.Split(name, ".")
	if len(parts) > 2 {
		return fmt.Errorf("table %q must be a name, or schema.name", name)
	}
	for _, part := range parts {
		if !ident.MatchString(part) {
			return fmt.Errorf("table %q must be a name, or schema.name", name)
		}
	}
	return nil
}

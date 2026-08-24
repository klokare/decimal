package decimal

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Value implements [driver.Valuer].
//
// The value is sent as a string so that no precision is lost on the way to a
// NUMERIC or DECIMAL column. Sending a float64 would defeat the purpose of the
// type, and few drivers accept a 96-bit integer.
func (d Decimal) Value() (driver.Value, error) { return d.String(), nil }

// Scan implements [sql.Scanner]. It accepts the types database drivers produce
// for numeric columns: string, []byte, the signed and unsigned integers, and
// the floats. A NULL is an error here; use [NullDecimal] for a nullable column.
func (d *Decimal) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		return wrapf(ErrSyntax, "cannot scan NULL into a Decimal; use NullDecimal")
	case string:
		return d.scanText(v)
	case []byte:
		return d.scanText(string(v))
	case int64:
		*d = FromInt(v)
		return nil
	case int32:
		*d = FromInt(v)
		return nil
	case int:
		*d = FromInt(v)
		return nil
	case uint64:
		*d = FromUint(v)
		return nil
	case uint32:
		*d = FromUint(v)
		return nil
	case float64:
		v2, err := FromFloat64(v)
		if err != nil {
			return err
		}
		*d = v2
		return nil
	case float32:
		v2, err := FromFloat32(v)
		if err != nil {
			return err
		}
		*d = v2
		return nil
	case bool, time.Time:
		return wrapf(ErrSyntax, "cannot scan %T into a Decimal", src)
	default:
		return wrapf(ErrSyntax, "cannot scan %T into a Decimal", src)
	}
}

// scanText parses a value that arrived as text. Drivers vary in what they emit
// for a NUMERIC column, so scientific notation is accepted alongside the plain
// form.
func (d *Decimal) scanText(s string) error {
	v, err := ParseStyle(s, StyleFloat|AllowThousands, Invariant)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// NullDecimal is a Decimal that may be NULL, for use with [database/sql].
//
// It also marshals to and from JSON null, which the embedded Decimal alone
// would not do.
type NullDecimal struct {
	Decimal Decimal
	Valid   bool
}

// NewNullDecimal returns a valid NullDecimal holding d.
func NewNullDecimal(d Decimal) NullDecimal { return NullDecimal{Decimal: d, Valid: true} }

// Scan implements [sql.Scanner], mapping NULL to an invalid NullDecimal.
func (n *NullDecimal) Scan(src any) error {
	if src == nil {
		n.Decimal, n.Valid = Decimal{}, false
		return nil
	}
	if err := n.Decimal.Scan(src); err != nil {
		n.Valid = false
		return err
	}
	n.Valid = true
	return nil
}

// Value implements [driver.Valuer], mapping an invalid NullDecimal to NULL.
func (n NullDecimal) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Decimal.Value()
}

// MarshalJSON implements [json.Marshaler], writing null when not valid and
// otherwise the quoted string form used by [Decimal.MarshalJSON].
func (n NullDecimal) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Decimal.MarshalJSON()
}

// UnmarshalJSON implements [json.Unmarshaler], reading null as not valid.
func (n *NullDecimal) UnmarshalJSON(text []byte) error {
	if string(trimSpace(text)) == "null" {
		n.Decimal, n.Valid = Decimal{}, false
		return nil
	}
	if err := n.Decimal.UnmarshalJSON(text); err != nil {
		n.Valid = false
		return err
	}
	n.Valid = true
	return nil
}

// String returns the decimal's text, or "NULL" when not valid.
func (n NullDecimal) String() string {
	if !n.Valid {
		return "NULL"
	}
	return n.Decimal.String()
}

// trimSpace drops leading and trailing JSON whitespace.
func trimSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && isWhite(b[i]) {
		i++
	}
	for j > i && isWhite(b[j-1]) {
		j--
	}
	return b[i:j]
}

var (
	_ fmt.Stringer  = NullDecimal{}
	_ driver.Valuer = NullDecimal{}
)

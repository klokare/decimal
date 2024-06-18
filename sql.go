package decimal

import (
	"database/sql/driver"
	"fmt"
)

// NullDecimal ...
type NullDecimal struct {
	Decimal
	Valid bool
}

// Scan implements the Scanner interface
func (d *NullDecimal) Scan(value interface{}) (err error) {
	if value == nil {
		d.Decimal, d.Valid = Zero, false
		return nil
	}
	d.Decimal, err = scan(value)
	d.Valid = err == nil
	return err
}

// Value provides a string value to the database.
func (d NullDecimal) Value() (driver.Value, error) {
	if !d.Valid {
		return nil, nil
	}
	return d.String(), nil
}

// Scan assigns value from a database driver.
func scan(src interface{}) (d Decimal, err error) {
	switch t := src.(type) {
	case int32:
		d = NewFromInt32(src.(int32))
	case int64:
		d = NewFromInt64(src.(int64))
	case float32:
		d = NewFromFloat32(src.(float32))
	case float64:
		d = NewFromFloat64(src.(float64))
	case string:
		d, err = Parse(src.(string))
	case []byte:
		d, err = Parse(string(src.([]byte)))
	default:
		err = fmt.Errorf("cannot create decimal from %v", t)
	}
	return d, err
}

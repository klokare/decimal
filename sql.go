package decimal

import "database/sql/driver"

// NullDecimal ...
type NullDecimal struct {
	Decimal
	Valid bool
}

// Scan implements the Scanner interface
func (d *NullDecimal) Scan(value interface{}) error {
	if value == nil {
		d.Decimal, d.Valid = Zero, false
		return nil
	}
	err := (&d.Decimal).Scan(value)
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

package decimal

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

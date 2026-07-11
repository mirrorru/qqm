package fixtures

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

type SubjBirthday struct {
	Date Date `tbl:"col=birthday"`
}

type Date time.Time

var (
	_ driver.Valuer = Date{}
	_ sql.Scanner   = (*Date)(nil)
)

func (d Date) String() string { return time.Time(d).String() }

func (d *Date) Parse(in string) error {
	t, err := time.Parse(time.DateOnly, in)
	*d = Date(t)
	return err
}

func (d Date) Value() (driver.Value, error) {
	return time.Time(d), nil
}

func (d *Date) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time:
		*d = Date(v)
	case *time.Time:
		if v != nil {
			*d = Date(*v)
		}
	case string:
		t, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return err
		}
		*d = Date(t)
	case nil:
		*d = Date{}
	default:
		return fmt.Errorf("не удалось преобразовать %T в Date", src)
	}
	return nil
}

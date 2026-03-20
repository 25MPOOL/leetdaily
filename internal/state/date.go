package state

import (
	"encoding/json"
	"fmt"
	"time"
)

const dateLayout = "2006-01-02"

type Date struct {
	time.Time
}

func ParseDate(raw string) (Date, error) {
	parsed, err := time.Parse(dateLayout, raw)
	if err != nil {
		return Date{}, fmt.Errorf("parse date %q: %w", raw, err)
	}

	return Date{Time: parsed.UTC()}, nil
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(dateLayout))
}

func (d *Date) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parsed, err := ParseDate(raw)
	if err != nil {
		return err
	}

	*d = parsed
	return nil
}

func (d Date) String() string {
	return d.Format(dateLayout)
}

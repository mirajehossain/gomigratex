package migrator

import "time"

type Row struct {
	Version        string
	Name           string
	Checksum       string
	AppliedAt      time.Time
	AppliedBy      string
	DurationMS     int64
	Status         string // success | failed
	ExecutionOrder int64
}

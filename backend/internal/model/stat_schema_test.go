package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// The deployed SQL schema uses singular stat_* names. Check GORM's resolved
// mapping rather than just calling TableName, so worker inserts stay compatible.
func TestStatisticsSchemaMapping(t *testing.T) {
	for _, tc := range []struct {
		model any
		table string
	}{
		{&StatDaily{}, "stat_daily"},
		{&StatHourly{}, "stat_hourly"},
		{&StatDevice{}, "stat_device"},
		{&StatGeo{}, "stat_geo"},
	} {
		t.Run(tc.table, func(t *testing.T) {
			parsed, err := schema.Parse(tc.model, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Table != tc.table {
				t.Fatalf("GORM table = %s, want %s", parsed.Table, tc.table)
			}
		})
	}
}

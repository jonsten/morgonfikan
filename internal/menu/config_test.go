package menu

import (
	"strings"
	"testing"
	"time"
)

const scheduleYAML = `schedules:
  - valid_from: 2026-11-09
    anchor_date: 2026-11-09
    weekdays:
      monday: [monday]
      tuesday: [tuesday]
      wednesday: [wednesday]
      thursday: [cucumber, cucumber, marmalade, cucumber]
      friday: [marmalade, pate, ham, pate]
`

func testConfig(t *testing.T, schedules, closures string) *Config {
	t.Helper()
	config, err := Parse([]byte(schedules), []byte(closures))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return config
}

func testDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(dateLayout, value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

func TestDayCyclesEachWeekdayIndependently(t *testing.T) {
	config := testConfig(t, scheduleYAML, "closures: []\n")
	tests := []struct {
		date string
		want string
	}{
		{"2026-11-12", "cucumber"},
		{"2026-11-19", "cucumber"},
		{"2026-11-26", "marmalade"},
		{"2026-12-03", "cucumber"},
		{"2026-11-13", "marmalade"},
		{"2026-11-20", "pate"},
		{"2026-11-27", "ham"},
		{"2026-12-04", "pate"},
		{"2026-12-11", "marmalade"},
	}
	for _, test := range tests {
		t.Run(test.date, func(t *testing.T) {
			got, err := config.Day(testDate(t, test.date))
			if err != nil {
				t.Fatal(err)
			}
			if !got.Served || len(got.Items) != 1 || got.Items[0] != test.want {
				t.Fatalf("Day() = %+v, want served menu %q", got, test.want)
			}
		})
	}
}

func TestClosureDoesNotShiftRotation(t *testing.T) {
	config := testConfig(t, scheduleYAML, "closures:\n  - date: 2026-11-13\n    note: Closed\n")
	closed, err := config.Day(testDate(t, "2026-11-13"))
	if err != nil {
		t.Fatal(err)
	}
	if closed.Served || len(closed.Items) != 0 || closed.Note != "Closed" {
		t.Fatalf("closed day = %+v", closed)
	}

	next, err := config.Day(testDate(t, "2026-11-20"))
	if err != nil {
		t.Fatal(err)
	}
	if next.Items[0] != "pate" {
		t.Fatalf("next Friday menu = %v, want pate", next.Items)
	}
}

func TestScheduleVersionsUseLatestValidFrom(t *testing.T) {
	schedules := scheduleYAML + `  - valid_from: 2026-11-23
    anchor_date: 2026-11-23
    weekdays:
      monday: [new monday]
      tuesday: [new tuesday]
      wednesday: [new wednesday]
      thursday: [new thursday]
      friday: [new friday]
`
	config := testConfig(t, schedules, "closures: []\n")
	got, err := config.Day(testDate(t, "2026-11-23"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Items[0] != "new monday" {
		t.Fatalf("menu = %v, want new monday", got.Items)
	}
}

func TestParseRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name      string
		schedules string
		closures  string
		wantError string
	}{
		{"unknown schedule field", strings.Replace(scheduleYAML, "anchor_date:", "anchored_at:", 1), "closures: []\n", "field anchored_at not found"},
		{"non-Monday anchor", strings.Replace(scheduleYAML, "anchor_date: 2026-11-09", "anchor_date: 2026-11-10", 1), "closures: []\n", "anchor_date must be a Monday"},
		{"empty weekday", strings.Replace(scheduleYAML, "monday: [monday]", "monday: []", 1), "closures: []\n", "monday must have at least one menu entry"},
		{"duplicate valid_from", scheduleYAML + scheduleYAML[strings.Index(scheduleYAML, "  - valid_from:"):], "closures: []\n", "valid_from dates must be unique"},
		{"duplicate closure", scheduleYAML, "closures:\n  - date: 2026-12-25\n  - date: 2026-12-25\n", "closure dates must be unique"},
		{"invalid closure date", scheduleYAML, "closures:\n  - date: 2026-02-30\n", "must use YYYY-MM-DD format"},
		{"missing closure list", scheduleYAML, "{}\n", "closure config must contain a closures list"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.schedules), []byte(test.closures))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Parse() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestWeekStartAndWeekdays(t *testing.T) {
	tests := []struct {
		date string
		want string
	}{
		{"2026-11-11", "2026-11-11"},
		{"2026-11-14", "2026-11-16"},
		{"2026-11-15", "2026-11-16"},
	}
	for _, test := range tests {
		start := WeekStart(testDate(t, test.date))
		if got := start.Format(dateLayout); got != test.want {
			t.Errorf("WeekStart(%s) = %s, want %s", test.date, got, test.want)
		}
		days := WeekdaysFrom(start)
		if len(days) != 5 {
			t.Fatalf("WeekdaysFrom(%s) has %d dates, want 5", test.date, len(days))
		}
		for _, day := range days {
			if !IsWeekday(day) {
				t.Errorf("WeekdaysFrom(%s) contains weekend date %s", test.date, day.Format(dateLayout))
			}
		}
	}
}

func TestDayReturnsErrorBeforeFirstSchedule(t *testing.T) {
	config := testConfig(t, scheduleYAML, "closures: []\n")
	_, err := config.Day(testDate(t, "2026-11-06"))
	if err == nil {
		t.Fatal("Day() error = nil, want no-schedule error")
	}
}

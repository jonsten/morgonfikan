package menu

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const dateLayout = "2006-01-02"

var weekdays = []time.Weekday{
	time.Monday,
	time.Tuesday,
	time.Wednesday,
	time.Thursday,
	time.Friday,
}

var weekdayNames = map[string]time.Weekday{
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
}

type ScheduleFile struct {
	Schedules []Schedule `yaml:"schedules"`
}

type Schedule struct {
	ValidFrom  string              `yaml:"valid_from"`
	AnchorDate string              `yaml:"anchor_date"`
	Weekdays   map[string][]string `yaml:"weekdays"`

	validFrom  time.Time
	anchorDate time.Time
	menus      map[time.Weekday][]string
}

type ClosureFile struct {
	Closures []Closure `yaml:"closures"`
}

type Closure struct {
	Date string `yaml:"date"`
	Note string `yaml:"note,omitempty"`
}

type Config struct {
	schedules []Schedule
	closures  map[string]string
}

type Day struct {
	Date   string   `json:"date"`
	Served bool     `json:"served"`
	Items  []string `json:"items"`
	Note   string   `json:"note,omitempty"`
}

func Parse(scheduleData, closureData []byte) (*Config, error) {
	var scheduleFile ScheduleFile
	if err := decodeYAML(scheduleData, &scheduleFile); err != nil {
		return nil, fmt.Errorf("decode schedule config: %w", err)
	}
	if len(scheduleFile.Schedules) == 0 {
		return nil, fmt.Errorf("schedule config must contain at least one schedule")
	}

	seenValidFrom := make(map[string]struct{}, len(scheduleFile.Schedules))
	for i := range scheduleFile.Schedules {
		schedule := &scheduleFile.Schedules[i]
		if err := schedule.validate(); err != nil {
			return nil, fmt.Errorf("schedule %d: %w", i+1, err)
		}
		if _, exists := seenValidFrom[schedule.ValidFrom]; exists {
			return nil, fmt.Errorf("schedule valid_from dates must be unique: %s", schedule.ValidFrom)
		}
		seenValidFrom[schedule.ValidFrom] = struct{}{}
	}
	sort.Slice(scheduleFile.Schedules, func(i, j int) bool {
		return scheduleFile.Schedules[i].validFrom.Before(scheduleFile.Schedules[j].validFrom)
	})

	var closureFile ClosureFile
	if err := decodeYAML(closureData, &closureFile); err != nil {
		return nil, fmt.Errorf("decode closure config: %w", err)
	}
	if closureFile.Closures == nil {
		return nil, fmt.Errorf("closure config must contain a closures list")
	}
	closures := make(map[string]string, len(closureFile.Closures))
	for i := range closureFile.Closures {
		closure := &closureFile.Closures[i]
		if _, err := parseDate(closure.Date); err != nil {
			return nil, fmt.Errorf("closure %d: %w", i+1, err)
		}
		if _, exists := closures[closure.Date]; exists {
			return nil, fmt.Errorf("closure dates must be unique: %s", closure.Date)
		}
		closures[closure.Date] = strings.TrimSpace(closure.Note)
	}

	return &Config{
		schedules: scheduleFile.Schedules,
		closures:  closures,
	}, nil
}

func (s *Schedule) validate() error {
	var err error
	if s.validFrom, err = parseDate(s.ValidFrom); err != nil {
		return fmt.Errorf("valid_from: %w", err)
	}
	if s.anchorDate, err = parseDate(s.AnchorDate); err != nil {
		return fmt.Errorf("anchor_date: %w", err)
	}
	if s.anchorDate.Weekday() != time.Monday {
		return fmt.Errorf("anchor_date must be a Monday")
	}
	if s.anchorDate.After(s.validFrom) {
		return fmt.Errorf("anchor_date must not be after valid_from")
	}
	if len(s.Weekdays) != len(weekdays) {
		return fmt.Errorf("weekdays must define exactly monday through friday")
	}

	s.menus = make(map[time.Weekday][]string, len(weekdays))
	for name, entries := range s.Weekdays {
		day, ok := weekdayNames[name]
		if !ok {
			return fmt.Errorf("unknown weekday %q", name)
		}
		if len(entries) == 0 {
			return fmt.Errorf("%s must have at least one menu entry", name)
		}
		for i, entry := range entries {
			if strings.TrimSpace(entry) == "" {
				return fmt.Errorf("%s menu entry %d must not be empty", name, i+1)
			}
			entries[i] = strings.TrimSpace(entry)
		}
		s.menus[day] = entries
	}
	for _, day := range weekdays {
		if _, exists := s.menus[day]; !exists {
			return fmt.Errorf("missing weekday %q", strings.ToLower(day.String()))
		}
	}
	return nil
}

func decodeYAML(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple YAML documents are not allowed")
		}
		return err
	}
	return nil
}

func parseDate(value string) (time.Time, error) {
	date, err := time.Parse(dateLayout, value)
	if err != nil || date.Format(dateLayout) != value {
		return time.Time{}, fmt.Errorf("date %q must use YYYY-MM-DD format", value)
	}
	return date, nil
}

func (c *Config) Day(date time.Time) (Day, error) {
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dateString := date.Format(dateLayout)
	if note, closed := c.closures[dateString]; closed {
		return Day{Date: dateString, Items: []string{}, Note: note}, nil
	}
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return Day{Date: dateString, Items: []string{}}, nil
	}

	schedule, ok := c.scheduleFor(date)
	if !ok {
		return Day{}, fmt.Errorf("no schedule is valid on %s", dateString)
	}
	entries := schedule.menus[date.Weekday()]
	anchorWeek := mondayOf(schedule.anchorDate)
	dateWeek := mondayOf(date)
	weeks := int(dateWeek.Sub(anchorWeek) / (7 * 24 * time.Hour))
	index := weeks % len(entries)
	return Day{
		Date:   dateString,
		Served: true,
		Items:  []string{entries[index]},
	}, nil
}

func (c *Config) scheduleFor(date time.Time) (Schedule, bool) {
	index := sort.Search(len(c.schedules), func(i int) bool {
		return c.schedules[i].validFrom.After(date)
	})
	if index == 0 {
		return Schedule{}, false
	}
	return c.schedules[index-1], true
}

func mondayOf(date time.Time) time.Time {
	offset := (int(date.Weekday()) + 6) % 7
	return date.AddDate(0, 0, -offset)
}

func WeekStart(now time.Time) time.Time {
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !IsWeekday(date) {
		return mondayOf(date).AddDate(0, 0, 7)
	}
	return date
}

func WeekdaysFrom(start time.Time) []time.Time {
	days := make([]time.Time, 0, 5)
	for date := start; len(days) < 5; date = date.AddDate(0, 0, 1) {
		if IsWeekday(date) {
			days = append(days, date)
		}
	}
	return days
}

func IsWeekday(date time.Time) bool {
	return date.Weekday() >= time.Monday && date.Weekday() <= time.Friday
}

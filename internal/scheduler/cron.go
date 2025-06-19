package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronSchedule represents a parsed cron schedule (minute, hour, day, month, weekday)
type CronSchedule struct {
	Minute  map[int]bool // 0-59
	Hour    map[int]bool // 0-23
	Day     map[int]bool // 1-31
	Month   map[int]bool // 1-12
	Weekday map[int]bool // 0-6 (Sunday=0)
}

// ParseCron parses a 5-field cron expression into a CronSchedule
func ParseCron(expr string) (*CronSchedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("invalid cron expression: expected 5 fields, got %d", len(fields))
	}
	minute, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("minute: %w", err)
	}
	hour, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("hour: %w", err)
	}
	day, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("day: %w", err)
	}
	month, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	weekday, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return nil, fmt.Errorf("weekday: %w", err)
	}
	return &CronSchedule{
		Minute:  minute,
		Hour:    hour,
		Day:     day,
		Month:   month,
		Weekday: weekday,
	}, nil
}

// parseCronField parses a single cron field (supports *, single values, lists, and ranges)
func parseCronField(field string, min, max int) (map[int]bool, error) {
	result := make(map[int]bool)
	if field == "*" {
		for i := min; i <= max; i++ {
			result[i] = true
		}
		return result, nil
	}
	parts := strings.Split(field, ",")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			start, err1 := strconv.Atoi(rangeParts[0])
			end, err2 := strconv.Atoi(rangeParts[1])
			if err1 != nil || err2 != nil || start > end || start < min || end > max {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			for i := start; i <= end; i++ {
				result[i] = true
			}
		} else {
			val, err := strconv.Atoi(part)
			if err != nil || val < min || val > max {
				return nil, fmt.Errorf("invalid value: %s", part)
			}
			result[val] = true
		}
	}
	return result, nil
}

// Next returns the next time after 'after' that matches the schedule
func (c *CronSchedule) Next(after time.Time) time.Time {
	// Brute-force: increment minute by minute until all fields match
	t := after.Add(time.Minute).Truncate(time.Minute)
	for {
		if c.Minute[t.Minute()] &&
			c.Hour[t.Hour()] &&
			c.Day[t.Day()] &&
			c.Month[int(t.Month())] &&
			c.Weekday[int(t.Weekday())] {
			return t
		}
		t = t.Add(time.Minute)
	}
} 
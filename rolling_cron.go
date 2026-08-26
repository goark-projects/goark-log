package goarklog

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const maxCronSearchSeconds = 5 * 366 * 24 * 60 * 60

type cronSchedule struct {
	seconds  cronField
	minutes  cronField
	hours    cronField
	days     cronField
	months   cronField
	weekdays cronField
}

type cronField struct {
	allowed []bool
}

func parseCronSchedule(expression string) (*cronSchedule, error) {
	fields := strings.Fields(strings.TrimSpace(expression))
	switch len(fields) {
	case 5:
		fields = append([]string{"0"}, fields...)
	case 6:
	case 7:
		if fields[6] != "*" && fields[6] != "?" {
			return nil, fmt.Errorf("year field is unsupported")
		}
		fields = fields[:6]
	default:
		return nil, fmt.Errorf("expected 5, 6, or 7 fields")
	}
	seconds, err := parseCronField(fields[0], 0, 59, nil, false)
	if err != nil {
		return nil, fmt.Errorf("seconds: %w", err)
	}
	minutes, err := parseCronField(fields[1], 0, 59, nil, false)
	if err != nil {
		return nil, fmt.Errorf("minutes: %w", err)
	}
	hours, err := parseCronField(fields[2], 0, 23, nil, false)
	if err != nil {
		return nil, fmt.Errorf("hours: %w", err)
	}
	days, err := parseCronField(fields[3], 1, 31, nil, true)
	if err != nil {
		return nil, fmt.Errorf("day-of-month: %w", err)
	}
	months, err := parseCronField(fields[4], 1, 12, monthNames(), false)
	if err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	weekdays, err := parseCronField(fields[5], 0, 7, weekdayNames(), true)
	if err != nil {
		return nil, fmt.Errorf("day-of-week: %w", err)
	}
	return &cronSchedule{
		seconds:  seconds,
		minutes:  minutes,
		hours:    hours,
		days:     days,
		months:   months,
		weekdays: weekdays,
	}, nil
}

func parseCronField(value string, min int, max int, names map[string]int, allowQuestion bool) (cronField, error) {
	field := cronField{allowed: make([]bool, max+1)}
	parts := strings.Split(strings.TrimSpace(value), ",")
	for _, part := range parts {
		if err := applyCronFieldPart(field.allowed, strings.TrimSpace(part), min, max, names, allowQuestion); err != nil {
			return cronField{}, err
		}
	}
	return field, nil
}

func applyCronFieldPart(allowed []bool, part string, min int, max int, names map[string]int, allowQuestion bool) error {
	if part == "" {
		return fmt.Errorf("empty field")
	}
	base := part
	step := 1
	if before, after, ok := strings.Cut(part, "/"); ok {
		base = strings.TrimSpace(before)
		parsed, err := strconv.Atoi(strings.TrimSpace(after))
		if err != nil || parsed <= 0 {
			return fmt.Errorf("invalid step %q", after)
		}
		step = parsed
	}
	start, end, err := cronPartRange(base, min, max, names, allowQuestion, step > 1)
	if err != nil {
		return err
	}
	for value := start; value <= end; value += step {
		allowed[value] = true
	}
	return nil
}

func cronPartRange(part string, min int, max int, names map[string]int, allowQuestion bool, stepped bool) (int, int, error) {
	if part == "*" || allowQuestion && part == "?" {
		return min, max, nil
	}
	if before, after, ok := strings.Cut(part, "-"); ok {
		start, err := parseCronNumber(before, min, max, names)
		if err != nil {
			return 0, 0, err
		}
		end, err := parseCronNumber(after, min, max, names)
		if err != nil {
			return 0, 0, err
		}
		if start > end {
			return 0, 0, fmt.Errorf("range %q is descending", part)
		}
		return start, end, nil
	}
	start, err := parseCronNumber(part, min, max, names)
	if err != nil {
		return 0, 0, err
	}
	if stepped {
		return start, max, nil
	}
	return start, start, nil
}

func parseCronNumber(value string, min int, max int, names map[string]int) (int, error) {
	text := strings.ToUpper(strings.TrimSpace(value))
	if mapped, ok := names[text]; ok {
		return mapped, nil
	}
	parsed, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", value)
	}
	if parsed < min || parsed > max {
		return 0, fmt.Errorf("value %d out of range [%d,%d]", parsed, min, max)
	}
	return parsed, nil
}

func (s *cronSchedule) next(after time.Time) (time.Time, bool) {
	if s == nil {
		return time.Time{}, false
	}
	candidate := after.Truncate(time.Second).Add(time.Second)
	for scanned := 0; scanned < maxCronSearchSeconds; scanned++ {
		if s.matches(candidate) {
			return candidate, true
		}
		candidate = candidate.Add(time.Second)
	}
	return time.Time{}, false
}

func (s *cronSchedule) matches(value time.Time) bool {
	weekday := int(value.Weekday())
	return s.seconds.match(value.Second()) &&
		s.minutes.match(value.Minute()) &&
		s.hours.match(value.Hour()) &&
		s.days.match(value.Day()) &&
		s.months.match(int(value.Month())) &&
		(s.weekdays.match(weekday) || weekday == 0 && s.weekdays.match(7))
}

func (f cronField) match(value int) bool {
	return value >= 0 && value < len(f.allowed) && f.allowed[value]
}

func monthNames() map[string]int {
	return map[string]int{
		"JAN": 1,
		"FEB": 2,
		"MAR": 3,
		"APR": 4,
		"MAY": 5,
		"JUN": 6,
		"JUL": 7,
		"AUG": 8,
		"SEP": 9,
		"OCT": 10,
		"NOV": 11,
		"DEC": 12,
	}
}

func weekdayNames() map[string]int {
	return map[string]int{
		"SUN": 0,
		"MON": 1,
		"TUE": 2,
		"WED": 3,
		"THU": 4,
		"FRI": 5,
		"SAT": 6,
	}
}

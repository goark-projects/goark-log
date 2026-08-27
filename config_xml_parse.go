package goarklog

import (
	"fmt"
	"strconv"
	"strings"
)

func parseXMLInt(value string, field string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s is invalid", field)
	}
	return parsed, nil
}

func parseXMLIntPointer(value string) *int {
	parsed, err := parseXMLInt(value, "")
	if err != nil || parsed == 0 {
		return nil
	}
	return &parsed
}

func parseXMLIntValue(value string) int {
	parsed, err := parseXMLInt(value, "")
	if err != nil {
		return 0
	}
	return parsed
}

func parseXMLBool(value string, field string) (bool, error) {
	if strings.TrimSpace(value) == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(strings.ToLower(strings.TrimSpace(value)))
	if err != nil {
		return false, fmt.Errorf("%s is invalid", field)
	}
	return parsed, nil
}

func parseXMLBoolPointer(value string) *bool {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := parseXMLBool(value, "")
	if err != nil {
		return nil
	}
	return &parsed
}

func parseXMLBoolPointerStrict(value string, field string) (*bool, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := parseXMLBool(value, field)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseXMLBoolValue(value string) bool {
	parsed, err := parseXMLBool(value, "")
	if err != nil {
		return false
	}
	return parsed
}

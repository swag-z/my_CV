package validation

import (
"fmt"
"time"
)

// ValidateDate validates a date string in YYYY-MM-DD format
func ValidateDate(dateStr string, required bool) (bool, error) {
if dateStr == "" {
if !required {
return true, nil
}
return false, fmt.Errorf("date is required")
}

_, err := time.Parse("2006-01-02", dateStr)
if err != nil {
return false, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
}

return true, nil
}

// ParseDate parses a date string to time.Time
func ParseDate(dateStr string) (time.Time, error) {
if dateStr == "" {
return time.Time{}, nil
}
return time.Parse("2006-01-02", dateStr)
}

// ValidateDateNotFuture checks if date is not in the future
func ValidateDateNotFuture(dateStr string) error {
if dateStr == "" {
return nil
}

date, err := time.Parse("2006-01-02", dateStr)
if err != nil {
return err
}

if date.After(time.Now()) {
return fmt.Errorf("date cannot be in the future")
}

return nil
}

// ValidateDateRange checks if loading date is before unloading date
func ValidateDateRange(loadingDate, unloadingDate string) error {
if loadingDate == "" || unloadingDate == "" {
return nil
}

load, err1 := time.Parse("2006-01-02", loadingDate)
unload, err2 := time.Parse("2006-01-02", unloadingDate)

if err1 != nil || err2 != nil {
return nil
}

if load.After(unload) {
return fmt.Errorf("loading date cannot be after unloading date")
}

return nil
}

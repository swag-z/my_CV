package validation

import (
"fmt"
"strconv"

"github.com/shopspring/decimal"
)

// ValidateAmount validates a monetary amount string
func ValidateAmount(amountStr string, required bool) (bool, error) {
if amountStr == "" {
if !required {
return true, nil
}
return false, fmt.Errorf("amount is required")
}

amount, err := decimal.NewFromString(amountStr)
if err != nil {
return false, fmt.Errorf("invalid amount format: %w", err)
}

if amount.IsNegative() {
return false, fmt.Errorf("amount cannot be negative")
}

return true, nil
}

// ValidateAmountsConsistency checks if amounts are consistent
// amount_without_vat + vat_amount should equal total_with_vat (with 1.00 tolerance)
func ValidateAmountsConsistency(withoutVAT, vat, totalWithVAT string) []string {
var errors []string

if withoutVAT == "" || vat == "" || totalWithVAT == "" {
return errors
}

wov, err1 := decimal.NewFromString(withoutVAT)
v, err2 := decimal.NewFromString(vat)
tot, err3 := decimal.NewFromString(totalWithVAT)

if err1 != nil || err2 != nil || err3 != nil {
return errors
}

expectedTotal := wov.Add(v)
diff := expectedTotal.Sub(tot).Abs()

// Allow 1.00 tolerance
if diff.GreaterThan(decimal.NewFromInt(1)) {
errors = append(errors, fmt.Sprintf("Sum mismatch: %s + %s != %s", withoutVAT, vat, totalWithVAT))
}

return errors
}

// ParseAmount parses an amount string to decimal
func ParseAmount(amountStr string) (decimal.Decimal, error) {
if amountStr == "" {
return decimal.Zero, nil
}
return decimal.NewFromString(amountStr)
}

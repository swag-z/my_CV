package validation

import (
"strconv"
"strings"
)

// ValidateINN validates a Russian INN (tax ID)
// Returns true if valid, false otherwise
func ValidateINN(inn string) bool {
if inn == "" {
return false
}

// Remove spaces and dashes
inn = strings.ReplaceAll(inn, " ", "")
inn = strings.ReplaceAll(inn, "-", "")

// Check length: 10 for organizations, 12 for individuals
length := len(inn)
if length != 10 && length != 12 {
return false
}

// Check all characters are digits
for _, c := range inn {
if c < '0' || c > '9' {
return false
}
}

// Calculate control sum
var expectedControl int
var sum int

if length == 10 {
weights := []int{2, 4, 10, 3, 5, 9, 4, 6, 8}
for i := 0; i < 9; i++ {
digit, _ := strconv.Atoi(string(inn[i]))
sum += digit * weights[i]
}
expectedControl = sum % 11
if expectedControl >= 10 {
expectedControl = 0
}

actualControl, _ := strconv.Atoi(string(inn[9]))
return expectedControl == actualControl
} else if length == 12 {
weights1 := []int{7, 2, 4, 10, 3, 5, 9, 4, 6, 8}
for i := 0; i < 10; i++ {
digit, _ := strconv.Atoi(string(inn[i]))
sum += digit * weights1[i]
}
control1 := sum % 11
if control1 >= 10 {
control1 = 0
}

actualControl1, _ := strconv.Atoi(string(inn[10]))
if control1 != actualControl1 {
return false
}

sum = 0
weights2 := []int{3, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8}
for i := 0; i < 11; i++ {
digit, _ := strconv.Atoi(string(inn[i]))
sum += digit * weights2[i]
}
control2 := sum % 11
if control2 >= 10 {
control2 = 0
}

actualControl2, _ := strconv.Atoi(string(inn[11]))
return control2 == actualControl2
}

return false
}

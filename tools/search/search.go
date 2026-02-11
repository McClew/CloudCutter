package search

import (
	// Standard library dependencies
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	// Internal dependencies
	"CloudCutter/internal/logger"
	"CloudCutter/models"
)

// Operators and their precedence
var operators = map[string]int{
	"OR":  1,
	"AND": 2,
	"==":  3, "!=": 3, ">": 3, ">=": 3, "<": 3, "<=": 3, "LIKE": 3,
}

// Query the events with the given query
func Query(events []models.PurviewEvent, query string) []models.PurviewEvent {
	// Tokenise the query string into tokens
	tokens := tokenise(query)
	logger.Debugf("Found %d tokens", len(tokens))

	// Preprocess the tokens
	tokens = preprocessTokens(tokens)
	rpn := shunt(tokens)
	logger.Debugf("RPN: %v", rpn)

	// Evaluate the RPN
	var filteredEvents []models.PurviewEvent

	// Loop through the events & evaluate the RPN
	for _, event := range events {
		if evaluate(rpn, event) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents
}

// Tokenise the query string into tokens
func tokenise(query string) []string {
	// Regex to match tokens: strings (single/double quoted), operators, parens, identifiers/numbers
	expression := regexp.MustCompile(`"(?:\\.|[^"])*"|'(?:\\.|[^'])*'|>=|<=|==|!=|[><()]|[^\s()!><=]+`)
	return expression.FindAllString(query, -1)
}

// Preprocess the tokens to handle cases where PowerShell strips quotes around strings with spaces
func preprocessTokens(tokens []string) []string {
	// Store the processed tokens
	var processed []string

	for index := 0; index < len(tokens); index++ {
		// Preprocess the token
		token := tokens[index]
		processed = append(processed, token)
		upper := strings.ToUpper(token)

		// Check if it is a comparison operator
		if operatorKey, isOperator := operators[upper]; isOperator && operatorKey == 3 {
			// Store the parts of the comparison
			var parts []string

			// Get the index of the next token
			futureIndex := index + 1

			// Loop through the tokens
			for futureIndex < len(tokens) {
				// Get the next token
				next := tokens[futureIndex]
				nextUpper := strings.ToUpper(next)

				// Stop if logical operator or parenthesis is found
				if next == "(" || next == ")" || nextUpper == "AND" || nextUpper == "OR" {
					break
				}

				// Add the next token to the parts array
				parts = append(parts, next)
				futureIndex++
			}

			// If there are multiple parts
			if len(parts) > 1 {
				// Merge them back together
				merged := strings.Join(parts, " ")
				processed = append(processed, merged)

				// Skip the merged parts
				index = futureIndex - 1
			} else if len(parts) == 1 {
				processed = append(processed, parts[0])
				index = futureIndex - 1
			}
		}
	}

	return processed
}

// Shunt the tokens into reverse polish notation
func shunt(tokens []string) []string {
	var output []string
	var stack []string

	for _, token := range tokens {
		upperToken := strings.ToUpper(token)

		// Handle the token
		// Reorder the tokens to reverse polish notation
		switch {
		case isValue(token):
			output = append(output, token)
		case token == "(":
			stack = append(stack, token)
		case token == ")":
			// Hunt for the matching parenthesis
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}

			// Remove the matching parenthesis
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
			// Default is an operator
			operatorKey := token
			if upperToken == "AND" || upperToken == "OR" {
				operatorKey = upperToken
			}

			// Push the operator to the stack
			for len(stack) > 0 && stack[len(stack)-1] != "(" && operators[stack[len(stack)-1]] >= operators[operatorKey] {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}

			// Add the operator to the stack
			stack = append(stack, operatorKey)
		}
	}

	// Add any remaining operators to the output
	for len(stack) > 0 {
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output
}

// Check if the token is a value, not an operator or parenthesis
func isValue(token string) bool {
	upperToken := strings.ToUpper(token)
	_, isOperator := operators[upperToken]
	_, isOperatorOrig := operators[token]
	return !isOperator && !isOperatorOrig && token != "(" && token != ")"
}

// Evaluate the reverse polish notation
func evaluate(rpn []string, event models.PurviewEvent) bool {
	// Must be []any to hold both strings (from resolve) and bools (from compute)
	var stack []any

	// Evaluate the RPN
	for _, token := range rpn {
		if isValue(token) {
			// Push the value to the stack
			stack = append(stack, resolveValue(token, event))
		} else {
			// If there are not enough values on the stack, return false
			if len(stack) < 2 {
				return false
			}

			// Pop the top two values from the stack
			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			// Compute the result
			result := compute(left, token, right)
			logger.Debugf("Compute: %v %s %v -> %v", left, token, right, result)
			stack = append(stack, result)
		}
	}

	// If there is not exactly one value on the stack, return false
	if len(stack) != 1 {
		return false
	}

	// Return the result
	result, isBool := stack[0].(bool)

	if !isBool {
		return false
	}

	return result
}

// Resolve the value of a token
func resolveValue(token string, event models.PurviewEvent) any {
	// Strip the token of quotes
	cleanToken := strings.Trim(token, "\"'")

	// If the token is a string, return it
	if strings.HasPrefix(token, "'") || strings.HasPrefix(token, "\"") {
		return cleanToken
	}

	// Split the token into parts
	parts := strings.Split(cleanToken, ".")

	// Try resolving via struct fields
	res := resolveRecursive(parts, reflect.ValueOf(event))
	if res != nil {
		logger.Debugf("Resolved '%s' via struct fields to '%v'", token, res)
		return res
	}

	// Try resolving via Flattened map (top level match)
	if val, ok := event.Flattened[strings.ToLower(parts[0])]; ok {
		res = resolveRecursive(parts[1:], reflect.ValueOf(val))
		if res != nil {
			logger.Debugf("Resolved '%s' via Flattened map to '%v'", token, res)
			return res
		}
	}

	// Try resolving via AuditData map (case-insensitive key match)
	for k, v := range event.AuditData {
		if strings.EqualFold(k, parts[0]) {
			res = resolveRecursive(parts[1:], reflect.ValueOf(v))
			if res != nil {
				return res
			}
		}
	}

	// Fallback for single parts that aren't fields: treat as literal
	if len(parts) == 1 {
		return cleanToken
	}

	return nil
}

func resolveRecursive(parts []string, val reflect.Value) any {
	// Handle pointer/interface first to get to the underlying value
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if len(parts) == 0 {
		if !val.IsValid() {
			return nil
		}
		return val.Interface()
	}

	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		var results []any
		for i := 0; i < val.Len(); i++ {
			res := resolveRecursive(parts, val.Index(i))
			if res != nil {
				if slice, ok := res.([]any); ok {
					results = append(results, slice...)
				} else {
					results = append(results, res)
				}
			}
		}
		if len(results) == 0 {
			return nil
		}
		return results

	case reflect.Struct:
		cleanPart := parts[0]
		for i := 0; i < val.NumField(); i++ {
			field := val.Type().Field(i)
			if strings.EqualFold(field.Name, cleanPart) {
				return resolveRecursive(parts[1:], val.Field(i))
			}
		}

	case reflect.Map:
		cleanPart := parts[0]
		for _, key := range val.MapKeys() {
			keyStr := fmt.Sprint(key.Interface())
			if strings.EqualFold(keyStr, cleanPart) {
				return resolveRecursive(parts[1:], val.MapIndex(key))
			}
		}
	}

	return nil
}

// Compute the result of an operation
// This is a bit of a mess, but it works
func compute(left any, op string, right any) bool {
	// If left is a slice, perform "any" logic
	if left != nil && reflect.TypeOf(left).Kind() == reflect.Slice {
		v := reflect.ValueOf(left)
		for i := 0; i < v.Len(); i++ {
			if compute(v.Index(i).Interface(), op, right) {
				return true
			}
		}
		return false
	}

	// If it's a logical operation, we expect booleans
	if op == "AND" || op == "OR" {
		lBool, _ := left.(bool)
		rBool, _ := right.(bool)
		if op == "AND" {
			return lBool && rBool
		}
		return lBool || rBool
	}

	// fmt.Printf("DEBUG: compare '%v' %s '%v'\n", left, op, right)

	// For comparisons, we expect strings/numbers
	sLeft := strings.TrimSpace(fmt.Sprintf("%v", left))
	sRight := strings.TrimSpace(fmt.Sprintf("%v", right))

	// Try date/time comparison first
	tLeft, okL := tryParseTime(sLeft)
	tRight, okR := tryParseTime(sRight)
	if okL && okR {
		switch op {
		case "==":
			return tLeft.Equal(tRight)
		case "!=":
			return !tLeft.Equal(tRight)
		case ">":
			return tLeft.After(tRight)
		case ">=":
			return tLeft.After(tRight) || tLeft.Equal(tRight)
		case "<":
			return tLeft.Before(tRight)
		case "<=":
			return tLeft.Before(tRight) || tLeft.Equal(tRight)
		}
	}

	lVal, errL := strconv.ParseFloat(sLeft, 64)
	rVal, errR := strconv.ParseFloat(sRight, 64)

	// Numeric comparison if both sides are valid numbers
	if errL == nil && errR == nil {
		switch op {
		case "==":
			return lVal == rVal
		case "!=":
			return lVal != rVal
		case ">":
			return lVal > rVal
		case ">=":
			return lVal >= rVal
		case "<":
			return lVal < rVal
		case "<=":
			return lVal <= rVal
		}
	}

	// Default to string comparison
	switch op {
	case "==":
		return strings.EqualFold(sLeft, sRight)
	case "!=":
		return !strings.EqualFold(sLeft, sRight)
	case ">":
		return sLeft > sRight
	case ">=":
		return sLeft >= sRight
	case "<":
		return sLeft < sRight
	case "<=":
		return sLeft <= sRight
	case "LIKE":
		// Handle SQL-style wildcards (* or %)
		if strings.ContainsAny(sRight, "*%") {
			// Convert wildcard pattern to regex
			pattern := sRight
			// Escape regex special characters except our wildcards
			pattern = regexp.QuoteMeta(pattern)
			// Replace our wildcards with .*
			// Note: * is escaped as \* by QuoteMeta, but % is not.
			pattern = strings.ReplaceAll(pattern, "\\*", ".*")
			pattern = strings.ReplaceAll(pattern, "%", ".*")
			// Add anchors for full match
			pattern = "^" + pattern + "$"

			matched, err := regexp.MatchString("(?i)"+pattern, sLeft)
			if err != nil {
				return false
			}
			return matched
		}
		// Fallback to substring match if no wildcards
		return strings.Contains(strings.ToLower(sLeft), strings.ToLower(sRight))
	}
	return false
}

// tryParseTime attempts to parse a string into a time.Time using common formats
func tryParseTime(s string) (time.Time, bool) {
	formats := []string{
		"2006-01-02",          // Date
		"15:04:05",            // Time
		time.RFC3339,          // RFC3339
		"2006-01-02T15:04:05", // Combined fallback
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

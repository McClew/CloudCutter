package search

import (
	// Standard library dependencies
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	// Internal dependencies
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
	tokens := tokenise(query)
	tokens = preprocessTokens(tokens)
	rpn := shunt(tokens)

	var filteredEvents []models.PurviewEvent
	for _, event := range events {
		if evaluate(rpn, event) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	return filteredEvents
}

// Helpers
// Tokenise the query string into tokens
func tokenise(query string) []string {
	// Regex to match tokens: strings (single/double quoted), operators, parens, identifiers/numbers
	re := regexp.MustCompile(`"(?:\\.|[^"])*"|'(?:\\.|[^'])*'|>=|<=|==|!=|[><()]|[^\s()!><=]+`)
	return re.FindAllString(query, -1)
}

// preprocessTokens handles cases where PowerShell strips quotes around strings with spaces
func preprocessTokens(tokens []string) []string {
	var processed []string
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		processed = append(processed, token)

		// Check if it's a comparison operator (==, !=, LIKE, etc.)
		upper := strings.ToUpper(token)
		if prec, ok := operators[upper]; ok && prec == 3 {
			// Look ahead for multiple non-operator tokens
			var valParts []string
			j := i + 1
			for j < len(tokens) {
				next := tokens[j]
				nextUpper := strings.ToUpper(next)
				// Stop if we hit a logical operator or parenthesis
				if next == "(" || next == ")" || nextUpper == "AND" || nextUpper == "OR" {
					break
				}
				valParts = append(valParts, next)
				j++
			}

			if len(valParts) > 1 {
				// Merge them back together
				merged := strings.Join(valParts, " ")
				processed = append(processed, merged)
				i = j - 1 // Skip the merged parts
			} else if len(valParts) == 1 {
				processed = append(processed, valParts[0])
				i = j - 1
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
		upperToken := strings.ToUpper(token) // Normalize AND/OR
		switch {
		case isValue(token):
			output = append(output, token)
		case token == "(":
			stack = append(stack, token)
		case token == ")":
			for len(stack) > 0 && stack[len(stack)-1] != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
			// Handle AND/OR case sensitivity by checking upperToken
			opKey := token
			if upperToken == "AND" || upperToken == "OR" {
				opKey = upperToken
			}

			for len(stack) > 0 && stack[len(stack)-1] != "(" && operators[stack[len(stack)-1]] >= operators[opKey] {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, opKey)
		}
	}

	for len(stack) > 0 {
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output
}

// Check if the token is a value, not an operator or parenthesis
func isValue(token string) bool {
	upper := strings.ToUpper(token)
	_, isOperator := operators[upper]
	_, isOperatorOrig := operators[token]
	return !isOperator && !isOperatorOrig && token != "(" && token != ")"
}

// Evaluate the reverse polish notation
func evaluate(rpn []string, event models.PurviewEvent) bool {
	// Must be []any to hold both strings (from resolve) and bools (from compute)
	var stack []any

	for _, token := range rpn {
		if isValue(token) {
			stack = append(stack, resolveValue(token, event))
		} else {
			if len(stack) < 2 {
				return false
			}

			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			result := compute(left, token, right)
			stack = append(stack, result)
		}
	}

	if len(stack) == 0 {
		return false
	}
	res, ok := stack[0].(bool)
	return ok && res
}

// Resolve the value of a token
func resolveValue(token string, event models.PurviewEvent) any {
	cleanToken := strings.Trim(token, "\"'")
	if strings.HasPrefix(token, "'") || strings.HasPrefix(token, "\"") {
		return cleanToken
	}

	parts := strings.Split(cleanToken, ".")

	// 1. Try resolving via struct fields
	res := resolveRecursive(parts, reflect.ValueOf(event))
	if res != nil {
		return res
	}

	// 2. Try resolving via Flattened map (top level match)
	if val, ok := event.Flattened[strings.ToLower(parts[0])]; ok {
		res = resolveRecursive(parts[1:], reflect.ValueOf(val))
		if res != nil {
			return res
		}
	}

	// 3. Try resolving via AuditData map (case-insensitive key match)
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
	sLeft := fmt.Sprintf("%v", left)
	sRight := fmt.Sprintf("%v", right)

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
		return strings.Contains(strings.ToLower(sLeft), strings.ToLower(sRight))
	}
	return false
}

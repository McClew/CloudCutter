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
	"OR": 1, "AND": 2,
	"==": 3, "!=": 3, ">": 3, ">=": 3, "<": 3, "<=": 3,
}

// Query the events with the given query
func Query(events []models.PurviewEvent, query string) []models.PurviewEvent {
	tokens := tokenise(query)
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
	re := regexp.MustCompile(`"(?:\\.|[^"])*"|'(?:\\.|[^'])*'|>=|<=|==|!=|[><()]|[\w\.\-]+`)
	return re.FindAllString(query, -1)
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
func resolveValue(token string, event models.PurviewEvent) string {
	cleanToken := strings.Trim(token, "\"'")
	val := reflect.ValueOf(event)
	for i := 0; i < val.NumField(); i++ {
		if strings.EqualFold(val.Type().Field(i).Name, cleanToken) {
			return fmt.Sprintf("%v", val.Field(i).Interface())
		}
	}
	return cleanToken
}

// Compute the result of an operation
// This is a bit of a mess, but it works
func compute(left any, op string, right any) bool {
	// If it's a logical operation, we expect booleans
	if op == "AND" || op == "OR" {
		lBool, _ := left.(bool)
		rBool, _ := right.(bool)
		if op == "AND" {
			return lBool && rBool
		}
		return lBool || rBool
	}

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
		return sLeft == sRight
	case "!=":
		return sLeft != sRight
	case ">":
		return sLeft > sRight
	case ">=":
		return sLeft >= sRight
	case "<":
		return sLeft < sRight
	case "<=":
		return sLeft <= sRight
	}
	return false
}

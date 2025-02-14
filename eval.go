package tggl

import (
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pierrec/xxHash/xxHash32"
)

// evalFlag evaluates a flag with a given context and returns the variation value
func evalFlag(flag Flag, context Context) interface{} {
	// Evaluate each condition
	for _, condition := range flag.Conditions {
		allRulesValid := true

		// All rules in a condition must be valid
		for _, rule := range condition.Rules {
			if !evaluateRule(rule, context) {
				allRulesValid = false
				break
			}
		}

		if allRulesValid {
			if condition.Variation.Active {
				return condition.Variation.Value
			}
			return nil
		}
	}

	if flag.DefaultVariation.Active {
		return flag.DefaultVariation.Value
	}
	return nil
}

func evaluateRule(rule Rule, context Context) bool {
	// Get the context value for the rule's key
	contextValue, exists := context[rule.Key]
	if (!exists || contextValue == nil) && rule.Operator != "EMPTY" {
		return false
	}

	// Evaluate based on operator
	result := false

	switch rule.Operator {
	case "EMPTY":
		if contextValue == nil {
			result = true
		} else {
			if str, ok := contextValue.(string); ok {
				result = str == ""
			}
		}

	case "STR_EQUAL":
		str, ok := contextValue.(string)
		if !ok {
			return false
		}
		for _, value := range rule.Values {
			if str == value {
				result = true
				break
			}
		}

	case "STR_EQUAL_SOFT":
		str, okStr := contextValue.(string)
		nb, okNb := contextValue.(float64)
		if !okStr && !okNb {
			return false
		}

		for _, value := range rule.Values {
			if strings.EqualFold(str, value) || strings.EqualFold(fmt.Sprint(nb), value) {
				result = true
				break
			}
		}

	case "STR_CONTAINS":
		str, ok := contextValue.(string)
		if !ok {
			return false
		}
		if str == "" {
			break
		}
		for _, value := range rule.Values {
			if strings.Contains(str, value) {
				result = true
				break
			}
		}

	case "STR_STARTS_WITH":
		str, ok := contextValue.(string)
		if !ok {
			return false
		}
		if str == "" {
			break
		}
		for _, value := range rule.Values {
			if strings.HasPrefix(str, value) {
				result = true
				break
			}
		}

	case "STR_ENDS_WITH":
		str, ok := contextValue.(string)
		if !ok {
			return false
		}
		if str == "" {
			break
		}
		for _, value := range rule.Values {
			if strings.HasSuffix(str, value) {
				result = true
				break
			}
		}

	case "REGEXP":
		str, ok := contextValue.(string)
		if !ok {
			return false
		}
		if str == "" {
			break
		}
		patternStr, ok2 := rule.Value.(string)
		if ok && ok2 {
			if matched, err := regexp.MatchString(patternStr, str); err == nil {
				result = matched
				break
			}
		}

	case "STR_BEFORE":
		valStr, ok := contextValue.(string)
		ruleStr, ok2 := rule.Value.(string)
		if !ok || !ok2 {
			return false
		}
		if valStr == "" {
			break
		}
		result = valStr <= ruleStr

	case "STR_AFTER":
		valStr, ok := contextValue.(string)
		ruleStr, ok2 := rule.Value.(string)
		if !ok || !ok2 {
			return false
		}
		if valStr == "" {
			break
		}
		result = valStr >= ruleStr

	case "EQ", "GT", "LT":
		valNb, ok := contextValue.(float64)
		ruleNb, ok2 := rule.Value.(float64)
		if !ok || !ok2 {
			return false
		}
		switch rule.Operator {
		case "EQ":
			result = valNb == ruleNb
		case "GT":
			result = valNb > ruleNb
		case "LT":
			result = valNb < ruleNb
		}

	case "TRUE":
		boolean, okBool := contextValue.(bool)
		if !okBool {
			return false
		}
		result = boolean

	case "ARR_OVERLAP":
		ok := reflect.TypeOf(contextValue).Kind() == reflect.Slice
		if !ok {
			return false
		}
		contextSlice := reflect.ValueOf(contextValue)
		result = false
		for i := 0; i < contextSlice.Len(); i++ {
			item := contextSlice.Index(i).Interface()
			for _, ruleValue := range rule.Values {
				if fmt.Sprint(item) == ruleValue {
					result = true
					break
				}
			}
			if result {
				break
			}
		}

	case "DATE_AFTER", "DATE_BEFORE":
		switch contextValue.(type) {
		case string:
			valStr, ok := contextValue.(string)
			if ok {
				if rule.ISO == nil {
					return false
				}
				t, err := parseDate(valStr)
				if err != nil {
					return false
				}
				ruleISO := *rule.ISO
				if len(ruleISO) > len(valStr) {
					ruleISO = ruleISO[:len(valStr)]
				}
				iso, err := parseDate(ruleISO)
				if err != nil {
					return false
				}

				switch rule.Operator {
				case "DATE_AFTER":
					result = t.Compare(iso) >= 0
				case "DATE_BEFORE":
					result = t.Compare(iso) <= 0
				}
			}
		case float64:
			valInt, ok := contextValue.(float64)
			if ok {
				if rule.Timestamp == nil {
					return false
				}

				// Convert to milliseconds if value is in seconds
				valInMillis := valInt
				if valInt < 631152000000 { // Before year 1990 in milliseconds
					valInMillis = valInt * 1000
				}

				switch rule.Operator {
				case "DATE_AFTER":
					result = int(valInMillis) >= *rule.Timestamp
				case "DATE_BEFORE":
					result = int(valInMillis) <= *rule.Timestamp
				}
			}
		default:
			return false
		}

	case "SEMVER_EQ":
		valStr, ok := contextValue.(string)
		if !ok {
			return false
		}
		s := strings.Split(valStr, ".")
		result = true
		for i := 0; i < len(rule.Version); i++ {
			if i >= len(s) {
				result = false
				break
			}
			v, err := strconv.Atoi(s[i])
			if err != nil {
				result = false
				break
			}
			if v != rule.Version[i] {
				result = false
				break
			}
		}

	case "SEMVER_GTE":
		valStr, ok := contextValue.(string)
		if !ok {
			return false
		}
		s := strings.Split(valStr, ".")
		for i := 0; i < len(rule.Version); i++ {
			if i >= len(s) {
				result = false
				break
			}
			v, err := strconv.Atoi(s[i])
			if err != nil {
				break
			}
			if v > rule.Version[i] {
				result = true
				break
			}
			if v < rule.Version[i] {
				result = false
				break
			}
			if v == rule.Version[i] {
				result = true
			}
		}

	case "SEMVER_LTE":
		valStr, ok := contextValue.(string)
		if !ok {
			return false
		}
		s := strings.Split(valStr, ".")
		for i := 0; i < len(rule.Version); i++ {
			if i >= len(s) {
				result = false
				break
			}
			v, err := strconv.Atoi(s[i])
			if err != nil {
				break
			}
			if v < rule.Version[i] {
				result = true
				break
			}
			if v > rule.Version[i] {
				result = false
				break
			}
			if v == rule.Version[i] {
				result = true
			}
		}

	case "PERCENTAGE":
		strValue := ""
		switch v := contextValue.(type) {
		case string:
			strValue = v
		case int, int64, float64:
			strValue = strconv.FormatFloat(v.(float64), 'f', -1, 64)
		default:
			return false
		}

		seed := 0
		if rule.Seed != nil {
			seed = *rule.Seed
		}

		x := xxHash32.New(uint32(seed))
		x.Write([]byte(strValue))
		h := x.Sum32()

		// Normalize between 0 and 1
		probability := float64(h) / float64(0xffffffff)

		// Handle case where probability == 1
		if probability == 1 {
			probability -= math.SmallestNonzeroFloat64
		}

		if rule.RangeStart == nil || rule.RangeEnd == nil {
			result = false
			break
		}

		result = probability >= *rule.RangeStart && probability < *rule.RangeEnd
	}

	// Apply negation if specified
	if rule.Negate != nil && *rule.Negate {
		return !result
	}
	return result
}

func parseDate(valStr string) (time.Time, error) {
	var format string
	switch len(valStr) {
	case len(time.RFC3339):
		format = time.RFC3339
	case len("2006-01-02T15:04:05Z"):
		format = "2006-01-02T15:04:05Z"
	case len("2006-01-02T15:04:05"):
		format = "2006-01-02T15:04:05"
	case len("2006-01-02T15:04"):
		format = "2006-01-02T15:04"
	case len("2006-01-02T15"):
		format = "2006-01-02T15"
	case len(time.DateOnly):
		format = time.DateOnly
	default:
		return time.Time{}, fmt.Errorf("invalid date format: %s", valStr)
	}
	return time.Parse(format, valStr)
}

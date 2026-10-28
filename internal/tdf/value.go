package tdf

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// Value is a value in a script. The engine keeps every value as a string; each reader converts it on its own, and
// leniently: atoi reads "12abc" as 12 and "abc" as 0. The conversions here are strict and return an error instead.
type Value string

// String returns the value as it is in the script.
func (v Value) String() string {
	return string(v)
}

// AsInt returns the value as a decimal integer, like "-12".
// Returns an error if it is anything else or does not fit an int.
func (v Value) AsInt() (int, error) {
	result, err := strconv.Atoi(string(v))
	if err != nil {
		return 0, fmt.Errorf("%q is not an integer: %w", string(v), numError(err))
	}
	return result, nil
}

// AsFloat returns the value as a finite floating point number, like "0.74000001" or "1e3".
// Returns an error if it is anything else, infinite or NaN.
func (v Value) AsFloat() (float64, error) {
	result, err := strconv.ParseFloat(string(v), 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not a number: %w", string(v), numError(err))
	}
	if math.IsInf(result, 0) || math.IsNaN(result) {
		return 0, fmt.Errorf("%q is not a finite number", string(v))
	}
	return result, nil
}

// AsBool returns the value as a boolean. The engine and the lobbies write booleans as "0" and "1".
// Returns an error if it is anything else.
func (v Value) AsBool() (bool, error) {
	switch v {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf(`%q is not a boolean, want "0" or "1"`, string(v))
	}
}

// numError returns the cause of a strconv error, which, unlike the error itself, does not repeat the value.
func numError(err error) error {
	if numErr, ok := errors.AsType[*strconv.NumError](err); ok {
		return numErr.Err
	}
	return err
}

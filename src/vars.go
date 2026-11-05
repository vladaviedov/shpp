/**
 * @file vars.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version 0.9.0
 * @date 2026
 * @license GPLv3.0
 * @brief Substitution variable manager.
 */
package main

import (
	"fmt"
	"strconv"
)

func setVar(state *State, key string, value string) error {
	_, err := strconv.ParseUint(key, 10, 64)
	if err == nil {
		return fmt.Errorf("syntax error: cannot set numerical variable with macro")
	}

	state.Vars[key] = value
	return nil
}

func getVar(state *State, key string) (string, error) {
	num, err := strconv.ParseUint(key, 10, 64)
	if err == nil {
		if num < state.ArgCount {
			return state.ArgData[num], nil
		}

		return "", fmt.Errorf("syntax error: numerical variable '%d' out of bounds", num)
	}

	value, valid := state.Vars[key]
	if !valid {
		return "", fmt.Errorf("syntax error: variable '%s' not defined", key)
	}

	return value, nil
}

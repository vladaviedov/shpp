/**
 * @file vars_test.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief File scanner and token lexer.
 */
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetVar_TryNumerical(t *testing.T) {
	assert := assert.New(t)
	state := &State{Vars: make(map[string]string)}

	err := setVar(state, "1", "test")

	assert.NotNil(err)
}

func TestSetVar_CreateNormal(t *testing.T) {
	assert := assert.New(t)
	state := &State{Vars: make(map[string]string)}

	err := setVar(state, "a", "test")

	assert.Nil(err)
	assert.Equal("test", state.Vars["a"])
}

func TestSetVar_UpdateNormal(t *testing.T) {
	assert := assert.New(t)
	state := &State{
		Vars: map[string]string{"a": "before"},
	}

	err := setVar(state, "a", "test")

	assert.Nil(err)
	assert.Equal("test", state.Vars["a"])
}

func TestGetVar_Numerical(t *testing.T) {
	assert := assert.New(t)
	state := &State{
		ArgCount: 1,
		ArgData:  []string{"numtest"},
		Vars:     make(map[string]string),
	}

	value, err := getVar(state, "0")

	assert.Nil(err)
	assert.Equal("numtest", value)
}

func TestGetVar_NumericalOutOfRange(t *testing.T) {
	assert := assert.New(t)
	state := &State{
		ArgCount: 1,
		ArgData:  []string{"numtest"},
		Vars:     make(map[string]string),
	}

	_, err := getVar(state, "2")

	assert.NotNil(err)

	_, err = getVar(state, "-1")

	assert.NotNil(err)
}

func TestGetVar_Undefined(t *testing.T) {
	assert := assert.New(t)
	state := &State{
		Vars: map[string]string{"a": "test"},
	}

	_, err := getVar(state, "b")

	assert.NotNil(err)
}

func TestGetVar_Normal(t *testing.T) {
	assert := assert.New(t)
	state := &State{
		Vars: map[string]string{
			"a": "test1",
			"b": "test2",
		},
	}

	value, err := getVar(state, "a")

	assert.Nil(err)
	assert.Equal("test1", value)

	value, err = getVar(state, "b")

	assert.Nil(err)
	assert.Equal("test2", value)
}

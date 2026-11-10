/**
 * @file util.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version 0.9.1
 * @date 2026
 * @license GPLv3.0
 * @brief Utility functions.
 */
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertPath_AbsoluteInput(t *testing.T) {
	assert := assert.New(t)
	opts.Root = "/home/mock/site"

	newPath := convertPath("/home/mock/site/index/index.shpp", "/home/mock/site/index")

	assert.Equal("/home/mock/site/index/index.shpp", newPath)
}

func TestConvertPath_RelativeInput(t *testing.T) {
	assert := assert.New(t)
	opts.Root = "/home/mock/site"

	newPath := convertPath("index.shpp", "/home/mock/site/index")

	assert.Equal("/home/mock/site/index/index.shpp", newPath)
}

func TestConvertPath_RootSubstitution(t *testing.T) {
	assert := assert.New(t)
	opts.Root = "/home/mock/site"

	newPath := convertPath("$nav/nav.shpp", "/home/mock/site/index")

	assert.Equal("/home/mock/site/nav/nav.shpp", newPath)
}

/**
 * @file assets.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version pre0.8.0
 * @date 2026
 * @license GPLv3.0
 * @brief Asset management.
 */
package main

import (
	"os"
	"strconv"
)

type AssetKind uint64

const (
	aStylesheet AssetKind = iota
	aScript
	aScriptDefer
	aScriptAsync
	aBinary
)

type Asset struct {
	Kind       AssetKind
	SourcePath string
}

func createAsset(kind AssetKind, fileDir string, path string) Asset {
	return Asset{
		Kind:       kind,
		SourcePath: ConvertPath(path, fileDir),
	}
}

func updateHead(state *State) error {
	// On shac files, we want to add a base tag
	// Do this before adding any links
	if opts.ShacInput {
		state.Emitter.WriteOut(`<base href="@$@/">`)
	}

	for i, asset := range state.Assets {
		generated, err := generateHeadAsset(&asset, i)
		if err != nil {
			return err
		}

		err = state.Emitter.WriteOut(generated)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateHeadAsset(asset *Asset, id int) (string, error) {
	switch asset.Kind {
	case aStylesheet:
		return generateStyleTag(asset, id)
	case aScript:
		return generateScriptTag(asset, id, "")
	case aScriptDefer:
		return generateScriptTag(asset, id, "defer")
	case aScriptAsync:
		return generateScriptTag(asset, id, "async")
	default:
		panic("invalid asset label")
	}
}

func generateStyleTag(asset *Asset, id int) (string, error) {
	if opts.ShacInput {
		return `<link rel="stylesheet" href="@` + strconv.Itoa(id) + `@">`, nil
	} else {
		content, err := os.ReadFile(asset.SourcePath)
		if err != nil {
			return "", err
		}

		return `<style>` + string(content) + `</style>`, nil
	}
}

func generateScriptTag(asset *Asset, id int, modifier string) (string, error) {
	if modifier != "" {
		modifier = " " + modifier
	}

	if opts.ShacInput {
		return `<script src="@` + strconv.Itoa(id) + `@"` + modifier + `></script>`, nil
	} else {
		content, err := os.ReadFile(asset.SourcePath)
		if err != nil {
			return "", err
		}

		return `<script` + modifier + `>` + string(content) + `</script>`, nil
	}
}

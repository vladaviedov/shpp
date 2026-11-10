# Changelog

## 0.9.1

- Fix shell escape not being enforced
- Fix miscellaneous lexing errors uncovered from testing
- Improve testing

## 0.9.0

- New directives
    - `@interp`
    - `@pathfrom`
- Add strong mode
- Fix data being lost when processing large files
- Fix variables not being parsed properly
- Improve testing

## 0.8.0
- **Breaking**: `@manage` is now used inline
    - `@manage <img src="file.png" />` => `<img src="@manage file.png@" />`
- Codebase rebuild
- Support directive inlining
- Add limited testing
- Stop parsing HTML tree

## 0.7.0
- New directives
    - `@run`
    - `@template`
    - `@derive`
- Support script loading modifiers (`async` and `defer`)
- Show errors in included files
- Fix stdin input not working
- Fix pandoc option parsing

## 0.6.0
- New directives
    - `@ignore`

## 0.5.0
- Generate HTML DOCTYPE to avoid quirks mode
- Inline inclusion of non-HTML documents via `pandoc`
- Root directory substitution
- Debug and release builds
- New directives
    - `@pandoc`

## 0.4.0
- `shac`: Manage binary assets
- Fix bugs in HTML generation
- New directives
    - `@manage`

## 0.3.0
- `shac`: Allow input file generation
- `shac`: Manage stylesheets and scripts
- Change default output file name
- New directives
    - `@url`

## 0.2.0
- Add preamble
- Improved path handling
- More correct compilation using HTML tree objects
- `@style` and `@script` now insert content into `<head>`

## 0.1.0
- Recursive document compilation
- New directives
    - `@include`
    - `@style`
    - `@script`

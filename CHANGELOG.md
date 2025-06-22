# Changelog

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
- Added preamble
- Improved path handling
- More correct compilation using HTML tree objects
- `@style` and `@script` now insert content into `<head>`

## 0.1.0
- Recursive document compilation
- New directives
    - `@include`
    - `@style`
    - `@script`

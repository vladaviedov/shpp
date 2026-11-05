# SHPP Directive List

## General Directives

General directives can be used in both the preamble and the body.

### nop

```
@nop [args...]
```

Indicates no action. All optional arguments are ignored. Geneates a comment tag
when used in body.

Example:
```
@nop i dont care
```

### pathfrom

```
@pathfrom <normal/root/parent>
```

Changes how filepaths are resolved. By default, `normal` mode is used.

- `@pathfrom normal`: paths are resolved from the current file.
- `@pathfrom root`: paths are resolved from the root.
- `@pathfrom parent`: paths are resolved from the parent file (fallbacks to
  normal mode if the file has no parent).

This directive can be used multiple times.

Example:
```
@pathfrom parent
```

## Preamble Directives

Preamble directives are only allowed in the preamble.

### style

```
@style <filepath>
```

Adds a stylesheet to the source file.

Example:
```
@style global.css
```

### script

```
@script <filepath> [modifier]
```

Adds a script to the source file. The `async` and `defer` modifiers from HTML
can be used as the second argument to pass them into the final result.

Example:
```
@script logic.js defer
```

### ignore

```
@ignore
```

Stop the file from being further processed.

Example:
```
@ignore
```

### url

*This is a SHAC-only directive*

```
@url <path>
```

Tell SHAC where to place the final output file. If used multiple times, only the
last call will be used.

Example:
```
@url index.html
```

### template

```
@template <argcount>
```

Specifies the amount of arguments required to process this file. Throws an error
if the argument passed in does not match this value.

```
@template 3
```

## Body Directives

Preamble directives are only allowed in the body.

### include

```
@include <filepath>
```

Copies the contents of a file into the current file.

Example:
```
@include footer.html
```

### derive

```
@derive <filepath> [arglist...]
```

Used to include template files which require additional arguments.

Example:
```
@derive template.shpp.in arg1 arg2 arg3
```

### pandoc

*Shell escape is required to use this directive*

```
@pandoc <filepath> [inputfmt]
```

Used to include files which need to be processed with pandoc first. The
`inputfmt` field is used the pandoc input file specifier.

Example:
```
@pandoc article.md markdown
```

### run

*Shell escape is required to use this directive*

```
@run <cmd> [arglist...]
```

Executes a system command and inserts the standard output into the source file.
This command should be used very carefully.

Example:
```
@run grep ^PRETTY_NAME= /etc/os-release | cut -d= -f2 | tr -d \'\"\'
```

### manage

*This is a SHAC-only directive*

```
@manage <filepath>
```

Marks a file to be managed as an asset by SHAC and generates a placeholder to
refer back. Primarily meant to be used inline as part of the `src` attribute.

```
<img src="@manage logo.png@">
```

### interp

```
@interp [exprlist...]
```

Evaluates the expression list in directive mode and adds the results into the
source file. Mainly used for variable substitution outside of other directives.

Example:
```
@interp {.pwd} {1} literal
```

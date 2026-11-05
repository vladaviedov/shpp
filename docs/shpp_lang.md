# SHPP Language Description

## Introduction

`shpp` (Static HTML Pre-Processor) is a quality-of-life tool for writing pure
HTML by hand. It is meant to address the common annoyances present in such work
without major obstructions to the syntax of the language nor a need to host a
NodeJS webapp.

The input files are HTML code annotated with SHPP directives, and the output is
pure HTML. A sample input file might look something like this:

```
@style global.css
@script script.js

<html>
    <head>
        <title>My webpage</title>
    </head>
    <body>
        @include nav.html
        @include content.html
        @include footer.html
    </body>
</html>
```

## Assumptions

Currently, SHPP requires a few things to be able to parse an HTML file.

1. Top-level file must provide an `<html>` tag

Invalid code:

```
@style global.css

<head>
    <title>My website</title>
</head>
<body>
    @include content.html
</body>
```

2. Top-level file must provide an `<head>` tag

Invalid code:

```
@style global.css

<html>
    <body>
        @include content.html
    </body>
</html>
```

3. All files must include an HTML tag (comment tag also okay)

Invalid code:

```
@style global.css

@derive template.shpp.in arg0 arg1 arg2
```

Valid code:

```
@style global.css

<!-- PHONY -->
@derive template.shpp.in arg0 arg1 arg2
```

## File Structure

SHPP files consist of a preamble and a body. The preamble is considered
completed once the first HTML tag is seen in the stream.

All non-directive text is treated as a comment in the preamble. The preamble
generally contains directives which apply to the entire file.

The body contains mixed HTML and SHPP code. The body generally contains
directives which generate new HTML.

## Directive Mode

Directives start with the `@` character. Directives end either with a newline or
another `@` character:

```
@interp {1}
@interp {1}@ and @interp {2}@
```

Directives can be multiline if the line is terminated with a backslash:

```
@derive template.shpp.in \
    arg1 \
    arg2 \
    arg3
```

Strong mode is activated with an exclamation mark before the directive name. In
strong mode, newlines are treated as regular whitespace, so directives must be
closed with another `@` character. This mode is intented for environments where
you don't have control over newline generation (i.e. pandoc).

```
@!derive template.shpp.in "Argument
One" "Argument Two" "Argument Three"@
```

## Variables

Variables might be present as part of directive arguments. A variable is
enclosed in curly braces. A variable identifier may not start with a number.

```
{test}
{.pwd}
{myvar2}
```

Numerical variables instead point to arguments. They must consist entirely of
digits to be valid.

```
{0}
{1}
{10}
```

Variable evaluation is only allowed in directives.

## Special Characters

A null directive (`@@` or `@` followed by a newline) is treated as an escaped
literal `@` in HTML.

Inside directives, escaping is done with backslash following rules similar to
`sh`.

The `$` is a special variable which is subsituted for root directory, specified
on the command line.

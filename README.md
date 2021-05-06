# Text layout library for Golang [![API reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/benoitkugler/textlayout)

This module provides a chain of tools to layout text. It is mainly a port of the following C libraries : Pango, fribidi, fontconfig and harfbuzz.

## Overview

The package [fonts](fonts) provides the low level primitives to load and read font files. The selection of a font given some criterion (like language or style) is facilitated by the [fontconfig](fontconfig) package. Once a font is selected, [harfbuzz](harfbuzz) is responsible for laying out a line of text, that is transforming a sequence of unicode points (runes) to a sequence of positionned glyphs. [fribidi](fribidi) provides support for bidirectional text : it finds the alternates LTR and RTL sequences (embedding levels) in a paragraph. Finally, [pango](pango) wraps these tools to provide an higher level interface capable of laying out an entire text.

## Status of the project

This project is still experimental. Some parts of it are already usable (mainly [fribidi](fribidi), [fonts/truetype](fonts/truetype) and [harfbuzz](harfbuzz)), but breaking changes may be committed on the fly.

## Licensing

This module is licensed as MIT, but some packages (fribidi, pango) are derivative work and thus licensed as LGPL.

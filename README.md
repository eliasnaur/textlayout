# Text layout library for Golang [![API reference](https://img.shields.io/badge/godoc-reference-5272B4)](https://pkg.go.dev/github.com/benoitkugler/textlayout)

This module provides a chain of tools to layout text. It is mainly a port of the following C libraries harfbuzz and graphite.

## Project structure update

As of v0.1.0, the content of this repository has been split, with higher level, more experimental packages moved to [textprocessing](https://github.com/benoitkugler/textlayout).
The remaining packages are the more stable, low level logic used by [go-text](https://github.com/go-text/typesetting).

## Overview

The package [fonts](fonts) provides the low level primitives to load and read font files. Once a font is selected, [harfbuzz](harfbuzz) is responsible for laying out a line of text, that is transforming a sequence of unicode points (runes) to a sequence of positionned glyphs. Graphite fonts are supported via the [graphite](graphite) package.
Some higher level library may wrap these tools to provide an interface capable of laying out an entire text.

## Status of the project

This project is a work in progress. Some parts of it are already usable : [fonts/truetype](fonts/truetype), [harfbuzz](harfbuzz) and [graphite](graphite), but breaking changes may be committed on the fly.

## Licensing

This module is licensed as MIT.

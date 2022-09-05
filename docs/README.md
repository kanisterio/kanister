## Kanister Documentation

This folder contains the source of the Kanister documentation at
https://docs.kanister.io

## Building The Source

The contents are written in the
[reStructuredText](https://docutils.sourceforge.io/rst.html) format.

To rebuild the documentation, run this command from the project root folder:

```sh
make docs
```

The `docs` target uses the `ghcr.io/kanisterio/docker-sphinx` public image to
generate the HTML documents and store them in your local `/docs/_build/html`
folder.

## Language, Grammar, and Tone

When updating the documentation content, please follow these guidelines on the
language, grammar, and tone style:

* Documentation should be written in English.
* Prefer an active voice and present tense when possible.
* Use simple and direct language.
* Use gender-neutral language.
* Avoid personal pronouns ("I," "we," "us," "our," and "ours").
* Address the reader as "you" instead of "we".
* Do not use Latin phrases.
* Avoid jargon and idioms.
* If using acronyms, ensure they are clearly defined in the same document.
* If using an abbreviation, spell it out the first time it is used in the document unless it is commonly known. (example: TCP/IP)
* When referring to a Kubernetes Group (SIG, WG, or UG) do not use the hyphenated form unless it is for a specific purpose such as a file-name or URI.

These guidelines are derived from the
[cheatsheet](https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md#cheatsheet-content-design-formatting-and-language)
in the Kubernetes community style guide. Take a look at some of their examples
[here](https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md#language-grammar-and-tone).

## reStructured Text Styling Guide

* Use `*` for bullet points.
* Section headers are created using underlines with text. Underlining hierarchy is:
    * `*` for chapters
    * `=` for sections
    * `-` for subsections
    * `^` for subsubsections
    * `"` for paragraphs
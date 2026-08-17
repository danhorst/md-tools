## Purpose

Renders a markdown document into HTML so the toolkit can produce a viewable artifact rather than only transforming markdown source, including resolving referenced SVG assets into a self-contained document.

## ADDED Requirements

### Requirement: Render markdown to HTML

The system SHALL render a GitHub Flavored Markdown document to HTML.
Output SHALL be written to STDOUT by default.

#### Scenario: GFM constructs render

- **WHEN** the input contains headings, emphasis, lists, links, tables, fenced code blocks, and blockquotes
- **THEN** each renders as the corresponding HTML element

#### Scenario: Footnotes render

- **WHEN** the input contains a footnote reference `[^1]` and its definition
- **THEN** the output contains a linked reference and a footnote section

#### Scenario: Deterministic output

- **WHEN** the same input is rendered twice with the same flags
- **THEN** both outputs are byte-for-byte identical

### Requirement: GitHub alerts render as callouts

A blockquote whose first line is an alert marker — `[!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`, or `[!CAUTION]` — SHALL render as a callout carrying GitHub's class names, with the marker removed and a title added.
Marker matching SHALL be case-insensitive.
A blockquote whose first line is not a recognized marker SHALL render unchanged.

#### Scenario: Alert renders with class and title

- **WHEN** a blockquote opens with `[!NOTE]`
- **THEN** it carries the classes `markdown-alert` and `markdown-alert-note`, contains a title element reading `Note`, and the literal `[!NOTE]` does not appear in the output

#### Scenario: Every type is recognized

- **WHEN** a blockquote opens with any of the five markers
- **THEN** each renders with its own type-specific class and its own title

#### Scenario: Case-insensitive

- **WHEN** a blockquote opens with `[!note]`
- **THEN** it renders as a note alert

#### Scenario: Marker alone

- **WHEN** a blockquote contains only the marker line
- **THEN** it renders as an alert with a title and no body

#### Scenario: Unrecognized marker left alone

- **WHEN** a blockquote opens with `[!BOGUS]`
- **THEN** it renders as an ordinary blockquote with the text intact

#### Scenario: Ordinary blockquote unaffected

- **WHEN** a blockquote does not open with a marker
- **THEN** it renders as an ordinary blockquote

### Requirement: Standalone document structure

By default the system SHALL emit a complete HTML document: a doctype, `html`, `head` with a charset declaration and a title, and `body` containing the rendered content.

#### Scenario: Default output is a complete document

- **WHEN** no structural flags are given
- **THEN** the output begins with a doctype and contains `head` and `body` elements

#### Scenario: Fragment output

- **WHEN** `--fragment` is given
- **THEN** the output contains only the rendered body content, with no doctype, `html`, `head`, `body`, or stylesheet

### Requirement: Document title

The system SHALL set the document title from `--title` when given, and otherwise from the input file's base name without extension.

#### Scenario: Explicit title

- **WHEN** `--title "My Document"` is given
- **THEN** the `title` element contains `My Document`

#### Scenario: Title derived from filename

- **WHEN** the input is the file `notes.md` and `--title` is not given
- **THEN** the `title` element contains `notes`

#### Scenario: No title available

- **WHEN** input arrives on STDIN and `--title` is not given
- **THEN** the `title` element is present and empty

### Requirement: Stylesheet handling

The system SHALL embed a default stylesheet in the document head.
The default SHALL include print rules so the document paginates sensibly when printed.

#### Scenario: Default stylesheet

- **WHEN** no stylesheet flags are given
- **THEN** the output head contains an inline `style` element

#### Scenario: Stylesheet override

- **WHEN** `--css custom.css` is given
- **THEN** the head contains the contents of `custom.css` and not the default stylesheet

#### Scenario: Stylesheet suppressed

- **WHEN** `--no-css` is given
- **THEN** the output contains no `style` element

#### Scenario: Missing stylesheet file

- **WHEN** `--css` names a file that does not exist
- **THEN** the system exits non-zero with an error naming the file

### Requirement: Frontmatter is not rendered

The system SHALL remove a leading YAML frontmatter block from the input before rendering, rather than rendering it as document content.

#### Scenario: Frontmatter stripped

- **WHEN** the input begins with a `---` delimited frontmatter block
- **THEN** no part of that block appears in the output, and no horizontal rule is emitted in its place

#### Scenario: Horizontal rule preserved

- **WHEN** the input begins with body text followed by a `---` horizontal rule
- **THEN** the rule renders as an `hr` element

### Requirement: Embedded HTML passes through

The system SHALL pass raw HTML in the markdown source through to the output unchanged, so markup produced by other tools in the toolkit survives rendering.

#### Scenario: Sidenote markup survives

- **WHEN** the input contains the sidenote HTML that `mdsidenote` produces
- **THEN** that markup appears in the output unchanged

### Requirement: Inline referenced SVG

When `--inline-svg` is given, the system SHALL replace each image reference to a `.svg` file with the contents of that file as an inline `svg` element.
Identifiers within each inlined document SHALL be namespaced so that inlining two SVG files, or the same file twice, does not produce colliding identifiers.

#### Scenario: SVG inlined

- **WHEN** `--inline-svg` is given and the document references `diagram.svg`
- **THEN** the output contains an inline `svg` element and no `img` element for it

#### Scenario: Identifiers do not collide

- **WHEN** the document references the same SVG file twice
- **THEN** each inlined copy carries distinct identifiers, and the internal references within each copy resolve to its own identifiers

#### Scenario: Not inlined by default

- **WHEN** `--inline-svg` is not given
- **THEN** SVG references render as `img` elements and no file is read

#### Scenario: Non-SVG images untouched

- **WHEN** `--inline-svg` is given and the document references a `.png` file
- **THEN** that reference renders as an `img` element

#### Scenario: Missing SVG is fatal

- **WHEN** `--inline-svg` is given and a referenced `.svg` file cannot be read
- **THEN** the system exits non-zero with an error naming the file

### Requirement: Active content in an inlined SVG is refused

Inlining moves an SVG from an image context, where a browser neither executes scripts nor loads external resources, into the live document where both happen.
The system SHALL therefore refuse to inline an SVG containing active content, naming the file and the construct found, unless `-unsafe-svg` is given.

An SVG that cannot be parsed SHALL also be refused: a document that cannot be inspected cannot be vouched for.

This check is a guard against accident and casual hostility, not a guarantee against a determined attacker.

#### Scenario: Script element refused

- **WHEN** `-inline-svg` is given and a referenced SVG contains a `script` element
- **THEN** the system exits non-zero with an error naming the file and the construct, and writes no document

#### Scenario: Event handler attribute refused

- **WHEN** a referenced SVG carries an `onload` or other `on*` attribute
- **THEN** the system exits non-zero with an error naming the file

#### Scenario: Embedded HTML refused

- **WHEN** a referenced SVG contains a `foreignObject` element
- **THEN** the system exits non-zero, and the error mentions that generating the diagram with HTML labels disabled avoids it

#### Scenario: Script URL refused

- **WHEN** a referenced SVG contains an `href` or `xlink:href` whose value is a `javascript:` URL
- **THEN** the system exits non-zero with an error naming the file

#### Scenario: External document reference refused

- **WHEN** a referenced SVG contains a `use` element whose reference is not a local fragment
- **THEN** the system exits non-zero with an error naming the file

#### Scenario: Unparseable file refused

- **WHEN** a referenced `.svg` file is not well-formed XML
- **THEN** the system exits non-zero with an error naming the file

#### Scenario: Override

- **WHEN** `-unsafe-svg` is given alongside `-inline-svg`
- **THEN** the SVG is inlined without inspection and the exit code is `0`

#### Scenario: Ordinary diagrams are unaffected

- **WHEN** a referenced SVG contains only shapes, text, markers, and a `style` block
- **THEN** it is inlined and the exit code is `0`

### Requirement: External references are reported, not resolved

The system SHALL NOT fetch anything over the network; output depends only on the input and the local files it names.
A remote `.svg` reference SHALL be left as an `img` element, and SHALL be reported on standard error so that an incomplete document is not mistaken for a self-contained one.
An SVG that loads external resources SHALL likewise be reported rather than refused, since it leaks the reader's address without executing.

#### Scenario: Remote reference left alone and reported

- **WHEN** `-inline-svg` is given and the document references an SVG by `https:` URL
- **THEN** that reference renders as an `img` element, a warning naming it is written to standard error, and the exit code is `0`

#### Scenario: Standard output stays clean

- **WHEN** a warning is written for a remote reference
- **THEN** standard output contains only the rendered document, so a pipeline is unaffected

#### Scenario: External resource inside an inlined SVG reported

- **WHEN** an inlined SVG references an external image or stylesheet
- **THEN** a warning naming the file is written to standard error and the SVG is still inlined

### Requirement: Resolving SVG references

Relative SVG references SHALL resolve against the input file's directory.
When input arrives on STDIN there is no such directory, so `--inline-svg` SHALL require `--base DIR` and resolve against it.

#### Scenario: Resolved against input file

- **WHEN** `--inline-svg` is given for the file `docs/notes.md`, which references `diagram.svg`
- **THEN** the system reads `docs/diagram.svg`

#### Scenario: Base directory required for STDIN

- **WHEN** `--inline-svg` is given, input arrives on STDIN, and `--base` is not given
- **THEN** the system exits non-zero with an error stating that `--base` is required

#### Scenario: Resolved against base directory

- **WHEN** `--inline-svg --base docs` is given and input on STDIN references `diagram.svg`
- **THEN** the system reads `docs/diagram.svg`

### Requirement: Filesystem access is opt-in

Without `--inline-svg` and `--css`, the system SHALL NOT read any file other than its input.

#### Scenario: No incidental reads

- **WHEN** neither `--inline-svg` nor `--css` is given
- **THEN** rendering succeeds even if referenced image files do not exist

### Requirement: Command-line contract

The system SHALL accept input from STDIN or a file path argument, write to STDOUT, exit `0` on success and non-zero on fatal error, and report its version under `-v` and `-version`.

The `-w` flag SHALL be rejected. Elsewhere in the toolkit it rewrites the input file with the result; here that would replace a markdown source file with HTML.

The `-unsafe-svg` flag SHALL skip inspection of inlined SVG. It has no effect without `-inline-svg`.

#### Scenario: Input from STDIN

- **WHEN** markdown is piped in with no file argument
- **THEN** the rendered HTML is written to STDOUT and the exit code is `0`

#### Scenario: Input from a file argument

- **WHEN** a file path is given as an argument
- **THEN** that file is rendered and the result written to STDOUT

#### Scenario: Write to a named file

- **WHEN** `-i out.html` is given with markdown on STDIN
- **THEN** the rendered HTML is written to `out.html`

#### Scenario: Destructive flag refused

- **WHEN** `-w` is given
- **THEN** the system exits non-zero with an error explaining that it would overwrite the markdown source, and suggesting `-i FILE` or shell redirection

#### Scenario: Version

- **WHEN** `-v` is given
- **THEN** the tool name and version are printed and the exit code is `0`

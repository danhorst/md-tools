## Purpose

Converting markdown footnotes into Tufte sidenote markup and back again, including what that conversion must survive when the toolkit's own reflow tools are applied between the two directions.

## ADDED Requirements

### Requirement: The marker attaches to the text it annotates

`mdsidenote` SHALL emit the sidenote marker immediately after the text the footnote reference followed, with no intervening whitespace.
A newline inside a paragraph is a space in every markdown renderer, so a marker on its own line renders detached from its referent.

#### Scenario: Marker follows sentence-ending punctuation

- **WHEN** a footnote reference follows a period
- **THEN** the emitted marker begins immediately after that period, with no space or newline between them

#### Scenario: Marker follows a word mid-sentence

- **WHEN** a footnote reference appears mid-sentence
- **THEN** the marker begins immediately after the preceding word, and the remainder of the sentence is preserved

#### Scenario: Reference at the start of a line

- **WHEN** a footnote reference is the first thing on a line
- **THEN** the marker starts that line, and no blank line is introduced

#### Scenario: Rendered output has no gap

- **WHEN** the emitted markdown is rendered to HTML
- **THEN** no whitespace appears between the referent text and the marker

### Requirement: Recovery tolerates whitespace

`mdfootnote` SHALL recover a footnote reference from sidenote markup regardless of the whitespace separating the marker's elements, and SHALL reattach the recovered reference to the preceding text.

#### Scenario: Attached layout

- **WHEN** the markup follows its referent with no intervening whitespace
- **THEN** the footnote reference is recovered and attached to that text

#### Scenario: Line-separated layout

- **WHEN** the markup is separated from its referent by a newline, as older documents contain
- **THEN** the footnote reference is recovered and attached to that text

#### Scenario: Space-joined layout

- **WHEN** the markup is separated from its referent by a space
- **THEN** the footnote reference is recovered and attached to that text

#### Scenario: Definitions are restored

- **WHEN** sidenote markup is recovered
- **THEN** the corresponding footnote definition is emitted with the note's content intact

### Requirement: Recovery is bounded to one note

`mdfootnote` SHALL recover each sidenote's content from that note alone, and SHALL NOT absorb text, markup, or subsequent notes that follow it.
Content between the note's own delimiters may itself contain markup, including nested elements, without extending the match.

#### Scenario: A second note is not absorbed

- **WHEN** a document contains two or more sidenotes
- **THEN** each is recovered as its own footnote definition, and no definition contains another note's content

#### Scenario: Following body text is preserved

- **WHEN** body text follows a sidenote
- **THEN** that text remains body text and does not become part of the footnote definition

#### Scenario: Nested markup inside a note

- **WHEN** a note's content contains its own inline elements
- **THEN** the note is recovered whole, and the match still ends at that note's own terminator

#### Scenario: Recovery is invertible

- **WHEN** a document without reference-style links inside its notes is converted to sidenotes and back
- **THEN** the result equals the original document byte for byte

#### Scenario: Emphasis style is preserved

- **WHEN** a note's content uses underscore emphasis
- **THEN** the recovered definition uses underscore emphasis, not asterisks

### Requirement: Reference-style links inside a note are not restorable

Converting to sidenotes inlines a reference-style link that appears inside a note, discarding the label.
Recovery therefore SHALL produce an inline link, and the surrounding link reference definitions may be renumbered.
This is a bound on the round-trip, not a recovery defect: the information needed to restore the reference form no longer exists in the sidenote markup.

#### Scenario: A reference link inside a note comes back inline

- **WHEN** a note containing a reference-style link is converted to a sidenote and recovered
- **THEN** the note's content is preserved, expressed as an inline link

#### Scenario: Everything else still round-trips

- **WHEN** such a document is converted and recovered
- **THEN** every note is recovered, no body text is absorbed, and each reference is reattached to its text

### Requirement: The round-trip survives reflow

Converting to sidenotes and back SHALL return the original document, and SHALL continue to do so when any of the toolkit's reflow tools runs between the two directions.

#### Scenario: Direct round-trip

- **WHEN** a document is converted to sidenotes and back
- **THEN** the result equals the original

#### Scenario: Round-trip through paragraph joining

- **WHEN** the sidenote form is passed through `mdjoin` before being converted back
- **THEN** the footnote references and definitions are recovered

#### Scenario: Round-trip through sentence splitting

- **WHEN** the sidenote form is passed through `mdsplit` before being converted back
- **THEN** the footnote references and definitions are recovered

#### Scenario: Round-trip through hard wrapping

- **WHEN** the sidenote form is passed through `mdwrap` before being converted back
- **THEN** the footnote references and definitions are recovered

### Requirement: Wrapping does not break HTML tags

`mdwrap` SHALL treat an HTML tag as a single unbreakable unit, never inserting a line break between a tag's name and its attributes or between two attributes.
A tag longer than the target width SHALL overflow onto its own line rather than being split.

#### Scenario: Tag with attributes is not split

- **WHEN** a paragraph containing a tag with several attributes is wrapped to a width narrower than the tag
- **THEN** the tag appears intact on one line

#### Scenario: Breaks still occur between tags

- **WHEN** a paragraph contains several tags and prose
- **THEN** line breaks fall at whitespace between tokens, and the text still wraps to the requested width where it can

#### Scenario: Comparison operators are not treated as tags

- **WHEN** the text contains `a < b` or a similar angle bracket that opens no tag
- **THEN** wrapping is unaffected and the remainder of the paragraph is not absorbed

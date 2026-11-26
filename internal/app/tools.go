package app

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/internal/texttools"
	"github.com/helloxz/zpad/locales"
)

// textTool is one transformation of the selected text, or of the whole text
// for the JSON tools on a JSON document.
type textTool string

const (
	toolBase64Encode textTool = "base64-encode"
	toolBase64Decode textTool = "base64-decode"
	toolMD5Lower32   textTool = "md5-32-lower"
	toolMD5Upper32   textTool = "md5-32-upper"
	toolMD5Lower16   textTool = "md5-16-lower"
	toolMD5Upper16   textTool = "md5-16-upper"
	toolJSONFormat   textTool = "json-format"
	toolJSONMinify   textTool = "json-minify"
)

// maxTextToolBytes is the most text one tool may process, across all
// selections. A tool copies what it converts, and formatting JSON can write
// several times the input, so a whole-file selection would spike the heap.
const maxTextToolBytes = texttools.MaxInput

// maxWholeJSONBytes is the most text a whole-document JSON tool works on. It
// is larger than maxTextToolBytes because a JSON file is what those tools
// most often act on and formatting streams in a few milliseconds per
// megabyte; the formatted output is still bounded by texttools.MaxJSONOutput.
const maxWholeJSONBytes = 8 << 20

// runTextTool transforms the selections of tab with tool. A JSON tool on a
// JSON document with nothing selected formats or minifies the whole text. With
// no selection otherwise, or one over the limit, it says so and changes
// nothing.
func (a *App) runTextTool(tab *Tab, tool textTool) {
	if isJSONTool(tool) && jsonWholeDoc(tab) && selectionBytes(tab) == 0 {
		a.runJSONWholeDoc(tab, tool)
		return
	}
	if !a.toolSelection(tab) {
		return
	}
	a.applyToolSelection(tab, func(text string, start int) (string, error) {
		return applyTextTool(tab, tool, text, start)
	})
}

// isJSONTool reports whether tool formats or minifies JSON.
func isJSONTool(tool textTool) bool {
	return tool == toolJSONFormat || tool == toolJSONMinify
}

// jsonWholeDoc reports whether tab is a JSON document a whole-document tool
// may rewrite. The language says so for a file's name and for the status
// bar's choice alike; a read-only document and a Markdown preview cannot be
// changed.
func jsonWholeDoc(tab *Tab) bool {
	if tab == nil || tab.doc.ReadOnly || tab.preview {
		return false
	}
	return tab.doc.Language != nil && tab.doc.Language.Name == "JSON"
}

// runJSONWholeDoc formats or minifies the whole text of a JSON document, which
// is what a JSON tool does when nothing is selected. The replacement is one
// step of undo. The cursor keeps its line and column as far as the new text
// has them, the view keeps its scroll, and the language is applied again
// because rewriting can make a long line short enough to highlight, or a whole
// value one long line.
func (a *App) runJSONWholeDoc(tab *Tab, tool textTool) {
	buf := tab.doc.Buffer
	n := buf.Len()
	if n > maxWholeJSONBytes {
		a.SetErrorToast(locales.TextToolTooLarge(), formatSize(int64(n))+" / "+formatSize(maxWholeJSONBytes))
		return
	}
	text := buf.Slice(0, n)
	out, err := applyTextTool(tab, tool, text, 0)
	if err != nil {
		a.SetErrorToast(textToolErrorTitle(err), err.Error())
		return
	}
	if out == text {
		return
	}
	cursor := tab.state.CursorPos()
	scroll := tab.state.ScrollY()
	buf.Set(out)
	tab.state.SetSelections(cursorAt(buf, cursor))
	tab.state.SetScrollY(scroll)
	tab.doc.applyLanguage(tab.doc.Language, tab.doc.Name())
	a.updateTitle()
	a.update()
}

// cursorAt returns an empty selection at p, with its line and column clamped
// to what buf still has.
func cursorAt(buf *editor.Buffer, p editor.Pos) editor.Selection {
	line := min(max(p.Line, 0), buf.LineCount()-1)
	col := min(max(p.Col, 0), len(buf.Line(line)))
	offset := buf.Offset(editor.Pos{Line: line, Col: col})
	return editor.Selection{Anchor: offset, Head: offset}
}

// lineNumberMinWidth is the fewest digits a line number gets: numbers below
// 10 are written as 01 to 09, as playlists and printed lists expect. A text
// with more lines than that grows to the digits it needs.
const lineNumberMinWidth = 2

// runLineNumbers numbers the selected lines in document order, padded to
// lineNumberMinWidth digits and followed by sep, so the lines of a playlist
// read 01, 02, 03. Numbers run on across selections, and the whole change is
// one step of undo.
func (a *App) runLineNumbers(tab *Tab, sep string) {
	if !a.toolSelection(tab) {
		return
	}
	first, width := lineNumberPlan(tab)
	a.applyToolSelection(tab, func(text string, start int) (string, error) {
		n, ok := first[start]
		if !ok {
			n = 1
		}
		return texttools.NumberLines(text, sep, n, width), nil
	})
}

// lineNumberPlan numbers the lines of every selection in document order and
// returns the number each selection's first line gets, keyed by the
// selection's start offset, and the digits every number is written with, at
// least lineNumberMinWidth. It reads the selections as they are, so the
// offsets are the ones the edits see.
func lineNumberPlan(tab *Tab) (map[int]int, int) {
	type span struct{ start, end int }
	spans := make([]span, 0, 2)
	for _, s := range tab.state.Selections() {
		if s.Start() < s.End() {
			spans = append(spans, span{s.Start(), s.End()})
		}
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	first := make(map[int]int, len(spans))
	total := 0
	for _, sp := range spans {
		first[sp.start] = total + 1
		// Counting needs the text, which the caller has already bounded
		// by maxTextToolBytes.
		total += texttools.CountLines(tab.doc.Buffer.Slice(sp.start, sp.end))
	}

	width := len(strconv.Itoa(total))
	if width < lineNumberMinWidth {
		width = lineNumberMinWidth
	}
	return first, width
}

// toolSelection reports whether a text tool may run on tab, showing the
// reason it may not as a toast. A read-only or Markdown preview tab refuses
// quietly: its tools are disabled in the menu.
func (a *App) toolSelection(tab *Tab) bool {
	if tab == nil || tab.doc.ReadOnly || tab.preview {
		return false
	}
	n := selectionBytes(tab)
	if n == 0 {
		a.SetErrorToast(locales.TextToolNoSelection(), "")
		return false
	}
	if n > maxTextToolBytes {
		a.SetErrorToast(locales.TextToolTooLarge(), formatSize(int64(n))+" / "+formatSize(maxTextToolBytes))
		return false
	}
	return true
}

// applyToolSelection rewrites each selection with fn and marks the window
// changed when anything was.
func (a *App) applyToolSelection(tab *Tab, fn func(text string, start int) (string, error)) {
	count, err := transformSelections(tab, fn)
	if err != nil {
		a.SetErrorToast(textToolErrorTitle(err), err.Error())
		return
	}
	if count == 0 {
		return
	}
	a.updateTitle()
	a.update()
}

// selectionBytes counts the selected UTF-8 bytes of a tab.
func selectionBytes(tab *Tab) int {
	total := 0
	for _, s := range tab.state.Selections() {
		total += s.End() - s.Start()
	}
	return total
}

func applyTextTool(tab *Tab, tool textTool, text string, start int) (string, error) {
	switch tool {
	case toolBase64Encode:
		return texttools.EncodeBase64(text), nil
	case toolBase64Decode:
		return texttools.DecodeBase64(text)
	case toolMD5Lower32:
		return texttools.MD5(text, texttools.MD5Hex32Lower), nil
	case toolMD5Upper32:
		return texttools.MD5(text, texttools.MD5Hex32Upper), nil
	case toolMD5Lower16:
		return texttools.MD5(text, texttools.MD5Hex16Lower), nil
	case toolMD5Upper16:
		return texttools.MD5(text, texttools.MD5Hex16Upper), nil
	case toolJSONFormat:
		out, err := texttools.FormatJSON(text)
		if err != nil {
			return "", err
		}
		return alignToLine(tab.doc.Buffer, start, out), nil
	case toolJSONMinify:
		return texttools.MinifyJSON(text)
	}
	return text, nil
}

// alignToLine indents the lines of a formatted value after the first to
// the indentation the selection starts at, so formatting a value inside a
// larger document does not send it back to column zero.
func alignToLine(buf *editor.Buffer, start int, text string) string {
	if !strings.Contains(text, "\n") {
		return text
	}
	prefix := lineIndent(buf, start)
	if prefix == "" {
		return text
	}
	return strings.ReplaceAll(text, "\n", "\n"+prefix)
}

// lineIndent returns the whitespace before offset on its line, empty when
// the text before it is not all whitespace.
func lineIndent(buf *editor.Buffer, offset int) string {
	p := buf.Pos(offset)
	line := buf.Line(p.Line)
	if p.Col > len(line) {
		p.Col = len(line)
	}
	prefix := line[:p.Col]
	for i := 0; i < len(prefix); i++ {
		if prefix[i] != ' ' && prefix[i] != '\t' {
			return ""
		}
	}
	return prefix
}

func textToolErrorTitle(err error) string {
	switch {
	case errors.Is(err, texttools.ErrNotText):
		return locales.TextToolNotText()
	case errors.Is(err, texttools.ErrEmpty):
		return locales.TextToolEmpty()
	case errors.Is(err, texttools.ErrJSONDepth):
		return locales.TextToolJSONDeep()
	case errors.Is(err, texttools.ErrJSONOutput):
		return locales.TextToolJSONLarge()
	case errors.Is(err, texttools.ErrJSON):
		return locales.TextToolJSONInvalid()
	}
	return locales.TextToolFailed()
}

// toolMenu fills m with the tools that work on the selected text, or on the
// whole text for JSON, disabled where tab cannot run them. It is the editor's
// own 工具 submenu too, so it holds nothing that acts on the window: the text
// comparison is added by compareToolMenu for the toolbar alone. Case
// conversion is not here: the editor's own context menu has a richer Transform
// submenu, and the app menu keeps Convert Case.
func (a *App) toolMenu(m *ui.Menu, tab *Tab) {
	editable := tab != nil && !tab.doc.ReadOnly && !tab.preview
	selected := 0
	if editable {
		selected = selectionBytes(tab)
	}
	tooLarge := selected > maxTextToolBytes
	hasText := selected > 0

	// A JSON document with nothing selected works as a whole, so its tools
	// stay enabled: that is the one tool that needs no selection.
	wholeJSON := !hasText && jsonWholeDoc(tab)
	wholeSize := 0
	if wholeJSON {
		wholeSize = tab.doc.Buffer.Len()
	}
	jsonReady := hasText && !tooLarge || wholeJSON && wholeSize > 0 && wholeSize <= maxWholeJSONBytes

	m.Submenu(locales.Base64Group(), func(mm *ui.Menu) {
		if mm.Item(locales.Base64Encode()).Disabled(!hasText || tooLarge).Chosen() {
			a.runTextTool(tab, toolBase64Encode)
		}
		if mm.Item(locales.Base64Decode()).Disabled(!hasText || tooLarge).Chosen() {
			a.runTextTool(tab, toolBase64Decode)
		}
	}).Disabled(!editable)

	m.Submenu(locales.Md5Group(), func(mm *ui.Menu) {
		if mm.Item(locales.Md5Lower32()).Disabled(!hasText || tooLarge).Chosen() {
			a.runTextTool(tab, toolMD5Lower32)
		}
		if mm.Item(locales.Md5Upper32()).Disabled(!hasText || tooLarge).Chosen() {
			a.runTextTool(tab, toolMD5Upper32)
		}
		if mm.Item(locales.Md5Lower16()).Disabled(!hasText || tooLarge).Chosen() {
			a.runTextTool(tab, toolMD5Lower16)
		}
		if mm.Item(locales.Md5Upper16()).Disabled(!hasText || tooLarge).Chosen() {
			a.runTextTool(tab, toolMD5Upper16)
		}
	}).Disabled(!editable)

	m.Submenu(locales.JsonGroup(), func(mm *ui.Menu) {
		if mm.Item(locales.JsonFormat()).Disabled(!jsonReady).Chosen() {
			a.runTextTool(tab, toolJSONFormat)
		}
		if mm.Item(locales.JsonMinify()).Disabled(!jsonReady).Chosen() {
			a.runTextTool(tab, toolJSONMinify)
		}
	}).Disabled(!editable)

	m.Submenu(locales.LineNumberGroup(), func(mm *ui.Menu) {
		if mm.Item(locales.LineNumberComma()).Disabled(!hasText || tooLarge).Chosen() {
			a.runLineNumbers(tab, ",")
		}
		if mm.Item(locales.LineNumberDot()).Disabled(!hasText || tooLarge).Chosen() {
			a.runLineNumbers(tab, ". ")
		}
		if mm.Item(locales.LineNumberParen()).Disabled(!hasText || tooLarge).Chosen() {
			a.runLineNumbers(tab, ") ")
		}
		if mm.Item(locales.LineNumberColon()).Disabled(!hasText || tooLarge).Chosen() {
			a.runLineNumbers(tab, ": ")
		}
	}).Disabled(!editable)
}

// compareToolMenu is the toolbar's tools menu: the window-level text
// comparison first, a rule, then the text tools. The editor's own 工具
// submenu stays on toolMenu alone, since a dialog about two tabs is not a
// selection tool.
func (a *App) compareToolMenu(m *ui.Menu, tab *Tab) {
	if m.Item(locales.CompareText()).Chosen() {
		a.openCompareText(tab)
	}
	m.Separator()
	a.toolMenu(m, tab)
}

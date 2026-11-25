package app

import (
	"slices"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/locales"
)

// The most text one side of a comparison may hold. The viewer copies both
// texts into buffers of its own, and this is an add-on: a pair this large
// already says more than the panel needs. Keep locales.CompareLimit in step
// with these.
const (
	diffMaxBytes = 1 << 20
	diffMaxLines = 20_000
)

// openCompareText shows the text comparison panel, with tab as side A when
// one is given, as from the editor's tools menu; the toolbar passes its
// active tab. B starts as the other pane's active document or the next one
// open, so the common case is open, then Compare.
func (a *App) openCompareText(tab *Tab) {
	if a.panelOpen() {
		return
	}
	docs := a.openDocuments()
	var current *Document
	if tab != nil {
		current = tab.doc
	} else if active := a.activeTab(); active != nil {
		current = active.doc
	}
	if current == nil && len(docs) > 0 {
		current = docs[0]
	}
	a.diffOpen = true
	a.diffA = current
	a.diffB = a.comparePeer(current)
	a.diffChanged()
	a.update()
}

// comparePeer picks side B: the other pane's active document when it is not
// current, else the next document open, wrapping, so two tabs side by side
// compare with one press. nil when current is the only one.
func (a *App) comparePeer(current *Document) *Document {
	docs := a.openDocuments()
	if len(docs) == 0 {
		return nil
	}
	if current == nil {
		current = docs[0]
	}
	if a.split {
		other := 1 - a.activePane
		if tab := a.pane(other).activeTab(); tab != nil && tab.doc != current {
			return tab.doc
		}
	}
	at := slices.Index(docs, current)
	for i := 1; i <= len(docs); i++ {
		if d := docs[(at+i+len(docs))%len(docs)]; d != current {
			return d
		}
	}
	return nil
}

// diffProblem says why the two sides cannot be compared, "" when they can.
// Empty text is refused: a blank untitled tab would only show every line
// added, and is nearly always a tab the user did not mean to pick.
func (a *App) diffProblem() string {
	if a.diffA == nil || a.diffB == nil {
		return locales.CompareNeedTwoTabs()
	}
	if a.diffA == a.diffB {
		return locales.CompareSameTab()
	}
	for _, doc := range []*Document{a.diffA, a.diffB} {
		if doc.Buffer.Len() == 0 {
			return locales.CompareEmptySide(doc.Name())
		}
		if doc.Buffer.Len() > diffMaxBytes || bufferTextLines(doc.Buffer) > diffMaxLines {
			return locales.CompareTooLarge(doc.Name(), locales.CompareLimit())
		}
	}
	return ""
}

// runCompare builds a diff of the two chosen documents, or says that they
// hold the same text. The viewer gets buffers of its own, so it never
// touches the documents and the comparison is of the moment Compare was
// pressed; Compare Again takes a new snapshot.
func (a *App) runCompare() {
	a.diff = nil
	a.diffIdentical = false
	a.diffFocus = false
	if a.diffProblem() != "" {
		a.update()
		return
	}
	oldText, newText := a.diffA.Buffer.String(), a.diffB.Buffer.String()
	if oldText == newText {
		a.diffIdentical = true
		a.update()
		return
	}
	oldBuf, newBuf := editor.NewBuffer(oldText), editor.NewBuffer(newText)
	if lang := a.diffA.Language; lang != nil {
		oldBuf.SetLanguage(lang)
	}
	if lang := a.diffB.Language; lang != nil {
		newBuf.SetLanguage(lang)
	}
	a.diff = &editor.DiffState{
		Old:        oldBuf,
		New:        newBuf,
		Layout:     editor.DiffUnified,
		Indicators: editor.DiffClassic,
		// Three lines of context, as git and code review tools show, with
		// the rest folded into separators.
		ContextLines:   3,
		KeepScrollbars: true,
		Wrap:           a.Settings.SoftWrap,
		Font:           ui.Font{Family: a.Settings.FontFamily, Size: a.Settings.FontSize},
		LineHeight:     a.Settings.LineHeight,
		TabSize:        a.Settings.TabSize,
		Theme:          a.themeDefinition().Code,
	}
	a.diffFocus = true
	a.update()
}

// swapCompare exchanges A and B, comparing again when a result showed.
func (a *App) swapCompare() {
	a.diffA, a.diffB = a.diffB, a.diffA
	if a.diff != nil || a.diffIdentical {
		a.runCompare()
		return
	}
	a.update()
}

// diffChanged forgets a result whose sides changed: the next Compare takes
// a new snapshot of the new pair.
func (a *App) diffChanged() {
	a.diff = nil
	a.diffIdentical = false
	a.diffFocus = false
}

// diffDocumentClosed takes a closed document out of the panel: the snapshot
// could still show it, but the names above it would point at a tab that is
// gone, so the user chooses again. The panel stays open.
func (a *App) diffDocumentClosed(doc *Document) {
	if !a.diffOpen || doc == nil {
		return
	}
	changed := false
	if a.diffA == doc {
		a.diffA = nil
		changed = true
	}
	if a.diffB == doc {
		a.diffB = nil
		changed = true
	}
	if changed {
		a.diffChanged()
		a.update()
	}
}

// diffLineCount is the line count the identical message names; both sides
// are the same text then.
func (a *App) diffLineCount() int {
	if a.diffA != nil {
		return bufferTextLines(a.diffA.Buffer)
	}
	if a.diffB != nil {
		return bufferTextLines(a.diffB.Buffer)
	}
	return 0
}

// bufferTextLines counts the lines of a text as the diff and git do: a text
// ending in a newline has no line after it, though the editor starts an
// empty one there.
func bufferTextLines(buf *editor.Buffer) int {
	n := buf.LineCount()
	if n > 0 && buf.Line(n-1) == "" {
		n--
	}
	return n
}

// findActive opens the find bar of what has the keyboard: the comparison
// when its panel is open, so Find works where the user is looking, else the
// active editor.
func (a *App) findActive() {
	if a.diffOpen {
		if a.diff != nil {
			a.diff.Run(editor.Find)
		}
		return
	}
	if tab := a.activeTab(); tab != nil && !tab.preview {
		tab.state.Run(editor.Find)
	}
}

package app

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/helloxz/zpad/locales"
)

// Search across the documents open in either pane. MyGo's find bar lives
// in one buffer, so the panel and its matches are the app's own: the scan
// reads buffer snapshots, which editor.Buffer allows from any goroutine,
// and the results jump back into the tab that shows the document.

const (
	// searchMaxPerFile and searchMaxTotal bound what one search keeps: the
	// list is for finding a place, not for reading every occurrence.
	searchMaxPerFile = 200
	searchMaxTotal   = 2000
)

// searchOptions are the toggles of the search panel.
type searchOptions struct {
	caseSensitive bool
	wholeWord     bool
	regex         bool
}

// textMatch is one match of a query in a text, in byte offsets.
type textMatch struct {
	start, end int
}

// searchHit is one match in one open document. start and end are buffer
// offsets, for the editor to select; version says which text it was found
// in, so a jump after an edit can look the match up again. preview is the
// line around it, with matchAt and matchLen marking the match in preview.
type searchHit struct {
	doc      *Document
	version  uint64
	start    int
	end      int
	line     int
	col      int
	preview  string
	matchAt  int
	matchLen int
}

// searchRow is one row of the results list: a file's header, or a match.
type searchRow struct {
	header bool
	doc    *Document
	count  int
	hit    searchHit
}

// searchInput is a document and the text it had when the scan started. A
// snapshot stays as it was while the buffer changes, so the scan runs off
// the main thread without locking the editor.
type searchInput struct {
	doc  *Document
	snap editor.Snapshot
}

// scanText finds the matches of query in text. It is pure, so the search
// rules are tested without a window. limit bounds the matches returned;
// one more than the caller keeps lets it tell that there were more.
func scanText(text, query string, opt searchOptions, limit int) ([]textMatch, error) {
	if query == "" || limit <= 0 {
		return nil, nil
	}
	var out []textMatch
	if opt.regex {
		pat := "(?m)" + query
		if !opt.caseSensitive {
			pat = "(?m)(?i)" + query
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return nil, err
		}
		for _, m := range re.FindAllStringIndex(text, limit) {
			if m[1] == m[0] {
				continue
			}
			if opt.wholeWord && !wholeWordAt(text, m[0], m[1]) {
				continue
			}
			out = append(out, textMatch{m[0], m[1]})
			if len(out) >= limit {
				break
			}
		}
		return out, nil
	}

	// A plain query, matching case: strings.Index walks the text once.
	if opt.caseSensitive {
		return plainMatches(text, query, query, opt.wholeWord, limit, strings.Index), nil
	}

	// Case-insensitive ASCII: fold bytes as the scan goes, without a copy
	// of the whole text to lower.
	if isASCII(query) {
		return plainMatches(text, query, asciiLower(query), opt.wholeWord, limit,
			func(text, q string) int {
				n := len(q)
				for i := 0; i+n <= len(text); i++ {
					if equalFoldASCII(text[i:i+n], q) {
						return i
					}
				}
				return -1
			}), nil
	}

	// Other letters have no simple byte fold: quote them into a
	// case-insensitive expression, which RE2 matches by Unicode case.
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	for _, m := range re.FindAllStringIndex(text, limit) {
		if m[1] == m[0] {
			continue
		}
		if opt.wholeWord && !wholeWordAt(text, m[0], m[1]) {
			continue
		}
		out = append(out, textMatch{m[0], m[1]})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// plainMatches walks text with index, which finds needle in a text, and
// keeps the matches that are whole words when opt asks.
func plainMatches(text, query, needle string, wholeWords bool, limit int, index func(text, needle string) int) []textMatch {
	var out []textMatch
	for at := 0; at+len(needle) <= len(text); {
		i := index(text[at:], needle)
		if i < 0 {
			break
		}
		a := at + i
		b := a + len(query)
		if !wholeWords || wholeWordAt(text, a, b) {
			out = append(out, textMatch{a, b})
			if len(out) >= limit {
				break
			}
		}
		at = b
		if b == a {
			at++
		}
	}
	return out
}

// wholeWordAt reports whether the match a to b is a whole word: a letter,
// a digit or an underscore on neither side.
func wholeWordAt(text string, a, b int) bool {
	if a > 0 {
		r, _ := utf8.DecodeLastRuneInString(text[:a])
		if isWordRune(r) {
			return false
		}
	}
	if b < len(text) {
		r, _ := utf8.DecodeRuneInString(text[b:])
		if isWordRune(r) {
			return false
		}
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// asciiLower lowers the ASCII letters of s, keeping its length, as the
// scan compares bytes.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func equalFoldASCII(s, lower string) bool {
	if len(s) != len(lower) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if c != lower[i] {
			return false
		}
	}
	return true
}

// scanDocuments scans the snapshots in order, grouping the matches by
// document: a header row, then its matches. It stops at the total cap or
// when cancel closes, and reports whether it left matches out.
func scanDocuments(inputs []searchInput, query string, opt searchOptions, cancel <-chan struct{}) (rows []searchRow, files int, err error, truncated bool) {
	total := 0
	for i, in := range inputs {
		if cancelled(cancel) {
			return rows, files, nil, truncated
		}
		if total >= searchMaxTotal {
			truncated = true
			break
		}
		remaining := min(searchMaxPerFile, searchMaxTotal-total)
		text := in.snap.String()
		matches, scanErr := scanText(text, query, opt, remaining+1)
		if scanErr != nil {
			return rows, files, scanErr, truncated
		}
		if len(matches) == 0 {
			continue
		}
		if len(matches) > remaining {
			matches = matches[:remaining]
			truncated = true
		}
		files++
		rows = append(rows, searchRow{header: true, doc: in.doc, count: len(matches)})
		for _, m := range matches {
			rows = append(rows, searchRow{hit: makeHit(in.doc, in.snap, m)})
		}
		total += len(matches)
		if total >= searchMaxTotal && i < len(inputs)-1 {
			truncated = true
		}
	}
	return rows, files, nil, truncated
}

func cancelled(cancel <-chan struct{}) bool {
	select {
	case <-cancel:
		return true
	default:
		return false
	}
}

// makeHit describes one match of a document: the line it is on, and the
// line clipped around it for the panel.
func makeHit(doc *Document, snap editor.Snapshot, m textMatch) searchHit {
	line := snap.LineOf(m.start)
	lineStart := snap.LineStart(line)
	col := m.start - lineStart
	preview, at, n := previewLine(snap.Line(line), col, m.end-m.start)
	return searchHit{
		doc:      doc,
		version:  snap.Version(),
		start:    m.start,
		end:      m.end,
		line:     line,
		col:      col,
		preview:  preview,
		matchAt:  at,
		matchLen: n,
	}
}

// previewLine clips line around the match at col, length bytes long: at
// most about 240 bytes of it, starting a little before the match. It
// replaces tabs with spaces and adds an ellipsis at either cut end. The
// returned offset and length mark the match inside the preview.
func previewLine(line string, col, length int) (preview string, at, n int) {
	const maxBytes = 240
	col = max(0, min(col, len(line)))
	length = max(0, min(length, len(line)-col))
	start := 0
	if col > 80 {
		start = col - 60
		for start > 0 && !utf8.RuneStart(line[start]) {
			start--
		}
	}
	end := min(len(line), start+maxBytes)
	if end < col+length {
		// A match longer than the preview brings its own end.
		end = min(len(line), col+length)
	}
	for end > col+length && end < len(line) && !utf8.RuneStart(line[end]) {
		end--
	}
	before := expandTabs(line[start:col])
	match := expandTabs(line[col : col+length])
	after := expandTabs(line[col+length : end])
	var b strings.Builder
	b.WriteString(before)
	b.WriteString(match)
	b.WriteString(after)
	preview = b.String()
	at = len(before)
	if start > 0 {
		preview = "…" + preview
		at += len("…")
	}
	if end < len(line) {
		preview += "…"
	}
	return preview, at, len(match)
}

func expandTabs(s string) string {
	if !strings.Contains(s, "\t") {
		return s
	}
	return strings.ReplaceAll(s, "\t", "    ")
}

// splitPreview cuts a hit's preview into the text before the match, the
// match, and the text after it, for the row to highlight the middle.
func splitPreview(h searchHit) (string, string, string) {
	at := max(0, min(h.matchAt, len(h.preview)))
	end := max(at, min(at+h.matchLen, len(h.preview)))
	return h.preview[:at], h.preview[at:end], h.preview[end:]
}

// openSearchPanel shows the panel, focused. Nothing is searched until the
// user asks, with Enter or the search button.
func (a *App) openSearchPanel() {
	if a.showSettings || a.aboutOpen || a.renameOpen || a.snippetsOpen ||
		a.snippetEditOpen || a.groupEditOpen {
		return
	}
	a.searchOpen = true
	a.searchFocus = true
	a.update()
}

// closeSearchPanel hides the panel and keeps the query and the results,
// so opening it again shows where the user was. The clear button, not
// Escape, is what forgets them.
func (a *App) closeSearchPanel() {
	if !a.searchOpen {
		return
	}
	a.searchOpen = false
	a.update()
}

// clearSearchInput empties the field and forgets the results, as the clear
// button does; opening the panel again then starts from nothing.
func (a *App) clearSearchInput() {
	a.searchQuery = ""
	a.searchInputChanged()
	a.searchFocus = true
}

// searchOptionsNow returns the toggles as they are now, to compare with
// the ones the rows were found with.
func (a *App) searchOptionsNow() searchOptions {
	return searchOptions{
		caseSensitive: a.searchCase,
		wholeWord:     a.searchWord,
		regex:         a.searchRegex,
	}
}

// searchFresh reports whether the rows stand for the query and toggles
// the panel shows now.
func (a *App) searchFresh() bool {
	return a.searchQuery != "" && a.searchQuery == a.searchRanFor &&
		a.searchOptionsNow() == a.searchRanOptions
}

// searchInputChanged drops results that no longer match what the panel
// shows, as typing or a toggle does, and waits for Enter or the search
// button.
func (a *App) searchInputChanged() {
	a.stopSearch()
	a.searchList = ui.ListState{}
	a.searchRows = nil
	a.searchFiles = 0
	a.searchSel = -1
	a.searchBusy = false
	a.searchErr = ""
	a.searchTruncated = false
	a.searchRanFor = ""
	a.searchRanOptions = searchOptions{}
}

// submitSearch is Enter: it searches a query or toggles not searched yet,
// and opens the chosen match when the rows are current.
func (a *App) submitSearch() {
	if a.searchQuery == "" {
		return
	}
	if !a.searchFresh() {
		a.startSearch()
		return
	}
	if hit, ok := a.selectedSearchHit(); ok {
		a.revealHit(hit)
		return
	}
	a.startSearch()
}

// startSearch scans the open documents now. It reads the snapshots it
// takes, off the main thread; only the newest scan's results are kept.
func (a *App) startSearch() {
	a.stopSearch()
	a.searchGen++
	gen := a.searchGen
	query := a.searchQuery
	opt := a.searchOptionsNow()
	if query == "" {
		a.searchRows = nil
		a.searchFiles = 0
		a.searchErr = ""
		a.searchTruncated = false
		a.searchBusy = false
		a.searchSel = -1
		return
	}
	docs := a.openDocuments()
	inputs := make([]searchInput, 0, len(docs))
	for _, doc := range docs {
		inputs = append(inputs, searchInput{doc: doc, snap: doc.Buffer.Snapshot()})
	}
	cancel := make(chan struct{})
	a.searchCancel = cancel
	a.searchBusy = true
	a.searchErr = ""
	a.searchTruncated = false
	// Results of the query before this one would jump to places that no
	// longer match; show the scan on its way instead.
	a.searchRows = nil
	a.searchFiles = 0
	a.searchSel = -1
	run := func() {
		rows, files, err, truncated := scanDocuments(inputs, query, opt, cancel)
		a.onUI(func() {
			if gen != a.searchGen {
				return
			}
			a.searchCancel = nil
			a.searchBusy = false
			rows = a.filterOpenRows(rows)
			files = 0
			for _, row := range rows {
				if row.header {
					files++
				}
			}
			a.searchRows = rows
			a.searchFiles = files
			a.searchTruncated = truncated
			a.searchSel = firstHitRow(rows)
			a.searchRanFor = query
			a.searchRanOptions = opt
			if err != nil {
				a.searchErr = locales.SearchInvalidRegex()
				a.searchRows = nil
				a.searchFiles = 0
				a.searchSel = -1
			}
		})
	}
	if a.win == nil {
		// Tests have no window to marshal through; scan now, so a frame
		// shows the results the click or the key asked for.
		run()
		return
	}
	go run()
}

// stopSearch drops the scan on its way, if any: its generation is old, so
// its results are ignored when they arrive.
func (a *App) stopSearch() {
	a.searchGen++
	if a.searchCancel != nil {
		close(a.searchCancel)
		a.searchCancel = nil
	}
}

// filterOpenRows drops the groups of documents closed since the scan took
// its snapshots, so no result leads to a tab that is gone.
func (a *App) filterOpenRows(rows []searchRow) []searchRow {
	open := make(map[*Document]bool)
	for _, doc := range a.openDocuments() {
		open[doc] = true
	}
	out := rows[:0]
	removing := false
	for _, row := range rows {
		if row.header {
			removing = !open[row.doc]
		}
		if removing {
			continue
		}
		out = append(out, row)
	}
	return out
}

func firstHitRow(rows []searchRow) int {
	for i, row := range rows {
		if !row.header {
			return i
		}
	}
	return -1
}

// selectedSearchHit returns the match the list chooses, or the first one
// when the list has no choice.
func (a *App) selectedSearchHit() (searchHit, bool) {
	if a.searchSel >= 0 && a.searchSel < len(a.searchRows) && !a.searchRows[a.searchSel].header {
		return a.searchRows[a.searchSel].hit, true
	}
	for _, row := range a.searchRows {
		if !row.header {
			return row.hit, true
		}
	}
	return searchHit{}, false
}

// revealHit shows the document of a match, selects it, and closes the
// panel. A document edited since the scan is searched again, and the match
// nearest the line the results showed is taken.
func (a *App) revealHit(h searchHit) {
	if !a.isOpenDocument(h.doc) {
		a.closeSearchPanel()
		return
	}
	a.activateDocument(h.doc)
	tab := a.tabForDocument(h.doc)
	if tab == nil {
		a.closeSearchPanel()
		return
	}
	if tab.preview {
		tab.preview = false
	}
	start, end, ok := a.matchFor(h)
	if ok {
		tab.state.Select(start, end)
	}
	a.focusEditor = true
	a.closeSearchPanel()
}

func (a *App) isOpenDocument(doc *Document) bool {
	for _, d := range a.openDocuments() {
		if d == doc {
			return true
		}
	}
	return false
}

// tabForDocument returns the tab showing doc, the active pane's first.
func (a *App) tabForDocument(doc *Document) *Tab {
	for _, pane := range []int{a.activePane, 1 - a.activePane} {
		if pane == 1 && !a.split {
			continue
		}
		for _, tab := range a.pane(pane).tabs {
			if tab.doc == doc {
				return tab
			}
		}
	}
	return nil
}

// matchFor returns where a hit is now: as found, when the text is as the
// scan left it, else the nearest match of the query in the text it has.
func (a *App) matchFor(h searchHit) (start, end int, ok bool) {
	snap := h.doc.Buffer.Snapshot()
	if snap.Version() == h.version {
		return h.start, h.end, true
	}
	opt := searchOptions{
		caseSensitive: a.searchCase,
		wholeWord:     a.searchWord,
		regex:         a.searchRegex,
	}
	matches, err := scanText(snap.String(), a.searchQuery, opt, 1000)
	if err != nil || len(matches) == 0 {
		return 0, 0, false
	}
	best, bestDist := matches[0], absInt(snap.LineOf(matches[0].start)-h.line)
	for _, m := range matches[1:] {
		if d := absInt(snap.LineOf(m.start) - h.line); d < bestDist {
			best, bestDist = m, d
		}
	}
	return best.start, best.end, true
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// pruneSearchDocument drops the rows of a document that closed while the
// panel was open, so no result leads to a tab that is gone.
func (a *App) pruneSearchDocument(doc *Document) {
	if doc == nil || len(a.searchRows) == 0 {
		return
	}
	if a.isOpenDocument(doc) {
		return
	}
	out := a.searchRows[:0]
	removing := false
	for _, row := range a.searchRows {
		if row.header {
			removing = row.doc == doc
		}
		if removing {
			continue
		}
		out = append(out, row)
	}
	a.searchRows = out
	a.searchFiles = 0
	for _, row := range a.searchRows {
		if row.header {
			a.searchFiles++
		}
	}
	a.searchSel = firstHitRow(a.searchRows)
	if a.searchOpen {
		a.update()
	}
}

package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/jeffwilliams/basejump/diff"
	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

type Basejump struct {
	P *plugin.Plugin
}

func (n Basejump) nvim() *nvim.Nvim {
	return n.P.Nvim
}

// Echom formats it's arguments using fmt.Sprintf, then performs the echom command with the resulting
// string. Basically a printf to vim's status line and stores it in vim's messages.
func (n Basejump) Echom(fmts string, args ...interface{}) {
	s := fmt.Sprintf(fmts, args...)
	s = strings.Replace(s, "'", "''", -1)
	n.P.Nvim.Command(fmt.Sprintf(":echom '%s'", s))
}

// Selection returns the coordinates of the current selection, if it is within a single line
// Note that if we are not in visual mode, this returns the last selection in visual mode.
func (n Basejump) Selection() (startLine, startCol, endLine, endCol int, err error) {
	result := make([]float32, 4)
	nv := n.nvim()

	err = nv.Call("getpos", result, "'<")
	if err != nil {
		return
	}

	startLine = int(result[1])
	startCol = int(result[2])

	err = nv.Call("getpos", result, "'>")
	if err != nil {
		return
	}

	endLine = int(result[1])
	endCol = int(result[2])

	return
}

// Return the current line and column of the cursor
func (n Basejump) Cursor() (line, col int, err error) {
	result := make([]float32, 4)
	nv := n.nvim()

	err = nv.Call("getpos", result, ".")
	if err != nil {
		return
	}

	line = int(result[1])
	col = int(result[2])
	return
}

// SelectionText returns the text contained in the current selection.
func (n Basejump) SelectionText() (text string, err error) {
	nv := n.nvim()

	var startLine, startCol, endLine, endCol int
	startLine, startCol, endLine, endCol, err = n.Selection()
	if err != nil {
		return
	}

	trace(n, "selected text is from line %d col %d to line %d col %d", startLine, startCol, endLine, endCol)

	var buf nvim.Buffer
	buf, err = nv.CurrentBuffer()
	if err != nil {
		return
	}

	var lines [][]byte
	// Indexing is zero-based, end-exclusive
	lines, err = nv.BufferLines(buf, startLine-1, endLine, true)
	if err != nil {
		return
	}

	var bbuf bytes.Buffer

	if len(lines) == 1 {
		bbuf.Write(lines[0][startCol-1 : endCol])
	} else if len(lines) > 1 {
		bbuf.Write(lines[0][startCol-1 : len(lines[0])])
		for i := 1; i < len(lines)-1; i++ {
			bbuf.Write(lines[i])
		}
		bbuf.Write(lines[len(lines)-1][0:endCol])
	}

	text = bbuf.String()
	return
}

// CurrentWordText returns the current word under the cursor
func (n Basejump) CurrentWordText() (text string, err error) {
	nv := n.nvim()
	err = nv.Call("expand", &text, "<cWORD>")
	return
}

func (n Basejump) CurrentLineNumber() (num int, err error) {
	nv := n.nvim()
	err = nv.Call("line", &num, ".")
	return
}

func (n Basejump) CurrentLineText() (text string, err error) {
	nv := n.nvim()
	err = nv.Call("getline", &text, ".")
	return
}

func (n Basejump) LineText(line int) (text string, err error) {
	nv := n.nvim()
	err = nv.Call("getline", &text, line)
	return
}

var pathRegex = regexp.MustCompile(`^([^:]+)(?::(\d+))?(?::(\d+))?`)

// ParsePath parses `text` into a filesystem path, line, and column. The `text`
// parameter must have one of the formats:
//
//    <path>								(for example file.go, or /bin/bash)
//    <path>:<line>					(for example file.go:100)
//    <path>:<line>:<col>		(for example file.go:100:20)
//
// If the parsed path is not absolute it is made absolute by prepending the
// cwd of the current window in vim.
//
// If line and or col is missing, they are set to 0.
func (n Basejump) ParsePath(text string) (fpath string, line, col int, err error) {
	text = strings.TrimSpace(text)

	match := pathRegex.FindStringSubmatch(text)
	if match == nil || len(match) < 2 {
		err = fmt.Errorf("doesn't seem to be a valid path")
		return
	}
	fpath = match[1]
	if len(match) > 2 && match[2] != "" {
		line, err = strconv.Atoi(match[2])
		if err != nil {
			return
		}
	}
	if len(match) > 3 && match[3] != "" {
		col, err = strconv.Atoi(match[3])
		if err != nil {
			return
		}
	}

	fpath, err = n.AbsPath(fpath)
	if err != nil {
		return
	}
	return
}

// AbsPath makes the path `fpath` absolute if it is not by prepending
// the working directory of the current window.
func (n Basejump) AbsPath(fpath string) (result string, err error) {
	return n.AbsPathRelWindow(fpath, -1)
}

func (n Basejump) pidCwd(pid int) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
}

// AbsPathRelWindow makes the path `fpath` absolute if it is not by prepending
// the working directory of the window `window`.
func (n Basejump) AbsPathRelWindow(fpath string, window int) (result string, err error) {
	nv := n.nvim()

	result = fpath
	if !path.IsAbs(fpath) {

		var cwd string

		if window == -1 {
			// If the current window is a terminal window, then we need
			// to get the cwd in a special way. We can't use this method for
			// the non-current window.
			var b bool
			err = nv.Call("exists", &b, "b:term_title")
			if err != nil {
				return
			}
			if b {
				// Is a terminal.
				var pid float32
				var buf nvim.Buffer
				buf, err = nv.CurrentBuffer()
				if err != nil {
					return
				}
				err = nv.BufferVar(buf, "terminal_job_pid", &pid)
				if err != nil {
					return
				}

				cwd, err = n.pidCwd(int(pid))
				if err != nil {
					return
				}
			}

		}

		if cwd == "" {

			args := make([]interface{}, 0, 1)
			if window != -1 {
				args = append(args, window)
			}

			err = nv.Call("getcwd", &cwd, args...)
			if err != nil {
				return
			}
		}
		result = cwd + "/" + fpath
	}
	return
}

type SearchType int

const (
	SearchOnlyInCurrentTab = iota
	SearchInAllTabs
)

func (n Basejump) findWindow(fpath string, srchType SearchType) (win *nvim.Window, tabNr, winNr int, err error) {
	nv := n.nvim()

	defer func() {
		if win != nil {
			trace(n, "trace: findWindow: `%s` found in tab %d window %d", fpath, tabNr, winNr)
		} else {
			trace(n, "trace: findWindow: `%s` not found in open windows", fpath)
		}
	}()

	nfpath, err := n.AbsPath(fpath)
	if err == nil {
		fpath = nfpath
	}

	findInList := func(wins []nvim.Window) {
		for _, cwin := range wins {
			var buf nvim.Buffer
			var bufFileName string

			buf, err = nv.WindowBuffer(cwin)
			if err != nil {
				trace(n, "trace: findWindow: WindowBuffer(%d) failed: %v", cwin, err)
				return
			}

			bufFileName, err = nv.BufferName(buf)
			if err != nil {
				trace(n, "trace: findWindow: BufferName failed: %v", err)
				return
			}

			winNr, err = nv.WindowNumber(cwin)
			if err != nil {
				trace(n, "trace: findWindow: WindowNumber(%d) failed: %v", cwin, err)
				return
			}

			bufFileName, err = n.AbsPathRelWindow(bufFileName, winNr)
			if err != nil {
				trace(n, "trace: findWindow: n.AbsPathRelWindow failed: %v", err)
				return
			}

			trace(n, "trace: findWindow: '%s' vs '%s'", path.Clean(bufFileName), path.Clean(fpath))
			if path.Clean(bufFileName) == path.Clean(fpath) {
				win = &cwin
				return
			}
		}

		return
	}

	if srchType == SearchOnlyInCurrentTab {
		trace(n, "trace: findWindow: searching for `%s` in current tab's windows", fpath)
		var wins []nvim.Window
		wins, err = nv.Windows()
		if err != nil {
			return
		}
		findInList(wins)
	} else {
		trace(n, "trace: findWindow: searching for `%s` in all tabs windows", fpath)
		var tabs []nvim.Tabpage
		tabs, err = nv.Tabpages()
		if err != nil {
			return
		}

		// Move the current tab to the first, so that if the window is in the
		// current tabpage and another one, we prefer the local one
		var curTab nvim.Tabpage
		curTab, err = nv.CurrentTabpage()
		if err != nil {
			return
		}
		for i, tab := range tabs {
			if tab == curTab && i > 0 {
				t := tabs[0]
				tabs[0] = tabs[i]
				tabs[i] = t
				break
			}
		}

		for _, tab := range tabs {
			var wins []nvim.Window
			wins, err = nv.TabpageWindows(tab)
			if err != nil {
				return
			}

			tabNr, err = nv.TabpageNumber(tab)
			if err != nil {
				return
			}

			findInList(wins)
			if err != nil || win != nil {
				return
			}
		}
	}

	return
}

// OpenOrChangeTo ensures the specified file is open in vim. If the path is found in a
// window, that window is made current. If no window contains that path, it is split and
// opened.
func (n Basejump) OpenOrChangeTo(fpath, method string) (wasOpen bool, err error) {
	nv := n.nvim()

	//win, winNr, err := n.findWindow(fpath, SearchOnlyInCurrentTab)
	win, tabNr, winNr, err := n.findWindow(fpath, SearchInAllTabs)
	if err != nil {
		return
	}

	if win != nil {
		// Change to this window
		trace(n, "trace: SplitOrChangeTo: changing to tab")
		nv.Command(fmt.Sprintf("%dtabnext", tabNr))

		trace(n, "trace: SplitOrChangeTo: changing to existing window")
		err = nv.Command(fmt.Sprintf("%dwincmd w", winNr))
		wasOpen = true
		return
	}

	// Not found. Split new window
	// Seems like :split is not working from a script for directories for me
	// (see https://superuser.com/questions/1243344/vim-wont-split-open-a-directory-from-python-but-it-works-interactively)
	// so if it's a directory, use Hexplore instead.
	isDir := false
	var fi os.FileInfo
	if fi, err = os.Stat(fpath); err == nil && fi.IsDir() {
		isDir = true
	}

	action := "splitting"
	dirCmd := "Hexplore"
	splitCmd := "split"
	if method == openByTab {
		action = "tabbing"
		dirCmd = "Texplore"
		splitCmd = "tabedit"
	}

	trace(n, "trace: SplitOrChangeTo: no window matches. %s %s.", action, fpath)
	if isDir {
		// If it's a directory, use :Hexplore instead.
		err = nv.Command(fmt.Sprintf("%s %s", dirCmd, fpath))
	} else {
		err = nv.Command(fmt.Sprintf("%s %s", splitCmd, fpath))
	}

	return
}

func commandWhichExists(cmds []string) string {
	for _, cmd := range cmds {
		path, err := exec.LookPath(cmd)
		if err == nil {
			return path
		}
	}
	return ""
}

func (n Basejump) OpenRemoteUrl(url *url.URL, method string) error {
	nv := n.nvim()

	browsers := []string{"elinks", "w3m", "links", "lynx"}

	err := nv.Var("basejump_browsers", browsers)
	if err != nil {
		n.Echom("basejump_browsers is not defined (%v). Defaulting to %s", err, browsers)
	}

	b := commandWhichExists(browsers)
	if b == "" {
		return fmt.Errorf("error: no browser found. Tried %v", browsers)
	}

	action := "new"
	if method == openByTab {
		action = "tabnew"
	}
	err = nv.Command(action)
	if err != nil {
		return err
	}

	err = nv.Call("termopen", nil, fmt.Sprintf("%s %s", b, url))
	return err
}

func (n Basejump) OpenPath(text, method string) error {
	var path string
	var line, col int

	nv := n.nvim()

	trace(n, "trace: checking for URL")
	// First, check for a URL
	url, err := url.Parse(text)
	if err == nil {
		if url.Scheme == "file" {
			path = url.Path
		} else if url.Scheme == "http" || url.Scheme == "https" {
			// Handle a remote URL specially
			return n.OpenRemoteUrl(url, method)
		}
	}

	if path == "" {
		trace(n, "trace: parsing path")
		path, line, col, err = n.ParsePath(text)
		if err != nil {
			return err
		}
	}

	var openNonexistent int
	nv.Var("basejump_open_nonexistent", &openNonexistent)

	trace(n, "trace: checking if path exists")
	if openNonexistent == 0 && !pathExists(path) {
		return fmt.Errorf("error: no such file '%s'", path)
	}

	/*
		trace(n, "trace: ensuring file is open or opening it")
		var wasOpen bool
		wasOpen, err = n.OpenOrChangeTo(path, method)
		if err != nil {
			return err
		}

		if col == 0 {
			col = 1
		}
		if !wasOpen {
			if line == 0 {
				line = 1
			}
			err = n.JumpToLineAndCol(line, col)
		} else {
			if line != 0 {
				err = n.JumpToLineAndCol(line, col)
			}
		}

		if err != nil {
			return err
		}
	*/

	n.OpenPathAtLineCol(path, line, col, method)
	return nil
}

func (n Basejump) OpenPathAtLineCol(path string, line, col int, method string) (err error) {
	trace(n, "trace: ensuring file is open or opening it")
	var wasOpen bool
	wasOpen, err = n.OpenOrChangeTo(path, method)
	if err != nil {
		return
	}

	if col == 0 {
		col = 1
	}
	if !wasOpen {
		if line == 0 {
			line = 1
		}
		err = n.JumpToLineAndCol(line, col)
	} else {
		if line != 0 {
			err = n.JumpToLineAndCol(line, col)
		}
	}

	return
}

func (n Basejump) OpenSelectedPath(method string) error {
	trace(n, "trace: obtaining selected text")

	text, err := n.SelectionText()
	if err != nil {
		return err
	}

	if len(text) == 0 {
		err = fmt.Errorf("selection is empty")
		return err
	}

	// To expand tildes into home directories, we need a second expand
	nv := n.nvim()
	err = nv.Call("expand", &text, text)
	if err != nil {
		return err
	}

	return n.OpenPath(text, method)
}

func (n Basejump) OpenPathUnderCursor(method string) error {
	text, err := n.CurrentLineText()
	if err != nil {
		return err
	}
	_, col, err := n.Cursor()
	if err != nil {
		return err
	}

	nv := n.nvim()
	var pathChars string
	err = nv.Var("basejump_pathchars", &pathChars)
	if err != nil {
		pathChars = "-~/[a-z][A-Z].:[0-9]"
		n.Echom("basejump_pathchars is not defined. Defaulting to %s", pathChars)
	}

	text = matching(text, col-1, pathChars)

	// To expand tildes into home directories, we need a second expand
	err = nv.Call("expand", &text, text)
	if err != nil {
		return err
	}

	return n.OpenPath(text, method)
}

// JumpToLineAndCol moves the cursor to the specified line and column in the
// current buffer.
func (n Basejump) JumpToLineAndCol(line, col int) (err error) {
	nv := n.P.Nvim
	// In order to store these jumps in the jump history
	// we use the 'G' command first. This only stores the line
	// number, though, with column 1 (instead of the correct column).
	// We store a second jump by doing a forward search for any character,
	// with the count of the column number i.e. 10/.
	nv.Command(fmt.Sprintf("normal %dG", line))
	if col > 1 {
		nv.Command(fmt.Sprintf("normal %d/.", col-1))
	}

	// Just to make sure we didn't mess up
	err = nv.Call("cursor", nil, line, col)
	return
}

func (n Basejump) OpenLineFromDiff(method string) error {
	// When the diff code checks if a path exists, we want it to be
	// relative to the current window, not to the basejump process.
	pe := func(path string) bool {
		npath, err := n.AbsPath(path)
		if err != nil {
			trace(n, "trace: OpenLineFromDiff: AbsPath(%s) failed: %v", path, err)
			npath = path
		}
		trace(n, "trace: OpenLineFromDiff: npath is %s", npath)

		return pathExists(npath)
	}

	path, lineNo, err := diff.CalcFileAndLine(n, pe)

	trace(n, "trace: diff file: computed file '%s' line %d", path, lineNo)

	if err != nil {
		return err
	}

	return n.OpenPathAtLineCol(path, lineNo, 1, method)
}

func expandCharRanges(chars string) string {
	crunes := []rune(chars)

	output := make([]rune, 0, 26)

	const (
		inside = iota
		outside
	)

	state := outside
	start := -1
	var low, high rune

	var i int

	// false alarm. Was not a range
	restart := func() {
		i = start
		output = append(output, '[')
		state = outside
	}

	expand := func() {
		if high < low {
			low, high = high, low
		}

		for r := low; r <= high; r++ {
			output = append(output, r)
		}
	}

	for i = 0; i < len(crunes); i++ {

		r := crunes[i]
		//fmt.Println("i=", i, "r=", string(r))

		if state == inside {
			// handle each position within [X-Y]
			switch i - start {
			case 1:
				low = r
			case 2:
				if r != '-' {
					restart()
					continue
				}
			case 3:
				high = r
			case 4:
				if r != ']' {
					restart()
					continue
				}
				state = outside
				expand()
			}

			if i == len(crunes)-1 && state != outside {
				restart()
			}
		} else {

			switch r {
			case '[':
				if i < len(crunes)-1 {
					state = inside
					start = i
				} else {
					output = append(output, r)
				}
			default:
				output = append(output, r)
			}
		}
	}

	return string(output)
}

// Starting at `index` in string `s`, move forwards and backwards
// to find the longest string around `index` that contains only characters in
// `chars`.
func matching(s string, index int, chars string) string {
	if index < 0 || index >= len(s) {
		return ""
	}

	srunes := []rune(s)

	crunes := []rune(expandCharRanges(chars))
	good := func(i int) bool {
		for _, r := range crunes {
			if srunes[i] == r {
				return true
			}
		}
		return false
	}

	if !good(index) {
		return ""
	}

	left := index
	for ; left >= 0; left-- {
		if !good(left) {
			break
		}
	}

	right := index
	for ; right < len(s); right++ {
		if !good(right) {
			break
		}
	}

	return string(srunes[left+1 : right])
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

var optLogPanic = flag.Bool("logpanic", false, "log panics to the file /tmp/basejump.panic")
var optTrace = flag.Bool("trace", false, "trace execution to messages history")

func logPanic() {
	if v := recover(); v != nil {
		f, err := os.Create("/tmp/basejump.panic")
		if err != nil {
			return
		}
		fmt.Fprintf(f, "%s\n", v)
		fmt.Fprintf(f, "%s\n", debug.Stack())
		f.Close()
	}
}

func trace(n Basejump, fmt string, args ...interface{}) {
	if *optTrace {
		n.Echom(fmt, args...)
	}
}

const (
	openBySplit = "split"
	openByTab   = "tab"
)

func main() {

	flag.Parse()

	plugin.Main(func(p *plugin.Plugin) error {

		a := Basejump{p}

		openSelectedPath := func(args []string) (string, error) {
			if *optLogPanic {
				defer logPanic()
			}

			if len(args) == 0 {
				args = append(args, openBySplit)
			}

			err := a.OpenSelectedPath(args[0])
			if err != nil {
				a.Echom("error: %v", err)
			}
			// Returning an error here prints too much overdramatic red text
			return "", nil
		}

		openPathUnderCursor := func(args []string) (string, error) {
			if *optLogPanic {
				defer logPanic()
			}

			if len(args) == 0 {
				args = append(args, openBySplit)
			}

			err := a.OpenPathUnderCursor(args[0])
			if err != nil {
				a.Echom("error: %v", err)
			}
			// Returning an error here prints too much overdramatic red text
			return "", nil
		}

		openLineFromDiff := func(args []string) (string, error) {
			if *optLogPanic {
				defer logPanic()
			}

			if len(args) == 0 {
				args = append(args, openBySplit)
			}

			err := a.OpenLineFromDiff(args[0])

			if err != nil {
				a.Echom("error: %v", err)
			}
			// Returning an error here prints too much overdramatic red text
			return "", nil
		}

		p.HandleFunction(&plugin.FunctionOptions{Name: "OpenSelectedPath"}, openSelectedPath)
		p.HandleFunction(&plugin.FunctionOptions{Name: "OpenPathUnderCursor"}, openPathUnderCursor)
		p.HandleFunction(&plugin.FunctionOptions{Name: "OpenLineFromDiff"}, openLineFromDiff)
		return nil
	})
}

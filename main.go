package main

//
// ~/.local/share/nvim/site/plugin
// cp $GOPATH/src/github.com/jeffwilliams/nvacme/nvacme.vim $GOPATH/bin/nvacme .

import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/neovim/go-client/nvim"
	"github.com/neovim/go-client/nvim/plugin"
)

type NvAcme struct {
	P *plugin.Plugin
}

func (n NvAcme) nvim() *nvim.Nvim {
	return n.P.Nvim
}

// Echom formats it's arguments using fmt.Sprintf, then performs the echom command with the resulting
// string. Basically a printf to vim's status line and stores it in vim's messages.
func (n NvAcme) Echom(fmts string, args ...interface{}) {
	s := fmt.Sprintf(fmts, args...)
	s = strings.Replace(s, "'", "''", -1)
	n.P.Nvim.Command(fmt.Sprintf(":echom '%s'", s))
}

// Selection returns the coordinates of the current selection, if it is within a single line
func (n NvAcme) Selection() (line, startCol, endCol int, err error) {
	result := make([]float32, 4)
	nv := n.nvim()

	err = nv.Call("getpos", result, "'<")
	if err != nil {
		return
	}

	startLine := result[1]
	startCol = int(result[2])

	err = nv.Call("getpos", result, "'>")
	if err != nil {
		return
	}

	endLine := result[1]
	endCol = int(result[2])

	if startLine != endLine {
		err = fmt.Errorf("selection is multiple lines")
		return
	}

	line = int(startLine)

	return
}

// SelectionText returns the text contained in the current selection.
func (n NvAcme) SelectionText() (text string, err error) {
	nv := n.nvim()

	var line, startCol, endCol int
	line, startCol, endCol, err = n.Selection()
	if err != nil {
		return
	}

	var buf nvim.Buffer
	buf, err = nv.CurrentBuffer()
	if err != nil {
		return
	}

	var bytes [][]byte
	bytes, err = nv.BufferLines(buf, line-1, line, true)
	if err != nil {
		return
	}

	if len(bytes) == 0 {
		err = fmt.Errorf("selection is empty")
		return
	}

	if len(bytes) > 1 {
		err = fmt.Errorf("selection is multiple lines")
		return
	}

	// For some reason a visual selection can select one character past the end of the line
	if endCol > len(bytes[0]) {
		endCol = len(bytes[0])
	}

	text = string(bytes[0][startCol-1 : endCol])

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
func (n NvAcme) ParsePath(text string) (fpath string, line, col int, err error) {
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
func (n NvAcme) AbsPath(fpath string) (result string, err error) {
	return n.AbsPathRelWindow(fpath, -1)
}

func (n NvAcme) pidCwd(pid int) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
}

// AbsPathRelWindow makes the path `fpath` absolute if it is not by prepending
// the working directory of the window `window`.
func (n NvAcme) AbsPathRelWindow(fpath string, window int) (result string, err error) {
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
				//buf, err = nv.WindowBuffer(win)
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

// SplitOrChangeTo ensures the specified file is open in vim. If the path is found in a
// window, that window is made current. If no window contains that path, it is split and
// opened.
func (n NvAcme) SplitOrChangeTo(fpath string) (wasOpen bool, err error) {
	nv := n.nvim()

	wins, err := nv.Windows()
	if err != nil {
		return
	}

	for _, win := range wins {
		var buf nvim.Buffer
		var bufFileName string
		var winNr int

		buf, err = nv.WindowBuffer(win)
		if err != nil {
			return
		}
		bufFileName, err = nv.BufferName(buf)
		if err != nil {
			return
		}

		winNr, err = nv.WindowNumber(win)
		if err != nil {
			return
		}

		bufFileName, err = n.AbsPathRelWindow(bufFileName, winNr)
		if err != nil {
			return
		}
		if path.Clean(bufFileName) == path.Clean(fpath) {
			// Change to this window
			trace(n, "trace: SplitOrChangeTo: changing to existing window")
			err = nv.Command(fmt.Sprintf("%dwincmd w", winNr))
			wasOpen = true
			return
		}
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

	trace(n, "trace: SplitOrChangeTo: no window matches. splitting %s.", fpath)
	if isDir {
		// If it's a directory, use :Hexplore instead.
		err = nv.Command(fmt.Sprintf("Hexplore %s", fpath))
	} else {
		err = nv.Command(fmt.Sprintf("split %s", fpath))
	}

	return
}

// JumpToLineAndCol moves the cursor to the specified line and column in the
// current buffer.
func (n NvAcme) JumpToLineAndCol(line, col int) (err error) {
	nv := n.P.Nvim
	err = nv.Call("cursor", nil, line, col)
	return
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

var optLogPanic = flag.Bool("logpanic", false, "log panics to the file /tmp/nvacme.panic")

func logPanic() {
	if v := recover(); v != nil {
		f, err := os.Create("/tmp/nvacme.panic")
		if err != nil {
			return
		}
		fmt.Fprintf(f, "%s\n", v)
		fmt.Fprintf(f, "%s\n", debug.Stack())
		f.Close()
	}
}

var doTrace = false

func trace(n NvAcme, fmt string, args ...interface{}) {
	if doTrace {
		n.Echom(fmt, args...)
	}
}

func main() {

	flag.Parse()

	plugin.Main(func(p *plugin.Plugin) error {

		a := NvAcme{p}

		openPath := func(args []string) (string, error) {
			if *optLogPanic {
				defer logPanic()
			}

			trace(a, "trace: obtaining selected text")

			text, err := a.SelectionText()
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			trace(a, "trace: parsing path")
			path, line, col, err := a.ParsePath(text)
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			trace(a, "trace: checking if path exists")
			if !pathExists(path) {
				a.Echom("error: no such file '%s'", path)
				return "", nil
			}

			trace(a, "trace: ensuring file is open or opening it")
			var wasOpen bool
			wasOpen, err = a.SplitOrChangeTo(path)
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			if col == 0 {
				col = 1
			}
			if !wasOpen {
				if line == 0 {
					line = 1
				}
				err = a.JumpToLineAndCol(line, col)
			} else {
				if line != 0 {
					err = a.JumpToLineAndCol(line, col)
				}
			}

			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			return "", nil
		}

		p.HandleFunction(&plugin.FunctionOptions{Name: "OpenPath"}, openPath)
		return nil
	})
}

package main

// imap <M-MiddleMouse> asd
// :tmap <M-MiddleMouse> <C-\><C-n><C-w><Down>

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

func hello(args []string) (string, error) {
	return "Hello " + strings.Join(args, " "), nil
}

type NvAcme struct {
	P *plugin.Plugin
}

func (n NvAcme) Echom(fmts string, args ...interface{}) {
	s := fmt.Sprintf(fmts, args...)
	s = strings.Replace(s, "'", "''", -1)
	n.P.Nvim.Command(fmt.Sprintf(":echom '%s'", s))
}

// Selection returns the coordinates of the current selection, if it is within a single line
func (n NvAcme) Selection() (line, startCol, endCol int, err error) {
	result := make([]float32, 4)
	err = n.P.Nvim.Call("getpos", result, "'<")
	if err != nil {
		return
	}

	startLine := result[1]
	startCol = int(result[2])

	err = n.P.Nvim.Call("getpos", result, "'>")
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

func (n NvAcme) SelectionText() (text string, err error) {

	nv := n.P.Nvim

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

	text = string(bytes[0][startCol-1 : endCol])

	return
}

var pathRegex = regexp.MustCompile(`^([^:]+)(?::(\d+))?(?::(\d+))?`)

// KEEP CODING HERE: now that we have the text, parse it as a filename and open it. Use getcwd for relative paths.
// - convert path to absolute
// - parse the line number:column suffix
func (n NvAcme) parsePath(text string) (fpath string, line, col int, err error) {
	line = 1
	col = 1

	text = strings.TrimSpace(text)
	match := pathRegex.FindStringSubmatch(text)
	if match == nil || len(match) < 2 {
		err = fmt.Errorf("doesn't seem to be a valid path")
		return
	}
	n.Echom("match: %#v", match)
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

	if !path.IsAbs(fpath) {
		result := ""
		err = n.P.Nvim.Call("getcwd", &result)
		if err != nil {
			return
		}

		fpath = result + "/" + fpath
	}

	return
}

// splitOrChangeTo opens the path, unless it's already open in another wundow
func (n NvAcme) splitOrChangeTo(fpath string) (err error) {
	nv := n.P.Nvim

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

		if !path.IsAbs(bufFileName) {
			result := ""
			err = nv.Call("getcwd", &result, winNr)
			if err != nil {
				return
			}
			bufFileName = result + "/" + bufFileName
		}

		if bufFileName == fpath {
			// Change to this window
			err = nv.Command(fmt.Sprintf("%dwincmd w", winNr))
			return
		}
	}

	// Not found. Split new window
	err = nv.Command(fmt.Sprintf("split %s", fpath))

	return
}

func (n NvAcme) jumpToLineAndCol(line, col int) (err error) {
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

func main() {

	flag.Parse()

	plugin.Main(func(p *plugin.Plugin) error {

		a := NvAcme{p}

		openPath := func(args []string) (string, error) {
			if *optLogPanic {
				defer logPanic()
			}
			line, startCol, endCol, err := a.Selection()

			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			a.Echom("line: %d, scol: %d, ecol: %d", line, startCol, endCol)

			text, err := a.SelectionText()
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			a.Echom("text: %s", text)

			path, line, col, err := a.parsePath(text)
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}
			a.Echom("path: %s line: %d col: %d", path, line, col)

			if !pathExists(path) {
				a.Echom("error: no such file '%s'", path)
				return "", nil
			}

			err = a.splitOrChangeTo(path)
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			err = a.jumpToLineAndCol(line, col)
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}

			return "", nil
		}

		p.HandleFunction(&plugin.FunctionOptions{Name: "Hello"}, hello)
		p.HandleFunction(&plugin.FunctionOptions{Name: "OpenPath"}, openPath)
		return nil
	})
}

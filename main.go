package main

// imap <M-MiddleMouse> asd
// :tmap <M-MiddleMouse> <C-\><C-n><C-w><Down>

//
// ~/.local/share/nvim/site/plugin
// cp $GOPATH/src/github.com/jeffwilliams/nvacme/nvacme.vim $GOPATH/bin/nvacme .

import (
	"fmt"
	"regexp"
	//"strconv"
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

	n.Echom("bytes len: %d", len(bytes))
	for i, v := range bytes {
		n.Echom("bytes[%d]: %s", i, v)
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

var pathRegex = regexp.MustCompile(`^([^:]+)(:\d+)(:\d+)`)

// KEEP CODING HERE: now that we have the text, parse it as a filename and open it. Use getcwd for relative paths.
// - convert path to absolute
// - parse the line number:column suffix
func (n NvAcme) parsePath(text string) (path string, line, col int, err error) {
	line = 1
	col = 1

	text = strings.TrimSpace(text)
	match := pathRegex.FindStringSubmatch(text)
	if match == nil || len(match) < 2 {
		err = fmt.Errorf("doesn't seem to be a valid path")
		return
	}
	path = match[1]
	/*
		if len(match) > 2 {
			line, err = strconv.Atoi(match[2])
			if err != nil {
				return
			}
		}
		if len(match) > 3 {
			col, err = strconv.Atoi(match[3])
			if err != nil {
				return
			}
		}
	*/
	return
}

func main() {
	plugin.Main(func(p *plugin.Plugin) error {

		a := NvAcme{p}

		openPath := func(args []string) (string, error) {
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

			a.Echom("texta: %s", text)

			path, line, col, err := a.parsePath(text)
			if err != nil {
				a.Echom("error: %v", err)
				return "", nil
			}
			a.Echom("path: %s line: %d col: %d", path, line, col)
			return "", nil
		}

		p.HandleFunction(&plugin.FunctionOptions{Name: "Hello"}, hello)
		p.HandleFunction(&plugin.FunctionOptions{Name: "OpenPath"}, openPath)
		return nil
	})
}

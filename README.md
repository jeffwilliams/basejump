# Basejump

Basejump is an nvim plugin for jumping to a file path -- optionally with a line and column number -- contained in text.

For example, if an nvim buffer contains the text:

    /home/user/src/file.c

if you move the cursor to somewhere in that text and press ALT-RightMouse, it will split the buffer and open that file in the new buffer. If that file is already open in a buffer basejump doesn't split and instead focuses that buffer.

If the file was suffixed with a line number like:

    /home/user/src/file.c:40

the cursor is moved to line 40. Finally a column number like so:

    /home/user/src/file.c:40:5

will move to line 40 and column 5. 

Basejump also supports opening file:// and http:// URLs. For http:// URLs basejump attempts to start an installed text-mode browser in a new terminal window or tab.

Finally, when the cursor is positioned inside a unified diff, pressing ALT-Shift-RightMouse will split the buffer and jump to the line in the modified file that the cursor is positioned over.

# Default Keybindings

Mode        | Binding        | Description
------------|----------------|------------
Visual Mode | ALT-RightMouse | Treat the selected text as a path or URL and jump to it. If the path is not found in the open windows, it is opened as a new window.
Normal Mode | ALT-RightMouse | Find the longest valid path or URL under the cursor, and jump to it. If the path is not found in the open windows, it is opened as a new window.
Normal Mode | ALT-SHIFT-RightMouse | Assume the cursor is inside a unified diff and jump to that line in the diff's modified file.
Normal Mode | ALT-]          | Find the longest valid path or URL under the cursor, and jump to it. 
Normal Mode | ALT-SHIFT-]    | Jump to the diff line under the cursor in the modified file

# Functions

The bindings described above are implemented by calling vim functions. The functions can be called directly if desired. They are:

    BasejumpOpenSelectedPathRange(mode)
    OpenPathUnderCursor(mode)
    OpenLineFromDiff(mode)

Each takes one parameter describing the mode by which files are opened. It may be either 'tab' or 'split'.

# Configuring

By default basejump opens the files it jumps to by splitting the current buffer. This can be changed to instead open the file
in a new tab by setting the variable `g:basejump_openmode`. The allowed values are:

    " Open the file by splitting the buffer
    let g:basejump_openmode = 'split'

    " Open the file by creating a new tab
    let g:basejump_openmode = 'tab'

When finding the longest path under the cursor, the characters that basejump assumes are part of the path are controlled using the 
variable:

    let g:basejump_pathchars = '-~/[a-z][A-Z].:[0-9]_'

Add or remove characters to change the allowed set.

You can change the keybindings by unmapping them and then mapping the desired mapping in your .vimrc. For example, to bind 
ALT-SHIFT-MiddleMouse to open a line from a diff do:

    nmap <M-MiddleMouse> :call OpenLineFromDiff(g:basejump_openmode)<CR>

# Comparison to gf

Basejump behaves very similar to the gf, gF, CTRL-W F, etc. family of commands. The main differences are:

  * The normal mode form of the commands don't support a column suffix after the line
  * The visual mode form of the commands doesn't support the line number suffix of the path, only the file path itself
  * The "new window" form of the commands always opens a new window, even if one already exists for the file
  * Basejump supports http:// URLs
  * Basejump supports diffs

# Installation

Install using [https://github.com/junegunn/vim-plug](vim-plug). Add the following to your plug section:

    Plug 'jeffwilliams/basejump', { 'do': './install' }




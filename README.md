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

# Default Keybindings

Mode        | Binding        | Description
------------|----------------|------------
Visual Mode | ALT-RightMouse | Treat the selected text as a path or URL and jump to it. If the path is not found in the open windows, it is opened as a new window.
Visual Mode | ALT-SHIFT-RightMouse | Same as above, but if not already open, it is opened as a new tabpage.
Normal Mode | ALT-RightMouse | Find the longest valid path or URL under the cursor, and jump to it. If the path is not found in the open windows, it is opened as a new window.
Normal Mode | ALT-SHIFT-RightMouse | Same as above, but if not already open, it is opened as a new tabpage.
Normal Mode | ALT-]          | Find the longest valid path or URL under the cursor, and jump to it. 
Normal Mode | ALT-}          | Same as above, but if not already open, it is opened as a new tabpage.

# Comparison to gf

Basejump behaves very similar to the gf, gF, CTRL-W F, etc. family of commands. The main differences are:

  * The normal mode form of the commands don't support a column suffix after the line
  * The visual mode form of the commands doesn't support the line number suffix of the path, only the file path itself
  * The "new window" form of the commands always opens a new window, even if one already exists for the file
  * Basejump supports http:// URLs

# Installation

Install using [https://github.com/junegunn/vim-plug](vim-plug). Add the following to your plug section:

    Plug 'jeffwilliams/basejump', { 'do': './install' }




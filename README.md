# Basejump

Basejump is an nvim plugin for jumping to a file path -- optionally with a line and column number -- contained in text.

For example, if an nvim buffer contains the text:

    /home/user/src/file.c

if you move the cursor to somewhere in that text and press ALT-RightMouse, it will split the buffer and open that file in the new buffer. If that file is already open in a buffer basejump doesn't split and instead focuses that buffer.

If the file was suffixed with a line number like:

    /home/user/src/file.c:40

the cursor is moved to line 40. Finally a column number like so:

    /home/user/src/file.c:40:5

will move to line 40 and column 5. For convenience in the documentation we'll refer to a path with an optional line and column as an _offset-path_. The act of opening an offset-path in a new buffer, or if the path is already open moving the cursor to that buffer and positioning the cursor in it, will be referred to as _jumping_.

# Supported Systems

Currently only Linux amd64 is supported. Support for other architectures is in progress.

# Default Keybindings

Mode        | Binding        | Description
------------|----------------|------------
Visual Mode | ALT-RightMouse | Treat the selected text as an offset-path and jump to it. If the path is not found in the open windows, it is opened as a new window.
Visual Mode | ALT-SHIFT-RightMouse | Same as above, but if not already open, it is opened as a new tabpage.
Normal Mode | ALT-RightMouse | Find the longest valid offset-path under the cursor, and jump to it. If the path is not found in the open windows, it is opened as a new window.
Normal Mode | ALT-SHIFT-RightMouse | Same as above, but if not already open, it is opened as a new tabpage.
Normal Mode | ALT-]          | Find the longest valid offset-path under the cursor, and jump to it. 
Normal Mode | ALT-}          | Same as above, but if not already open, it is opened as a new tabpage.

# Comparison to gf

Basejump behaves very similar to the gf, gF, CTRL-W F, etc. family of commands. The main differences are:

  * The normal mode form of the commands don't support a column suffix after the line
  * The visual mode form of the commands doesn't support the line number suffix of the path, only the file path itself
  * The "new window" form of the commands always opens a new window, even if one already exists for the file

# Installation

Install using [https://github.com/junegunn/vim-plug](vim-plug). Add the following to your plug section:

    Plug 'jeffwilliams/basejump'





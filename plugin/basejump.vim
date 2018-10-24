if exists('g:loaded_basejump')
  finish
endif

let g:loaded_basejump = 1
" basejump_pathchars are the characters that are considered path of a valid
" path. These are used by OpenPathUnderCursor to determine the extent of the
" path under the cursor.
let g:basejump_pathchars = '-~/[a-z][A-Z].:[0-9]_'

let s:basejump_path = expand('<sfile>:p:h') . '/basejump' 

function! s:RequireBasejump(host) abort
  " 'basejump' is the binary created by compiling the program.
  " If '-logpanic' is specified, panics in the binary are logged to
  " /tmp/basejump.panic
  return jobstart([s:basejump_path], {'rpc': v:true})
  "return jobstart([s:basejump_path,'-logpanic'], {'rpc': v:true})
endfunction

call remote#host#Register('basejump', 'x', function('s:RequireBasejump'))

vmap <M-RightMouse> :call OpenSelectedPath()<CR>
nmap <M-RightMouse> :call OpenPathUnderCursor()<CR>
nmap <M-]> :call OpenPathUnderCursor()<CR>

" The following lines are generated by running the program
" command line flag --manifest basejump
call remote#host#RegisterPlugin('basejump', '0', [
\ {'type': 'function', 'name': 'OpenPathUnderCursor', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'OpenSelectedPath', 'sync': 1, 'opts': {}},
\ ])


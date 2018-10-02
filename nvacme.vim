if exists('g:loaded_nvacme')
  finish
endif
let g:loaded_nvacme = 1

function! s:RequireNvacme(host) abort
  " 'nvacme' is the binary created by compiling the program.
  return jobstart(['nvacme','-logpanic'], {'rpc': v:true})
  "return jobstart(['nvacme'], {'rpc': v:true})
endfunction

call remote#host#Register('nvacme', 'x', function('s:RequireNvacme'))


" For debugging
vmap <M-RightMouse> :call OpenPath()<CR>

" The following lines are generated by running the program
" command line flag --manifest nvacme
call remote#host#RegisterPlugin('nvacme', '0', [
\ {'type': 'function', 'name': 'OpenPath', 'sync': 1, 'opts': {}},
\ ])

-- Editable review thread. Source edits are ordinary human buffer edits; no Git,
-- disk save or agent submission. The pure session owns save/close decisions.
local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local thread = dofile(here .. 'comment_thread.lua')
local markers = dofile(here .. 'markers.lua')
local NS = vim.api.nvim_create_namespace('pair_comment_thread')
local active = {} -- source buffer -> its one owned scratch/window
local function notify(msg) vim.notify('review thread: ' .. msg, vim.log.levels.WARN) end
local function text(buf) return table.concat(vim.api.nvim_buf_get_lines(buf,0,-1,false),'\n') end

local function anchor(st,row,col,raw)
  if st.mark then pcall(vim.api.nvim_buf_del_extmark,st.src,NS,st.mark) end
  st.mark = vim.api.nvim_buf_set_extmark(st.src,NS,row,col,{
    end_row=row,end_col=col+#raw,right_gravity=true,end_right_gravity=false,
    invalidate=true,undo_restore=false,
  })
end
local function observed(st)
  if not vim.api.nvim_buf_is_valid(st.src) then return nil end
  local p = vim.api.nvim_buf_get_extmark_by_id(st.src,NS,st.mark,{details=true})
  if #p == 0 or p[3].invalid or p[1] ~= p[3].end_row then return nil end
  local raw = table.concat(vim.api.nvim_buf_get_text(st.src,p[1],p[2],p[3].end_row,p[3].end_col,{}),'\n')
  return raw,p
end
local function paint(st)
  if not vim.api.nvim_buf_is_valid(st.buf) then return end
  vim.api.nvim_buf_clear_namespace(st.buf,NS,0,-1)
  for i,role in ipairs(thread.roles(vim.api.nvim_buf_get_lines(st.buf,0,-1,false))) do
    vim.api.nvim_buf_set_extmark(st.buf,NS,i-1,0,{
      line_hl_group=role=='agent' and 'ParleyCommentAgent' or 'ParleyCommentUser',
    })
  end
end
local function owns_window(st)
  return st.win and vim.api.nvim_win_is_valid(st.win)
    and vim.api.nvim_win_get_buf(st.win)==st.buf
end
local function cleanup(st,discard,departing)
  if active[st.src] ~= st then return end
  local dirty = vim.api.nvim_buf_is_valid(st.buf) and vim.bo[st.buf].modified
  local effect = st.session:transition({kind='close',dirty=dirty,discard=discard})
  active[st.src] = nil
  if effect.rescue then
    vim.fn.setreg('"',text(st.buf))
    notify('thread closed without saving; text kept in register "')
  end
  if st.group then pcall(vim.api.nvim_del_augroup_by_id,st.group) end
  if vim.api.nvim_buf_is_valid(st.src) then pcall(vim.api.nvim_buf_del_extmark,st.src,NS,st.mark) end
  local function release()
    if not departing and owns_window(st) then
      local ok=pcall(vim.api.nvim_win_close,st.win,true)
      if not ok then return false end
    end
    if vim.api.nvim_buf_is_valid(st.buf) then return pcall(vim.api.nvim_buf_delete,st.buf,{force=true}) end
    return true
  end
  -- Buffer teardown may hold Neovim's text/window lock. Ownership is already
  -- ended; finish releasing that exact resource set on the next event turn.
  if departing then
    -- BufWinLeave runs before the replacement becomes current. Preserve that
    -- user-owned window, and release only our scratch after the switch ends.
    vim.schedule(release)
  elseif not release() then vim.schedule(release) end
end
function M.close(src) local st=active[src]; if st then cleanup(st,false) end end

local function write_back(st)
  local raw,p = observed(st)
  local proposed,err = thread.from_lines(vim.api.nvim_buf_get_lines(st.buf,0,-1,false),st.meta)
  local effect=st.session:transition({kind='save',observed=raw,proposed=proposed,error=err})
  if effect.kind ~= 'replace' then error('review thread: '..(effect.message or 'save refused'),0) end
  if not vim.bo[st.src].modifiable then
    st.session:transition({kind='failed'})
    error('review thread: source is not modifiable',0)
  end
  if proposed ~= raw then
    local ok,why=pcall(vim.api.nvim_buf_set_text,st.src,p[1],p[2],p[3].end_row,p[3].end_col,{proposed})
    if not ok then
      st.session:transition({kind='failed'})
      error('review thread: '..tostring(why),0)
    end
    anchor(st,p[1],p[2],proposed)
  end
  st.session:transition({kind='saved',raw=proposed})
  local _,meta=thread.to_lines(markers.parse_markers({proposed})[1])
  st.meta=meta
  vim.bo[st.buf].modified=false
  if st.changed then st.changed(st.src) end
end

-- m is the shared compact projection record, never an independent line lookup.
function M.open_thread(src,m,changed)
  if not m or m.broken or not m.marker then return false end
  local old=active[src]
  if old then
    if owns_window(old) then vim.api.nvim_set_current_win(old.win); return true end
    cleanup(old,false)
  end
  local marker=m.marker
  if marker.complete==false or marker.raw:find('\n',1,true) then return false end
  local lines,meta=thread.to_lines(marker)
  if not lines then notify(meta); return false end
  local buf=vim.api.nvim_create_buf(false,true)
  local st={src=src,buf=buf,meta=meta,session=thread.new_session(marker.raw),changed=changed}
  local ok,err=pcall(function()
    vim.api.nvim_buf_set_name(buf,('pair-comment://%d/%d'):format(src,buf))
    vim.api.nvim_buf_set_lines(buf,0,-1,false,lines)
    vim.bo[buf].buftype='acwrite'; vim.bo[buf].bufhidden='wipe'; vim.bo[buf].swapfile=false
    vim.bo[buf].syntax='markdown'; vim.bo[buf].modified=false
    local maxw,maxh=math.max(1,vim.o.columns-2),math.max(1,vim.o.lines-2)
    local width=math.min(maxw,math.max(1,math.floor(vim.o.columns*.8)))
    local height=0
    for _,line in ipairs(lines) do height=height+math.max(1,math.ceil(vim.fn.strdisplaywidth(line)/width)) end
    height=math.min(maxh,math.max(3,math.min(height,math.floor(vim.o.lines*.8))))
    st.win=vim.api.nvim_open_win(buf,true,{relative='editor',style='minimal',border='rounded',
      width=width,height=height,row=math.max(0,math.floor((vim.o.lines-height-2)/2)),
      col=math.max(0,math.floor((vim.o.columns-width-2)/2)),
      title=width>=16 and ' 🤖 thread ' or nil,
      footer=width>=45 and ' :w save · q save/close · :q! discard ' or nil})
    vim.wo[st.win].wrap=true; vim.wo[st.win].linebreak=true
    anchor(st,marker.line,marker.col,marker.raw)
  end)
  if not ok then
    if st.win and vim.api.nvim_win_is_valid(st.win) then pcall(vim.api.nvim_win_close,st.win,true) end
    if vim.api.nvim_buf_is_valid(buf) then vim.api.nvim_buf_delete(buf,{force=true}) end
    notify(tostring(err)); return false
  end
  active[src]=st
  st.group=vim.api.nvim_create_augroup('PairCommentThread'..buf,{clear=true})
  vim.api.nvim_create_autocmd({'TextChanged','TextChangedI'},{group=st.group,buffer=buf,callback=function() paint(st) end})
  vim.api.nvim_create_autocmd('BufWriteCmd',{group=st.group,buffer=buf,callback=function() write_back(st) end})
  -- A successful explicit quit of a dirty acwrite buffer is :q!. A refused :q
  -- must not mark a later forced close as a discard; clear at next event turn.
  vim.api.nvim_create_autocmd('QuitPre',{group=st.group,buffer=buf,callback=function()
    st.quitting=true
    vim.schedule(function() st.quitting=false end)
  end})
  vim.api.nvim_create_autocmd('WinClosed',{group=st.group,pattern=tostring(st.win),callback=function()
    cleanup(st,st.quitting==true)
  end})
  vim.api.nvim_create_autocmd('BufWipeout',{group=st.group,buffer=src,callback=function() cleanup(st,false) end})
  vim.api.nvim_create_autocmd({'BufWinLeave','BufWipeout'},{group=st.group,buffer=buf,callback=function()
    cleanup(st,st.quitting==true,true)
  end})
  vim.keymap.set('n','q','<Cmd>x<CR>',{buffer=buf,silent=true,desc='review: save and close thread'})
  paint(st)
  vim.api.nvim_win_set_cursor(st.win,{#lines,#lines[#lines]})
  vim.cmd('startinsert!')
  return true
end
return M

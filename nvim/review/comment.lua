-- Compact review view: activation-scoped cache, decorations and window options.
local M={}
local here=debug.getinfo(1,'S').source:match('@?(.*/)') or './'
local view=dofile(here..'comment_view.lua')
local markers=dofile(here..'markers.lua')
local floats=dofile(here..'comment_float.lua')
local NS=vim.api.nvim_create_namespace('review_markers')
local active={}
local MAX_LINES,MAX_BYTES=1000,128*1024

local function restore(win,w,all)
  if not vim.api.nvim_win_is_valid(win) then return end
  if w.forced then vim.wo[win].concealcursor=w.cc;w.forced=false end
  if all then vim.wo[win].conceallevel=w.level end
end
local function cursor(buf, entering)
  local s=active[buf]
  local win=vim.api.nvim_get_current_win()
  if not s or vim.api.nvim_win_get_buf(win)~=buf then return end
  local w=s.windows[win]
  if not w then
    w={level=vim.wo[win].conceallevel,cc=vim.wo[win].concealcursor}
    s.windows[win]=w
    vim.wo[win].conceallevel=2
  end
  local pos=vim.api.nvim_win_get_cursor(win)
  local row,col=pos[1]-1,pos[2]
  local rows=s.rows[row] or {}
  local rendered=false
  for _,m in ipairs(rows) do if not m.broken then rendered=true;break end end
  if rendered and not w.forced then
    w.cc=vim.wo[win].concealcursor;w.forced=true;vim.wo[win].concealcursor='nvic'
  elseif not rendered then restore(win,w,false) end
  if rendered and vim.wo[win].conceallevel>0 then
    local line=s.lines[row+1] or ''
    local mode=vim.api.nvim_get_mode().mode:sub(1,1)
    local insert=entering or mode=='i' or mode=='R'
    local to=view.snap(rows,w.col or 0,col,insert and #line or math.max(0,#line-1),line,insert)
    if to then vim.api.nvim_win_set_cursor(win,{row+1,to});col=to end
  end
  w.col=col
end
local function mark(buf,row,col,opts)
  -- All coordinates come from the parser of this exact buffer snapshot.
  vim.api.nvim_buf_set_extmark(buf,NS,row,col,opts)
end
function M.render(buf)
  if not vim.api.nvim_buf_is_valid(buf) then return end
  local s=active[buf]
  if not s then return end
  local tick=vim.api.nvim_buf_get_changedtick(buf)
  if tick==s.tick then cursor(buf);return end
  local lines=vim.api.nvim_buf_get_lines(buf,0,-1,false)
  local bytes=0;for _,l in ipairs(lines) do bytes=bytes+#l+1 end
  local within=#lines<=MAX_LINES and bytes<=MAX_BYTES
  s.rows=within and view.layout(lines) or {}
  s.lines=lines;s.tick=tick
  vim.api.nvim_buf_clear_namespace(buf,NS,0,-1)
  for _,sp in ipairs(markers.spans_multiline(lines)) do
    mark(buf,sp.row,sp.col,{end_row=sp.end_row,end_col=sp.end_col,hl_group=sp.hl_group})
  end
  for row,list in pairs(s.rows) do
    for _,m in ipairs(list) do
      if m.broken then
        mark(buf,row,m.start,{end_col=m.stop,hl_group='ParleyReviewBroken',priority=250})
      else
        if m.visible then
          mark(buf,row,m.visible[1],{end_col=m.visible[2],hl_group=m.visible[3],priority=200})
        end
        for _,r in ipairs(m.turns_hl) do mark(buf,row,r[1],{end_col=r[2],hl_group=r[3],priority=200}) end
        local function role_at(col)
          for _,r in ipairs(m.turns_hl) do if col>=r[1] and col<r[2] then return r[3] end end
        end
        for _,r in ipairs(m.hidden) do mark(buf,row,r[1],{end_col=r[2],conceal=r[3],hl_group=role_at(r[1]),priority=210}) end
        for _,r in ipairs(m.brackets) do mark(buf,row,r[1],{end_col=r[1]+1,conceal=r[2],hl_group=role_at(r[1]),priority=220}) end
      end
    end
  end
  if not within and not s.noticed then
    s.noticed=true
    vim.notify('review: compact comments disabled above 1,000 lines or 128 KiB; showing raw markers',vim.log.levels.INFO)
  end
  cursor(buf)
end
function M.layout(buf) M.render(buf);return active[buf] and active[buf].rows or {} end
function M.detach(buf)
  local s=active[buf];if not s then return end
  floats.close(buf)
  for win,w in pairs(s.windows) do restore(win,w,true) end
  active[buf]=nil
  vim.on_key(nil,s.key_listener)
  pcall(vim.api.nvim_del_augroup_by_id,s.group)
  if vim.api.nvim_buf_is_valid(buf) then
    pcall(vim.api.nvim_buf_del_keymap,buf,'n','<CR>')
    if s.enter and s.enter.buffer==1 then
      vim.api.nvim_buf_call(buf,function() vim.fn.mapset('n',false,s.enter) end)
    end
    vim.api.nvim_buf_clear_namespace(buf,NS,0,-1)
  end
end
function M.attach(buf)
  if active[buf] then M.render(buf);return end
  local s={rows={},lines={},windows={}}
  active[buf]=s
  vim.api.nvim_buf_call(buf,function() s.enter=vim.fn.maparg('<CR>','n',false,true) end)
  s.group=vim.api.nvim_create_augroup('PairReviewComment'..buf,{clear=true})
  local function watch(events,fn)
    vim.api.nvim_create_autocmd(events,{group=s.group,buffer=buf,callback=fn})
  end
  watch({'TextChanged','TextChangedI','InsertLeave','FileChangedShellPost'},function() M.render(buf) end)
  watch({'CursorMoved','CursorMovedI','BufEnter','WinEnter'},function() M.render(buf);cursor(buf) end)
  -- Admission event classes: entry (i/a/I/A/gi, R/gR and :startinsert),
  -- every input key (including queued motion + newline/delete/register paste),
  -- and post-edit refresh above. InsertCharPre misses non-character mutations.
  -- on_key runs after mappings but before processing each resulting key. This
  -- admits insertion points; native range edits from visible points stay native.
  -- InsertEnter restores the cursor unless v:char is nonempty.
  watch('InsertEnter',function()
    local before=vim.api.nvim_win_get_cursor(0)
    M.render(buf)
    cursor(buf,true)
    local after=vim.api.nvim_win_get_cursor(0)
    if before[1]~=after[1] or before[2]~=after[2] then vim.v.char=' ' end
  end)
  s.key_listener=vim.on_key(function()
    if vim.api.nvim_get_current_buf()~=buf then return end
    local mode=vim.api.nvim_get_mode().mode:sub(1,1)
    if mode=='i' or mode=='R' then M.render(buf);cursor(buf,true) end
  end)
  watch('BufLeave',function()
    local win=vim.api.nvim_get_current_win();if s.windows[win] then restore(win,s.windows[win],false) end
  end)
  watch('BufWinLeave',function()
    local win=vim.api.nvim_get_current_win()
    if s.windows[win] then restore(win,s.windows[win],true);s.windows[win]=nil end
  end)
  watch('BufWipeout',function() M.detach(buf) end)
  vim.api.nvim_create_autocmd('WinClosed',{group=s.group,callback=function(ev) s.windows[tonumber(ev.match)]=nil end})
  vim.keymap.set('n','<CR>',function()
    M.render(buf)
    local pos=vim.api.nvim_win_get_cursor(0)
    local m=view.marker_at(s.rows[pos[1]-1] or {},pos[2])
    if not floats.open_thread(buf,m,M.render) then
      local keys=(vim.v.count>0 and tostring(vim.v.count) or '')..'<CR>'
      vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes(keys,true,false,true),'n',false)
    end
  end,{buffer=buf,silent=true,desc='review: open comment thread'})
  M.render(buf)
end
return M

-- Pure compact-marker geometry; run with nvim -l.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local ok, view = pcall(dofile, here .. 'comment_view.lua')
assert(ok and type(view.layout) == 'function', 'compact marker projection is available')
local function eq(got, want, label)
  assert(vim.deep_equal(got, want), label .. ': got ' .. vim.inspect(got) .. ', wanted ' .. vim.inspect(want))
end
local function row(line) return view.layout({ line })[0] or {} end
local function shown(line)
  local cuts = {}
  for _, m in ipairs(row(line)) do for _, h in ipairs(m.hidden or {}) do cuts[#cuts + 1] = h end end
  table.sort(cuts, function(a,b) return a[1] < b[1] end)
  local out, pos = {}, 0
  for _, h in ipairs(cuts) do out[#out+1] = line:sub(pos+1,h[1]) .. h[3]; pos=h[2] end
  return table.concat(out) .. line:sub(pos+1)
end
for _, case in ipairs({
  {'see 🤖[why]{because}[ok] here','see 🤖[…]{…}[ok] here'},
  {'🤖[why]{because}','🤖[…]{…}'}, {'🤖{proposal}','🤖{…}'},
  {'🤖[typing]','🤖[typing]'}, {'🤖[]','🤖[]'},
  {'a 🤖<quick fox>[why]{yes}[more] b','a quick fox[…]{…}[more] b'},
  {'a 🤖~old~{new} b','a old{…} b'}, {'🤖<X>[]','X[]'},
  {'🤖<>[c]','🤖<>[c]'}, {'🤖<X>','X'},
  {'🤖[a] then 🤖<B>{c}','🤖[a] then B{…}'},
  {'use `🤖[literal]`','use `🤖[literal]`'},
}) do eq(shown(case[1]),case[2],'render '..case[1]) end
local lines = {'first', 'prefix 🤖<é>[q]{answer}[reply] tail'}
local m = view.layout(lines)[1][1]
eq(m.start,7,'second-row start'); eq(m.marker.raw,'🤖<é>[q]{answer}[reply]','source record retained')
eq(lines[2]:sub(m.visible[1]+1,m.visible[2]),'é','anchor geometry')
eq(lines[2]:sub(m.reply[1]+1,m.reply[2]),'reply','reply geometry')
eq(#m.brackets,6,'each bracket reasserted'); eq(#m.turns_hl,3,'speaker highlights')
eq(view.marker_at({m},m.start),m,'marker containment'); eq(view.marker_at({m},m.stop),nil,'exclusive stop')
for _, input in ipairs({'🤖[unclosed','🤖<X>[a]{unclosed','🤖<unclosed'}) do
  local broken=row(input)[1]; assert(broken and broken.broken,'broken visible '..input)
  eq(broken.hidden,nil,'broken not concealed'); eq(view.marker_at({broken},broken.start),nil,'broken not targetable')
end
eq(shown('🤖[unclosed 🤖{nested}'), '🤖[unclosed 🤖{nested}', 'malformed span stays entirely raw')
eq(row('ordinary 🤖 and 🤖: hello 🤖~/path'),{},'nonmarkers ignored')
local fenced=view.layout({'```lua','🤖[literal]','```','real 🤖{proposal}'})
eq(fenced[1],nil,'fenced marker excluded'); assert(fenced[3] and #fenced[3]==1,'outside fence eligible')
for _, char in ipairs({ '`', '~' }) do
  local fence = string.rep(char, 4)
  local projection = view.layout({ '   ' .. fence .. 'text', '🤖{hidden}',
    string.rep(char, 3), '🤖[still hidden]', fence .. ' invalid close',
    '🤖{broken', '  ' .. fence .. char, '🤖{visible}' })
  for row_idx = 0, 6 do eq(projection[row_idx], nil, 'Markdown fence remains literal') end
  assert(projection[7] and #projection[7] == 1, 'after matching fence remains compact')
end
eq(view.layout({'🤖<first','🤖[nested] second>[reply]'}),{},'multiline and nested marker stay literal')
eq(view.layout({'🤖[first','second]'}),{},'multiline turn stays literal')
-- Normal cursor cannot land on any concealed byte, including its first byte.
local l='ab 🤖<é>[q]{answer} z'; local ms=row(l); local a=ms[1]
eq(view.snap(ms,a.start-1,a.start,#l-1,l),a.visible[1],'right to anchor')
eq(view.snap(ms,a.visible[1],a.visible[1]-1,#l-1,l),a.start-1,'left before marker')
local closing=a.hidden[2]
eq(view.snap(ms,closing[2],closing[1],#l-1,l),a.visible[1],'UTF8 left landing')
for _, h in ipairs(a.hidden) do
  local to=view.snap(ms,math.max(0,h[1]-1),h[1],#l-1,l)
  assert(to and (to<h[1] or to>=h[2]),'first hidden byte skipped')
end
local ins='🤖[a]{bbb}[last] z'; local ims=row(ins); local im=ims[1]
local inside=im.hidden[2][1]+1
eq(view.snap(ims,inside-1,inside,#ins,ins,true),im.reply[1],'insert right into reply')
eq(view.snap(ims,inside+1,inside,#ins,ins,true),im.start,'insert left before marker')
for c=im.reply[1],im.reply[2] do eq(view.snap(ims,0,c,#ins,ins,true),nil,'reply point legal') end
for _, c in ipairs({im.start,im.stop}) do eq(view.snap(ims,0,c,#ins,ins,true),nil,'outside insertion legal') end
local fresh=row('x 🤖[] y')[1]
eq(view.snap({fresh},0,fresh.reply[1],12,'x 🤖[] y',true),nil,'empty reply insertion legal')
-- Seeded malformed/Unicode inputs ensure projection boundaries remain legal.
math.randomseed(426)
local atoms={'🤖','[',']','{','}','<','>','~','a',' ','é','`','\\'}
for _=1,1200 do
  local pieces={}; for _=1,math.random(0,16) do pieces[#pieces+1]=atoms[math.random(#atoms)] end
  local line=table.concat(pieces)
  for _, item in ipairs(row(line)) do
    assert(item.start>=0 and item.stop<=#line and item.start<item.stop,'marker in bounds')
    local last=-1
    local spans={}; for _, h in ipairs(item.hidden or {}) do spans[#spans+1]=h end
    if item.visible then spans[#spans+1]=item.visible end
    table.sort(spans,function(x,y) return x[1]<y[1] end)
    for _, span in ipairs(spans) do
      assert(span[1]>=last and span[1]<=span[2] and span[2]<=#line,'disjoint bounds')
      for _, c in ipairs({span[1],span[2]}) do
        assert(c==#line or vim.str_utf_start(line,c+1)==0,'UTF8 boundary')
      end
      last=span[2]
    end
  end
end
print('comment_view_test ok')

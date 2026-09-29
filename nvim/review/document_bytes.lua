-- Pure UTF-8 document-byte decoding. Neovim stores line separators and the BOM
-- in buffer options, not in line text; retaining both duplicates them on write.
local M={}
local function utf8(bytes)
  local i=1
  while i<=#bytes do
    local a,b,c,d=bytes:byte(i,i+3)
    if a<128 then i=i+1
    elseif a>=194 and a<=223 and b and b>=128 and b<=191 then i=i+2
    elseif a>=224 and a<=239 and b and c and c>=128 and c<=191
      and b>=(a==224 and 160 or 128) and b<=(a==237 and 159 or 191) then i=i+3
    elseif a>=240 and a<=244 and b and c and d and c>=128 and c<=191 and d>=128 and d<=191
      and b>=(a==240 and 144 or 128) and b<=(a==244 and 143 or 191) then i=i+4
    else return false end
  end
  return true
end
function M.decode(bytes)
  if type(bytes)~='string' or bytes:find('\0',1,true) or not utf8(bytes) then return nil,'review document must use valid UTF-8; unsupported encoding preserved' end
  local bomb=bytes:sub(1,3)=='\239\187\191'
  if bomb then bytes=bytes:sub(4) end
  local dos=bytes:find('\r\n',1,true) and not bytes:gsub('\r\n',''):find('\n',1,true)
  local lines,pos={},1
  while true do
    local nextline=bytes:find('\n',pos,true)
    if not nextline then break end
    local line=bytes:sub(pos,nextline-1)
    if dos then line=line:sub(1,-2) end -- strip this CRLF delimiter's CR only
    lines[#lines+1]=line
    pos=nextline+1
  end
  if pos<=#bytes then lines[#lines+1]=bytes:sub(pos) end
  if #lines==0 then lines={''} end
  return {lines=lines,endofline=bytes:sub(-1)=='\n',fileformat=dos and 'dos' or 'unix',
    fileencoding='utf-8',bomb=bomb,fixendofline=false}
end
function M.encode(lines,options)
  if options.fileencoding~='' and options.fileencoding~='utf-8' then return nil,'unsupported review output encoding' end
  local separator=({unix='\n',dos='\r\n',mac='\r'})[options.fileformat]
  if not separator then return nil,'unsupported review line format' end
  local bytes=table.concat(lines,separator)..(options.endofline and separator or '')
  if not utf8(bytes) or bytes:find('\0',1,true) then return nil,'review output must be valid UTF-8 text' end
  return (options.bomb and '\239\187\191' or '')..bytes
end
return M

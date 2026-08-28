local M = {}

-- Canonical cross-language identity projection for operator-authored text.
-- Keep this byte-identical to sessioninventory.NormalizePairText; both consume
-- the shared versioned golden in testdata/normalization.
function M.normalize_pair_text(text)
  text = text:gsub('\r\n', '\n')
  local out = {}
  for line in (text .. '\n'):gmatch('([^\n]*)\n') do
    line = line:gsub('[ \t]+$', '')
    if not line:match('^[ \t]*===') then out[#out + 1] = line end
  end
  while #out > 0 and out[1]:match('^[ \t\r\v\f]*$') do table.remove(out, 1) end
  while #out > 0 and out[#out]:match('^[ \t\r\v\f]*$') do table.remove(out) end
  return table.concat(out, '\n')
end

return M

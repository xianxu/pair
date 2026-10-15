-- Editor-owned filesystem watches end before Neovim's global loop teardown.
local M = {}

function M.start(editor, path, callback)
  local handle = editor.loop.new_fs_event()
  if not handle then return nil end
  local closed = false
  local function close()
    if closed then return end
    closed = true
    if not handle:is_closing() then
      handle:stop()
      handle:close()
    end
  end
  local ok, result = pcall(handle.start, handle, path, {}, function(err, filename)
    if closed then return end
    editor.schedule(function()
      if not closed then callback(err, filename) end
    end)
  end)
  if not ok or result == nil then close(); return nil end
  editor.api.nvim_create_autocmd('VimLeavePre', { once = true, callback = close })
  return close
end

return M

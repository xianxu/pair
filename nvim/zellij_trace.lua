-- nvim/zellij_trace.lua — trace pair-originated zellij action calls.
--
-- Records are diagnostic only. They intentionally redact prompt-bearing args:
-- enough metadata to correlate with zellij.log, not enough to leak the prompt.
local M = {}

local function now_iso()
  return os.date('!%Y-%m-%dT%H:%M:%SZ')
end

local function data_dir()
  return vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME and (vim.env.HOME .. '/.local/share')) or '/tmp') .. '/pair')
end

local function trace_path()
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'unknown'
  return data_dir() .. '/zellij-actions-' .. tag .. '.jsonl'
end

local function redact_argv(argv, redact)
  local out = {}
  local redacted = {}
  for i, v in ipairs(argv) do
    local s = tostring(v)
    if redact and redact[i] ~= nil then
      local body = tostring(redact[i])
      out[i] = '<redacted:body>'
      redacted[#redacted + 1] = {
        index = i,
        body_len = #body,
        body_sha256_12 = vim.fn.sha256(body):sub(1, 12),
      }
    else
      out[i] = s
    end
  end
  return out, redacted
end

local function append_record(record)
  if vim.env.PAIR_ZELLIJ_TRACE == '0' then return end
  local dir = data_dir()
  pcall(vim.fn.mkdir, dir, 'p')
  local f = io.open(trace_path(), 'a')
  if not f then return end
  f:write(vim.json.encode(record), '\n')
  f:close()
end

function M.action(label, argv, opts)
  opts = opts or {}
  local start = vim.loop.hrtime()
  local stdout = vim.fn.system(argv)
  local elapsed_ms = math.floor(((vim.loop.hrtime() - start) / 1000000) + 0.5)
  local code = vim.v.shell_error
  local safe_argv, redacted = redact_argv(argv, opts.redact)
  append_record({
    ts = now_iso(),
    component = opts.component or 'nvim',
    label = label,
    argv = safe_argv,
    redacted = redacted,
    duration_ms = elapsed_ms,
    exit_code = code,
    stdout_bytes = #(stdout or ''),
    stderr_bytes = vim.NIL,
  })
  return { code = code, stdout = stdout }
end

return M

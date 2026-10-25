-- nvim/zellij_trace.lua — trace pair-originated zellij action calls.
--
-- Records are diagnostic only. They intentionally redact prompt-bearing args:
-- enough metadata to correlate with zellij.log, not enough to leak the prompt.
local M = {}

local function now_iso()
  return os.date('!%Y-%m-%dT%H:%M:%SZ')
end

local function trace_path()
  return vim.env.PAIR_ZELLIJ_ACTIONS_PATH
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
  local path = trace_path()
  if not path or path == '' then return end
  pcall(vim.fn.mkdir, vim.fn.fnamemodify(path, ':h'), 'p')
  local f = io.open(path, 'a')
  if not f then return end
  f:write(vim.json.encode(record), '\n')
  f:close()
end

-- Keep native argv as data. All bounded native calls share Pair's admission
-- helper; callers retain their existing timeout and output collection boundary.
function M.command(argv, pair)
  if argv[1] ~= 'zellij' then return argv end
  -- Installed runtime bundles contain assets, not the Pair executable. The
  -- launcher puts its executable directory first in PATH for these children.
  local executable = pair or 'pair'
  if pair and vim.fn.executable(pair) ~= 1 then executable = 'pair' end
  local command = {executable,
    'internal', 'zellij-command', '--'}
  for i = 2, #argv do command[#command + 1] = argv[i] end
  return command
end

function M.system(argv)
  return vim.fn.system(M.command(argv))
end

function M.action(label, argv, opts)
  opts = opts or {}
  local start = vim.loop.hrtime()
  local stdout
  local code
  if opts.timeout_ms ~= nil and opts.timeout_ms <= 0 then
    stdout, code = '', 124
  elseif type(_G.PairTestZellijExecutor) == 'function' then
    local result = _G.PairTestZellijExecutor(label, argv, opts) or {}
    stdout = result.stdout or ''
    code = result.code
  elseif opts.timeout_ms ~= nil then
    local result = vim.system(M.command(argv), {text=true, timeout=math.max(1, math.floor(opts.timeout_ms))}):wait()
    stdout, code = result.stdout or '', result.code
  else
    stdout = M.system(argv)
    code = vim.v.shell_error
  end
  local elapsed_ms = math.floor(((vim.loop.hrtime() - start) / 1000000) + 0.5)
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

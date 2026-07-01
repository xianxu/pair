-- nvim/scrollback.lua — read-only viewer for pair-scrollback-render output.
--
-- Loaded via `nvim -u $PAIR_HOME/nvim/scrollback.lua <path-to-.ansi>`.
-- The .ansi file has SGR escapes inline (`\x1b[...m`); this init strips
-- them from the buffer and re-applies their effect via extmarks so the
-- visible content matches what the agent rendered, line numbers match
-- zellij's scroll indicator, and `:880` jumps where the user expects.
--
-- Filed as #000017 M4. Plugin-free on purpose: this file is the entire
-- viewer. ~150 lines is small enough to audit in one pass; pulling in
-- baleia.nvim or AnsiEsc.vim would add a vendor dir for marginal gain.

vim.opt.termguicolors = true
vim.opt.compatible = false
-- The scrollback viewer is launched with `nvim -u scrollback.lua`, which
-- skips init.lua — so the editor-wide `wrap`/`linebreak` defaults set
-- there don't apply here. Repeat them so long lines (and any 🤖[…]
-- markers the user has dropped into them) wrap at whitespace instead of
-- mid-word.
vim.opt.wrap = true
vim.opt.linebreak = true
vim.opt.breakindent = true

-- Smart-case search: `/foo` matches Foo/FOO/foo; `/Foo` matches only Foo.
-- Standard nvim idiom (ignorecase + smartcase together) — same defaults
-- a reader would expect from any modern editor's incremental search.
vim.opt.ignorecase = true
vim.opt.smartcase = true

local annotate

-- Stub out the pair-launcher cmdline targets so a stray zellij Alt+Up
-- / Alt+Down / Alt+x / Alt+n / Alt+d / Shift+Alt+N pressed while the
-- scrollback viewer is the focused pane degrades silently rather than
-- erroring on an undefined global.
--
-- zellij's bindings for those chords issue `MoveFocus Down` and then
-- `WriteChars ":lua PairLayoutBigger()"` (plus a CR). The MoveFocus
-- assumes the destination is the draft pane (where init.lua defines
-- the Pair* globals), but from inside the Alt+/ floating viewer the
-- focus shift doesn't escape the floating layer — the cmdline call
-- lands here instead. Defining the names as no-ops at scrollback-
-- nvim scope side-steps the error without touching zellij's
-- assumptions. Performing the actual layout / quit / restart action
-- from within scrollback would be the wrong default anyway; the user
-- almost certainly meant to act on the draft.
for _, name in ipairs({
  'PairLayoutBigger', 'PairLayoutSmaller',
  'PairConfirmQuit',  'PairConfirmDetach',
  'PairConfirmRestart', 'PairConfirmRestartNewSession',
}) do
  _G[name] = function() end
end

-- See init.lua for the full rationale: this writes the embed nvim's pid to
-- $PAIR_NVIM_PID_FILE so bin/pair's cleanup_quit_marker can reap it on
-- Alt+x without resorting to argv pattern matching. Scrollback nvim has
-- the same TUI/embed fork shape as the draft, so the same leak applies if
-- the user Alt+x's while this viewer is floating.
do
  local pidfile = vim.env.PAIR_NVIM_PID_FILE
  if pidfile and pidfile ~= '' then
    vim.api.nvim_create_autocmd('VimEnter', {
      once = true,
      callback = function()
        pcall(vim.fn.writefile, { tostring(vim.fn.getpid()) }, pidfile)
      end,
    })
  end
end

-- Standard 16-color palette. Approximate xterm defaults — close enough
-- to most TUI palettes that the rendered colors feel right. No user
-- configuration; the file is supposed to look like the agent's output.
local PALETTE = {
  ['30'] = '#000000', ['31'] = '#cc0000', ['32'] = '#4e9a06', ['33'] = '#c4a000',
  ['34'] = '#3465a4', ['35'] = '#75507b', ['36'] = '#06989a', ['37'] = '#d3d7cf',
  ['90'] = '#555753', ['91'] = '#ef2929', ['92'] = '#8ae234', ['93'] = '#fce94f',
  ['94'] = '#729fcf', ['95'] = '#ad7fa8', ['96'] = '#34e2e2', ['97'] = '#eeeeec',
  ['40'] = '#000000', ['41'] = '#cc0000', ['42'] = '#4e9a06', ['43'] = '#c4a000',
  ['44'] = '#3465a4', ['45'] = '#75507b', ['46'] = '#06989a', ['47'] = '#d3d7cf',
  ['100'] = '#555753', ['101'] = '#ef2929', ['102'] = '#8ae234', ['103'] = '#fce94f',
  ['104'] = '#729fcf', ['105'] = '#ad7fa8', ['106'] = '#34e2e2', ['107'] = '#eeeeec',
}

-- Apply one SGR `m` escape's parameter list to a state table. Mutates
-- `state` in place; codes consume up to 4 list slots for 38;2;r;g;b
-- truecolor and 38;5;n indexed (which we map approximately to RGB via
-- a small 256-color formula — see resolve_256).
local function resolve_256(n)
  if n < 16 then
    -- Standard 16: pull from the same xterm palette as named codes.
    local key = (n < 8) and tostring(30 + n) or tostring(90 + n - 8)
    return PALETTE[key]
  elseif n < 232 then
    -- 6×6×6 cube
    n = n - 16
    local r = math.floor(n / 36)
    local g = math.floor((n % 36) / 6)
    local b = n % 6
    local function comp(c) if c == 0 then return 0 else return 55 + c * 40 end end
    return string.format('#%02x%02x%02x', comp(r), comp(g), comp(b))
  else
    -- Greyscale ramp 232-255
    local v = 8 + (n - 232) * 10
    return string.format('#%02x%02x%02x', v, v, v)
  end
end

local function apply_sgr(state, params)
  -- Empty params == reset (`\x1b[m`).
  if params == '' then
    state.fg, state.bg = nil, nil
    state.bold, state.italic, state.underline, state.reverse, state.strike = false, false, false, false, false
    return
  end
  -- Match every `;`-delimited field, *including empty ones*. ECMA-48
  -- treats an omitted SGR parameter as 0 (reset), so `\x1b[;1m` is
  -- "reset + bold" — splitting on `[^;]+` would drop the empty leading
  -- field and leave any standing fg/bg/decoration in place. Appending
  -- a trailing `;` and matching `([^;]*);` makes every field — empty
  -- or not — a separate code.
  local codes = {}
  for code in (params .. ';'):gmatch('([^;]*);') do
    table.insert(codes, code)
  end
  local i = 1
  while i <= #codes do
    local n = tonumber(codes[i])
    -- Empty field == 0 (reset).
    if codes[i] == '' then n = 0 end
    if n == nil then i = i + 1 ; goto continue end
    if n == 0 then
      state.fg, state.bg = nil, nil
      state.bold, state.italic, state.underline, state.reverse, state.strike = false, false, false, false, false
    elseif n == 1 then state.bold = true
    elseif n == 22 then state.bold = false
    elseif n == 3 then state.italic = true
    elseif n == 23 then state.italic = false
    elseif n == 4 then state.underline = true
    elseif n == 24 then state.underline = false
    elseif n == 7 then state.reverse = true
    elseif n == 27 then state.reverse = false
    elseif n == 9 then state.strike = true
    elseif n == 29 then state.strike = false
    elseif (n >= 30 and n <= 37) or (n >= 90 and n <= 97) then
      state.fg = PALETTE[tostring(n)]
    elseif n == 39 then state.fg = nil
    elseif (n >= 40 and n <= 47) or (n >= 100 and n <= 107) then
      state.bg = PALETTE[tostring(n)]
    elseif n == 49 then state.bg = nil
    elseif n == 38 or n == 48 then
      local mode = tonumber(codes[i + 1] or '')
      if mode == 5 and codes[i + 2] then
        local idx = tonumber(codes[i + 2])
        if idx then
          if n == 38 then state.fg = resolve_256(idx) else state.bg = resolve_256(idx) end
        end
        i = i + 2
      elseif mode == 2 and codes[i + 4] then
        local r = tonumber(codes[i + 2]) or 0
        local g = tonumber(codes[i + 3]) or 0
        local b = tonumber(codes[i + 4]) or 0
        local hex = string.format('#%02x%02x%02x', r, g, b)
        if n == 38 then state.fg = hex else state.bg = hex end
        i = i + 4
      end
    end
    ::continue::
    i = i + 1
  end
end

-- Cache resolved (state → hl-group-name) so we don't create duplicate
-- highlight groups for every cell. Separate counter because hl_cache
-- is string-keyed (`#` on a dict returns 0, not the entry count).
local hl_cache = {}
local hl_counter = 0

local function hl_for(state)
  -- Build a stable cache key from the state attrs.
  local key = (state.fg or '_') .. '|' .. (state.bg or '_')
    .. '|' .. (state.bold and 'B' or '')
    .. (state.italic and 'I' or '')
    .. (state.underline and 'U' or '')
    .. (state.reverse and 'R' or '')
    .. (state.strike and 'S' or '')
  if hl_cache[key] then return hl_cache[key] end
  hl_counter = hl_counter + 1
  local name = 'PairScrollback_' .. hl_counter
  local def = {}
  if state.fg then def.fg = state.fg end
  if state.bg then def.bg = state.bg end
  if state.bold then def.bold = true end
  if state.italic then def.italic = true end
  if state.underline then def.underline = true end
  if state.reverse then def.reverse = true end
  if state.strike then def.strikethrough = true end
  vim.api.nvim_set_hl(0, name, def)
  hl_cache[key] = name
  return name
end

-- Walk one line: peel off SGR escapes (mutating a running state),
-- accumulate the visible bytes into stripped, and emit extmark spans
-- for each contiguous run of visible bytes under a single state.
local SGR = '\27%[[%d;]*m'

local function process_line(line)
  local stripped_parts = {}
  local spans = {} -- { {col_start, col_end, hl_group} ... } using BYTE indices
  local state = { fg = nil, bg = nil, bold = false, italic = false,
                  underline = false, reverse = false, strike = false }
  local i = 1
  local stripped_len = 0
  while i <= #line do
    local s, e = line:find(SGR, i)
    if s and s > i then
      -- Plain text up to the next escape.
      local seg = line:sub(i, s - 1)
      table.insert(stripped_parts, seg)
      local col_start = stripped_len
      stripped_len = stripped_len + #seg
      table.insert(spans, { col_start, stripped_len, hl_for(state) })
      i = s
    elseif s == i then
      -- Escape at current cursor.
      local params = line:sub(s + 2, e - 1)
      apply_sgr(state, params)
      i = e + 1
    else
      -- Tail with no more escapes.
      local seg = line:sub(i)
      table.insert(stripped_parts, seg)
      local col_start = stripped_len
      stripped_len = stripped_len + #seg
      table.insert(spans, { col_start, stripped_len, hl_for(state) })
      i = #line + 1
    end
  end
  return table.concat(stripped_parts), spans
end

local function decorate_buffer(bufnr)
  local ns = vim.api.nvim_create_namespace('pair_scrollback')
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  local stripped = {}
  local all_spans = {} -- per-line spans
  for idx, raw in ipairs(lines) do
    local s, sp = process_line(raw)
    stripped[idx] = s
    all_spans[idx] = sp
  end
  -- Replace buffer content with stripped text first; extmarks attach
  -- to the new content.
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, stripped)
  vim.api.nvim_buf_clear_namespace(bufnr, ns, 0, -1)
  for idx, sp in ipairs(all_spans) do
    for _, span in ipairs(sp) do
      local col_start, col_end, hl = span[1], span[2], span[3]
      if col_end > col_start then
        vim.api.nvim_buf_set_extmark(bufnr, ns, idx - 1, col_start, {
          end_col = col_end,
          hl_group = hl,
        })
      end
    end
  end
end

local function scrollback_paths(bufnr)
  local ansi = vim.api.nvim_buf_get_name(bufnr)
  if not ansi or ansi == '' or not ansi:match('%.ansi$') then
    return nil, 'current buffer is not a scrollback .ansi file'
  end
  return {
    ansi = ansi,
    raw = ansi:gsub('%.ansi$', '.raw'),
    events = ansi:gsub('%.ansi$', '.events.jsonl'),
  }
end

local function renderer_command(paths)
  -- One binary now: `pair scrollback-render` via the dispatcher (#92).
  local bin
  if vim.env.PAIR_HOME and vim.env.PAIR_HOME ~= '' then
    bin = vim.env.PAIR_HOME .. '/bin/pair'
  else
    bin = 'pair'
  end
  return { bin, 'scrollback-render', paths.raw, paths.events, paths.ansi }
end

local function run_renderer(paths, opts)
  opts = opts or {}
  if opts.renderer then
    local ok, result, err = pcall(opts.renderer, paths)
    if not ok then return false, tostring(result) end
    if result == false then return false, err or 'renderer failed' end
    return true
  end

  local cmd = renderer_command(paths)
  local out = vim.fn.system(cmd)
  if vim.v.shell_error ~= 0 then
    return false, table.concat(cmd, ' ') .. ' failed: ' .. out
  end
  return true
end

local function relock_scrollback_buffer(bufnr)
  if not vim.api.nvim_buf_is_valid(bufnr) then return end
  vim.bo[bufnr].modifiable = false
  vim.bo[bufnr].readonly = true
  vim.bo[bufnr].buftype = 'nofile'
  vim.bo[bufnr].swapfile = false
end

local function refresh_scrollback_buffer(bufnr, opts)
  opts = opts or {}
  bufnr = bufnr or vim.api.nvim_get_current_buf()
  local paths, path_err = scrollback_paths(bufnr)
  if not paths then
    vim.notify('scrollback refresh failed: ' .. path_err, vim.log.levels.WARN)
    return false
  end

  local old_lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  local ok, render_err = run_renderer(paths, opts)
  if not ok then
    vim.notify('scrollback refresh failed: ' .. (render_err or 'renderer failed'), vim.log.levels.WARN)
    relock_scrollback_buffer(bufnr)
    return false
  end

  local read_ok, new_lines = pcall(vim.fn.readfile, paths.ansi)
  if not read_ok then
    vim.bo[bufnr].modifiable = true
    vim.bo[bufnr].readonly = false
    vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, old_lines)
    relock_scrollback_buffer(bufnr)
    vim.notify('scrollback refresh failed: could not read rendered file', vim.log.levels.WARN)
    return false
  end

  if annotate.has_pending_annotations and annotate.has_pending_annotations(bufnr) then
    vim.notify('scrollback refresh skipped buffer reload: pending annotations kept', vim.log.levels.INFO)
    relock_scrollback_buffer(bufnr)
    return true
  end

  vim.bo[bufnr].modifiable = true
  vim.bo[bufnr].readonly = false
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, new_lines)
  decorate_buffer(bufnr)
  annotate.on_reloaded(bufnr)
  relock_scrollback_buffer(bufnr)
  return true
end

local function refresh_then_end(bufnr, opts)
  bufnr = bufnr or vim.api.nvim_get_current_buf()
  local ok = refresh_scrollback_buffer(bufnr, opts)
  if not ok then return false end
  local last = vim.api.nvim_buf_line_count(bufnr)
  if last > 0 then
    pcall(vim.api.nvim_win_set_cursor, 0, { last, 0 })
    vim.cmd('normal! zb')
  end
  return true
end

-- Per-agent user-prompt marker. Used by Alt+b to skip backward through
-- the user's turns in the scrollback. vim's regex engine (which
-- vim.fn.search uses) is UTF-8 aware, so the literal character in the
-- pattern works. Each agent paints its prompt-input line with a
-- distinct leading glyph:
--   claude — ❯  (U+276F, HEAVY RIGHT-POINTING ANGLE QUOTATION MARK)
--   codex  — ›  (U+203A, SINGLE RIGHT-POINTING ANGLE QUOTATION MARK)
-- Lookup falls back to claude's pattern so unknown agents still get a
-- useful default.
local PROMPT_PATTERN_BY_AGENT = {
  claude = [[^❯]],
  codex  = [[^›]],
  agy    = [[\(──.*\n\)\zs>]],
}

-- Adaptation flight recorder (atlas §3). Load the sibling emitter by this
-- file's own directory so it works however scrollback.lua was launched
-- (`nvim -u .../scrollback.lua`). Best-effort: a load failure leaves a no-op
-- stub so telemetry never breaks the viewer.
local adapt
do
  local src = debug.getinfo(1, 'S').source:sub(2) -- strip leading '@'
  local dir = src:match('(.*/)') or './'
  local ok, mod = pcall(dofile, dir .. 'adapt.lua')
  adapt = (ok and mod) or { log = function() end }
end

-- The shared 🤖-marker subsystem (#57) — same dir-relative load as adapt, so
-- it resolves however scrollback.lua was launched (`nvim -u .../scrollback.lua`).
-- Owns the marker parse/extract core + the Alt+q add/edit flow + the quit-emit
-- to the draft sidecar; this file keeps only the SGR rendering, Alt+b prompt
-- jump, and the scrollback-specific viewport/key safety below.
do
  local src = debug.getinfo(1, 'S').source:sub(2)
  local dir = src:match('(.*/)') or './'
  annotate = dofile(dir .. 'annotate.lua')
end
local prompt_nearmiss_logged = false

local function prompt_pattern()
  local agent = vim.env.PAIR_AGENT or ''
  return PROMPT_PATTERN_BY_AGENT[agent] or PROMPT_PATTERN_BY_AGENT.claude
end

-- Alt+b / Alt+Shift+B: search backward / forward for the next line
-- starting with the per-agent prompt marker. Lands on the prompt line
-- and pulls it to the top of the window (zt) so the response below
-- stays in view. No-wrap: hitting either end of scrollback with no
-- further matches reports "no {previous,next} prompt" rather than
-- silently jumping to the other end.
local function jump_to_prompt(direction)
  local pat = prompt_pattern()
  -- 'b' = backward, omitted = forward. 'W' = no wrap.
  local flags = (direction == 'prev') and 'bW' or 'W'
  if vim.fn.search(pat, flags) == 0 then
    vim.notify('no ' .. direction .. ' prompt', vim.log.levels.INFO)
    -- Drift fingerprint (aspect 7): a `search` miss in one direction is
    -- ordinary end-of-scrollback. But if the pattern matches NOWHERE in a
    -- non-empty buffer (the 'nw' = no-move, wrap probe also returns 0), the
    -- agent's prompt glyph likely changed and Alt+b can never jump. Log once.
    if not prompt_nearmiss_logged
        and vim.fn.line('$') > 1
        and vim.fn.search(pat, 'nw') == 0 then
      prompt_nearmiss_logged = true
      adapt.log(7, 'prompt-search', 'near-miss',
        'prompt pattern matched 0 lines in scrollback (agent=' .. (vim.env.PAIR_AGENT or '') .. ')')
    end
  else
    adapt.log(7, 'prompt-search', 'fired', direction)
    vim.cmd('normal! zt')
  end
end


-- Expose the scrollback-specific prompt pattern for the headless test
-- (nvim/scrollback_test.lua). The marker core moved to annotate.lua and is
-- tested directly in nvim/annotate_test.lua.
_G.PairScrollbackTest = {
  prompt_pattern = prompt_pattern,
  scrollback_paths = scrollback_paths,
  refresh_buffer = refresh_scrollback_buffer,
  refresh_then_end = refresh_then_end,
  annotate = annotate,
}


-- Wire it up: when the file is loaded, decorate then lock the buffer.
vim.api.nvim_create_autocmd('BufReadPost', {
  pattern = '*',
  callback = function(args)
    local bufnr = args.buf
    decorate_buffer(bufnr)
    -- Lock the buffer read-only before annotate attaches; its add/edit flow
    -- does the unlock→insert→relock dance against a locked buffer.
    vim.bo[bufnr].modifiable = false
    vim.bo[bufnr].readonly = true
    vim.bo[bufnr].buftype = 'nofile'  -- prevent accidental :w
    vim.bo[bufnr].swapfile = false
    -- Shared 🤖-marker subsystem (#57): snapshots the load-time baseline,
    -- highlights existing markers, appends the overall-comment footer
    -- affordance (#21, scrollback-only), wires Alt+q (normal + visual), and
    -- emits to the draft sidecar on quit. footer=true + the 'scrollback'
    -- quit-noun keep the UX byte-identical to pre-#57; the default pending
    -- path is the same file the draft pane picks up on FocusGained.
    annotate.attach({
      bufnr = bufnr,
      pending_path = annotate.default_pending_path(),
      footer = true,
      quit_noun = 'scrollback',
    })
    -- ESC is the only quit binding. `q` was tempting (less-style, no
    -- Esc-prefix timeout) but a fat-fingered `q` instead of `Alt+q`
    -- (the marker-comment binding) was a frequent footgun — one mistype
    -- and the viewer slammed shut, dropping any pending markers along with
    -- the session. Built-in `ZZ` / `ZQ` are shadowed with no-ops for the
    -- same reason. The shared confirm gate prompts *only when* there's
    -- something to ship (user-added markers beyond the baseline, or a footer
    -- comment); passive reads quit instantly, no friction.
    vim.keymap.set('n', '<Esc>', function() annotate.confirm_quit(bufnr) end,
                   { buffer = bufnr, silent = true })
    vim.keymap.set('n', 'ZZ', '<nop>', { buffer = bufnr, silent = true })
    vim.keymap.set('n', 'ZQ', '<nop>', { buffer = bufnr, silent = true })
    vim.keymap.set('n', '<M-b>', function() jump_to_prompt('prev') end,
                   { buffer = bufnr, silent = true })
    vim.keymap.set('n', '<M-B>', function() jump_to_prompt('next') end,
                   { buffer = bufnr, silent = true })
    vim.keymap.set('n', 'G', function() refresh_then_end(bufnr) end,
                   { buffer = bufnr, silent = true })
    -- Disable the Alt-arrow / Shift-Alt-arrow combos. Two distinct
    -- failure modes had to be neutralised:
    --   • <M-Up> / <M-Down>: nvim's defaults bind these to "move line
    --     up / down", which errors loudly on the scrollback's read-
    --     only buffer.
    --   • <M-Left> / <M-Right> (+ Shift variants): terminals emit
    --     them as `ESC <Arrow>` and the two bytes can split apart
    --     past ttimeoutlen — the ESC half then fires our exit binding
    --     and the viewer silently closes.
    -- Mapping all of them to <nop> at buffer scope takes the default
    -- handler out of the picture for the first class; bumping
    -- ttimeoutlen below maximises the chance nvim fuses ESC + Arrow
    -- into a single <M-Arrow> chord before our <Esc> handler sees a
    -- bare ESC.
    for _, key in ipairs({
      '<M-Up>', '<M-Down>', '<M-Left>', '<M-Right>',
      '<M-S-Up>', '<M-S-Down>', '<M-S-Left>', '<M-S-Right>',
    }) do
      vim.keymap.set({ 'n', 'x' }, key, '<nop>', { buffer = bufnr, silent = true })
    end
    vim.opt.ttimeoutlen = 100

    -- Hide the "end of buffer" tilde marker. The viewer is read-only,
    -- so the standard vim convention of marking past-EOF rows with `~`
    -- only invites confusion ("is there content I'm missing?") and is
    -- never going to receive a write. Replacing the marker with a
    -- space gives those rows the look of empty terminal cells, which
    -- is visually closer to what they actually represent.
    --
    -- An earlier attempt clamped the scroll position via CursorMoved
    -- + WinScrolled autocmds so the last line stayed pinned at-or-
    -- above the bottom of the window. That correctly suppressed the
    -- `~` rows but caused noticeable flicker on j / Ctrl-D scrolls
    -- (the clamp fired *after* the scroll repositioned the view,
    -- so the user saw a brief jump-and-snap-back). The eob fillchar
    -- approach achieves the same visual outcome with zero
    -- scroll-loop interference.
    vim.opt.fillchars:append({ eob = ' ' })

    -- Align the viewer's top with the agent pane's current viewport.
    -- The renderer writes a sibling `<ansi>.viewport` containing the
    -- 1-indexed line where the visible buffer (em.Height() rows at the
    -- bottom of the emulator) starts in the rendered output — i.e.
    -- the first line the user was just looking at in the agent pane.
    -- Position cursor there and `zt` to top so opening Alt+/ feels
    -- like a seamless lift of the current screen into a scrollable
    -- viewer.
    --
    -- Fallback: if the sidecar is missing or out of range, drop to
    -- the original "G + zb" bottom-anchored behaviour so a stale
    -- pair install with the new viewer but the old renderer still
    -- opens to something useful.
    --
    -- The positioning runs on `vim.schedule` rather than inline because
    -- a synchronous `zt` during BufReadPost can be undone by a window-
    -- size update that fires later in the same tick (the floating pane
    -- only finalises its layout after the file is loaded), leaving the
    -- viewer scrolled to wherever the layout-change put it instead of
    -- our requested top. Deferring one tick puts our positioning *after*
    -- those layout passes complete.
    local ansi_path = vim.api.nvim_buf_get_name(bufnr)
    local vp_path = ansi_path:match('%.ansi$')
      and (ansi_path:gsub('%.ansi$', '.viewport'))
      or nil
    vim.schedule(function()
      if not vim.api.nvim_buf_is_valid(bufnr) then return end
      local last = vim.api.nvim_buf_line_count(bufnr)
      local top
      if vp_path then
        local f = io.open(vp_path, 'r')
        if f then
          top = tonumber(f:read('*l'))
          f:close()
        end
      end
      if top and top >= 1 and top <= last then
        pcall(vim.api.nvim_win_set_cursor, 0, { top, 0 })
        vim.cmd('normal! zt')
      elseif last > 0 then
        pcall(vim.api.nvim_win_set_cursor, 0, { last, 0 })
        vim.cmd('normal! zb')
      end
      -- Alt+/-then-Alt+b shortcut: the draft pane opened us with
      -- PAIR_SCROLLBACK_JUMP set. Jump from the just-positioned cursor
      -- exactly as a manual Alt+b/Alt+B would — same starting point, so
      -- the shortcut is behaviourally identical to the two-key sequence.
      local jump = vim.env.PAIR_SCROLLBACK_JUMP
      if jump == 'prev' or jump == 'next' then
        jump_to_prompt(jump)
      end
    end)
  end,
})


-- Reclaim every available column for the rendered scrollback: the
-- agent pane writes lines that nearly fill its width, and any nvim
-- gutter (line numbers, sign column) would push the tail of those
-- lines into a wrap. `:N` still jumps to line N without a visible
-- gutter — line position lives in the statusline instead.
vim.opt.number = false
vim.opt.relativenumber = false
vim.opt.signcolumn = 'no'
vim.opt.foldcolumn = '0'
vim.opt.cursorline = true
-- Route yank/delete to the system clipboard. The viewer launches with
-- `nvim -u nvim/scrollback.lua`, which bypasses the user's regular
-- init — so any clipboard config there doesn't apply here. Without
-- this, `y` lands in the unnamed register only and cmd+V outside the
-- terminal pulls whatever was on the system clipboard before. Neovim
-- autodetects pbcopy/pbpaste on macOS so no provider config needed.
vim.opt.clipboard = 'unnamedplus'
vim.opt.laststatus = 2
vim.opt.statusline = ' pair scrollback · Esc quit (confirm) · G refresh/end · Alt+q 🤖[] (add/edit) · Alt+b/B prompts · :N jump %= L%l/%L '

-- pair/nvim/init.lua
-- Minimal nvim config for the pair input pane. Loaded via `nvim -u`,
-- so this is fully isolated from the user's normal nvim setup.

vim.g.mapleader = ' '

_G.PairZellijTrace = dofile(vim.fn.fnamemodify(debug.getinfo(1, 'S').source:sub(2), ':p:h') .. '/zellij_trace.lua')

-- Publish this nvim's pid so the quit path can reap it deterministically.
-- `nvim FILE` forks into a TUI parent + `nvim --embed` server child; when
-- zellij kills the pane, the TUI exits but the embed sometimes survives
-- the RPC-pipe EOF and gets reparented to launchd, accumulating across
-- Alt+x quits until the host runs out of memory. `getpid()` here returns
-- the embed pid (lua runs inside the embed); bin/pair's cleanup_quit_marker
-- reads this file and SIGKILLs the recorded pid. Best-effort: a write
-- failure is silent so init.lua never blocks startup over a pidfile.
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

-- Enable filetype detection + default syntax. Loaded via `nvim -u`, which
-- doesn't bypass nvim's bundled runtime but doesn't auto-enable these
-- either. The draft file is `.md`, so this picks up markdown highlighting.
-- termguicolors is required for the default colorscheme's gui-defined
-- palette to render in the terminal — without it most syntax groups fall
-- back to a near-monochrome cterm palette.
vim.opt.termguicolors = true
vim.cmd('syntax enable')
vim.cmd('filetype plugin indent on')

-- Highlight fenced code blocks by language. Without this, vim's markdown
-- syntax leaves ```<lang> ... ``` bodies as plain monospace regardless of
-- the tag. The `alias=target` entries map common short forms to the canonical
-- syntax filename (vim looks for `syntax/<name>.vim` under runtimepath).
-- Read by runtime/syntax/markdown.vim when the buffer is opened, so this
-- must be set before the .md file loads — top-of-init is safe.
vim.g.markdown_fenced_languages = {
  'python', 'py=python',
  'javascript', 'js=javascript',
  'typescript', 'ts=typescript',
  'lua',
  'bash', 'sh=bash', 'zsh=bash',
  'json', 'yaml', 'yml=yaml', 'toml',
  'html', 'css',
  'c', 'cpp', 'cxx=cpp',
  'rust', 'rs=rust',
  'go',
  'sql',
  'ruby', 'rb=ruby',
  'java', 'kotlin', 'kt=kotlin',
  'swift',
  'dockerfile',
  'make',
  'diff',
  'vim',
}
-- nvim's bundled `default` colorscheme is intentionally near-monochrome —
-- syntax groups get bold/italic but no fg colors. habamax is bundled and
-- gives readable colors for markdown headings, code spans, links, etc.
vim.cmd('colorscheme slate')
-- Slate's stock Comment is `#666666` — under pair's dark insert-mode bg
-- (`#1c1c1c`) the contrast lands below WCAG AA, so `===` annotations and
-- inline comments in any embedded language render too faded to read at a
-- glance. Lift to a slightly brighter gray; italic conveys "annotation"
-- without leaning on color. Reapplied on ColorScheme so a future theme
-- swap can override.
local function pair_apply_comment_hl()
  vim.api.nvim_set_hl(0, 'Comment', {
    fg = '#999999', ctermfg = 247,
  })
end
pair_apply_comment_hl()
vim.api.nvim_create_autocmd('ColorScheme', { callback = pair_apply_comment_hl })

-- FileType=markdown setup. Runs after vim's stock markdown.vim, so any
-- syntax matches we add here aren't clobbered, and any `highlight! link`
-- we issue overrides stock's `hi def link`.
--
-- Two responsibilities:
--   1. The pair-specific `===` comment line — faded (Comment) in general,
--      but bold normal-fg when it's the first line.
--   2. Brighten the markdown groups stock vim leaves linked to htmlLink /
--      htmlH1 / etc., which only carry underline/bold and no color. Without
--      this, [link] text, headings, and emphasis render plain in slate.
vim.api.nvim_create_autocmd('FileType', {
  pattern = 'markdown',
  callback = function()
    -- Stock vim ftplugin/markdown.vim sets ts/sw/sts to 4 buffer-locally,
    -- overriding our 2-space globals. Re-apply 2 here so Tab and `>>`
    -- match the rest of the editor.
    vim.bo.tabstop = 2
    vim.bo.shiftwidth = 2
    vim.bo.softtabstop = 2
    vim.bo.expandtab = true

    vim.cmd([[syntax match PairComment /^\s*===.*$/]])
    vim.cmd([[highlight default link PairComment Comment]])
    -- A `===` line on the *first* line is the deliberate header annotation the
    -- user wants to notice, so render it bold instead of the faded Comment
    -- gray. Setting only the bold attribute (no fg) leaves the foreground at
    -- the default Normal color, so it's bold white-on-dark-grey. `\%1l`
    -- restricts the match to line 1; defined after PairComment so it wins the
    -- overlap there.
    vim.cmd([[syntax match PairCommentFirst /\%1l^\s*===.*$/]])
    vim.cmd([[highlight PairCommentFirst gui=bold cterm=bold]])

    -- `highlight!` (bang) overrides any existing link from stock syntax.
    vim.cmd([[highlight! link markdownH1 Title]])
    vim.cmd([[highlight! link markdownH2 Title]])
    vim.cmd([[highlight! link markdownH3 Title]])
    vim.cmd([[highlight! link markdownH4 Title]])
    vim.cmd([[highlight! link markdownH5 Title]])
    vim.cmd([[highlight! link markdownH6 Title]])
    vim.cmd([[highlight! link markdownHeadingDelimiter Title]])
    vim.cmd([[highlight! link markdownLinkText Identifier]])
    vim.cmd([[highlight! link markdownUrl Constant]])
    vim.cmd([[highlight! link markdownLinkTextDelimiter Comment]])
    vim.cmd([[highlight! link markdownLinkDelimiter Comment]])
    vim.cmd([[highlight! link markdownUrlDelimiter Comment]])
    vim.cmd([[highlight! link markdownId Type]])
    vim.cmd([[highlight! link markdownIdDeclaration Type]])
    vim.cmd([[highlight! link markdownBold Special]])
    vim.cmd([[highlight! link markdownItalic Special]])
    vim.cmd([[highlight! link markdownBlockquote Comment]])
    -- Render `> ` markdown blockquotes as a vertical bar, matching how
    -- typical markdown renderers (GitHub, Obsidian) draw a gutter bar.
    -- Stock markdownBlockquote matches `>\s` or `>$` as one chunk and
    -- only dims that prefix; we want the bar visual AND the whole line
    -- dimmed. Split into two contained matches: `^>` with conceal
    -- cchar=▎ (inherits Comment via the existing link above), and
    -- `.*$` as markdownBlockquoteText linked to Comment. conceallevel=2
    -- enables cchar replacement; concealcursor='nc' keeps the bar
    -- visible in normal/command mode while showing the real `>` in
    -- insert/visual so editing the prefix isn't disorienting.
    vim.cmd([[syntax clear markdownBlockquote]])
    vim.cmd([[syntax match markdownBlockquote /^>/ contained conceal cchar=▎ nextgroup=markdownBlockquoteText]])
    vim.cmd([[syntax match markdownBlockquoteText /.*$/ contained]])
    vim.cmd([[highlight! link markdownBlockquoteText Comment]])
    vim.wo.conceallevel = 2
    vim.wo.concealcursor = 'nc'
    vim.cmd([[highlight! link markdownListMarker Statement]])
    vim.cmd([[highlight! link markdownOrderedListMarker Statement]])
    vim.cmd([[highlight! link markdownRule NonText]])

    -- Stock markdown.vim defines markdownCode (inline `code`) as a region
    -- but never `hi def link`s it — so inline backticks render plain by
    -- default. Link to String for a distinct color.
    vim.cmd([[highlight! link markdownCode String]])
    vim.cmd([[highlight! link markdownCodeDelimiter Comment]])

    -- Stock syntax only fires markdownLinkText for the `[text](url)` /
    -- `[text][ref]` forms; bare `[text]` brackets (common in drafts —
    -- `[Image #1]`, shorthand mentions, etc.) get no highlight at all.
    -- Add a match scoped to bare brackets, with a negative lookahead so
    -- the `[text]` portion of a real `[text](url)` is still owned by the
    -- stock markdownLinkText region.
    vim.cmd([[syntax match markdownPairBracket /\[[^\]\n]\{-1,}\]\%(\s*[[(]\)\@!/]])
    vim.cmd([[highlight! link markdownPairBracket Identifier]])
  end,
})

-- Pin a leading `===` comment to the top of the window once it scrolls off, so
-- the header annotation stays visible while drafting a long prompt. Uses winbar
-- (a per-window top bar): empty while line 1 is on screen (it's its own header
-- there — showing it in the winbar too would duplicate it), populated with line
-- 1's text once the window has scrolled past it (`line('w0') > 1`). Scoped to
-- markdown (the compose buffer) and only when line 1 is actually a `===`
-- comment — other buffers' winbars are left untouched.
local function pair_pin_header()
  if vim.bo.filetype ~= 'markdown' then return end
  -- In the minimized rung nvim is a single row; a winbar won't fit (nvim
  -- raises E36: Not enough room) and the resulting error pages into a
  -- `-- More --` prompt that swallows the next keystroke — e.g. Alt+Up to
  -- grow back. Never set a winbar there; clear any stale one. Clearing is
  -- always safe (it frees a row); only adding one to a too-short window errs.
  if vim.o.lines <= 2 then
    if vim.wo.winbar ~= '' then vim.wo.winbar = '' end
    return
  end
  local first = vim.fn.getline(1)
  if first:match('^%s*===') and vim.fn.line('w0') > 1 then
    -- The winbar spans the full window width, but buffer text is inset by the
    -- gutter (number column). Match that inset so the pinned header lines up
    -- exactly under the real line 1. `textoff` is the gutter width nvim is
    -- currently using — it tracks the 1-vs-2-digit line-number swing for free,
    -- so no need to compute numberwidth ourselves.
    local info = vim.fn.getwininfo(vim.api.nvim_get_current_win())[1]
    local pad = string.rep(' ', (info and info.textoff) or 0)
    -- `%` is statusline-special; double it so the text renders literally.
    -- Wrap in PairWinbar (DiffAdd green) so the pinned header is visually set
    -- apart from the compose text below it.
    vim.wo.winbar = '%#PairWinbar#' .. pad .. first:gsub('%%', '%%%%') .. '%*'
  else
    vim.wo.winbar = ''
  end
end
vim.api.nvim_create_autocmd(
  { 'WinScrolled', 'CursorMoved', 'CursorMovedI', 'TextChanged', 'TextChangedI', 'BufWinEnter', 'WinEnter', 'VimResized' },
  { callback = pair_pin_header }
)

-- Drafting-friendly editor settings
vim.opt.number = true
vim.opt.relativenumber = false
vim.opt.numberwidth = 1
vim.opt.signcolumn = 'no'
-- laststatus=2 + a custom statusline are set later (after nav helpers); they
-- back the position indicator `History H < pos > Q Queued` for the prompt
-- history/queue feature.
vim.opt.cmdheight = 0    -- no permanent command line; appears on demand only

-- Suppress the "written" / file-info messages on :w. Without this, every
-- autosave or send-and-clear write briefly pops the cmdline up under
-- `cmdheight=0` and that interaction blanks the custom statusline until the
-- next redraw. With "W" + "F" the messages never fire, so the statusline
-- stays put.
vim.opt.shortmess:append('WF')
vim.opt.showmode = false -- nvim's default `-- INSERT --` line; redundant with cmdheight=0
vim.opt.ruler = false
vim.opt.wrap = true
vim.opt.linebreak = true
vim.opt.breakindent = true
-- Scroll by screen line, not buffer line, so a long wrapped draft line scrolls
-- smoothly (partial top line + a `<<<` marker) instead of jumping a whole
-- paragraph at once.
vim.opt.smoothscroll = true
vim.opt.spell = true
vim.opt.spelllang = 'en_us'
vim.opt.swapfile = false
vim.opt.backup = false
vim.opt.writebackup = false
vim.opt.clipboard = 'unnamedplus'
vim.opt.expandtab = true
vim.opt.shiftwidth = 2
vim.opt.tabstop = 2

-- Insert-mode cursor as a blinking block. Default (`ver25`, a thin
-- vertical bar) gets lost against the text in the draft pane,
-- especially after a quote-paste flash clears. Block + blink makes the
-- caret position obvious. Normal-mode stays block (already obvious as
-- the inverted cell).
vim.opt.guicursor = 'n-v-c-sm:block,i-ci-ve:block-blinkon250-blinkoff250,r-cr-o:hor20'

-- When the draft loses focus while in insert mode, animate an `_`
-- (U+005F, LOW LINE) over the cursor cell. zellij draws an unfocused
-- pane's cursor as a hollow outline regardless of DECSCUSR, so the
-- only way to make the cursor visibly "alive" is to paint underneath
-- it. We toggle an extmark between two virt_text contents (`_` on
-- the visible tick, the underlying buffer character on the hidden
-- tick); the buffer's actual bytes are untouched (no undo entry, no
-- modification flag, no autosave trigger) except for one transient
-- trailing space when the cursor sits past EOL — see the pad
-- helpers below.
--
-- Glyph history: tried `█` (too loud — read as a regular cursor),
-- `🖱` (emoji presentation forced two cells, overflowed zellij's
-- 1-cell ghost rectangle), `⏺` (clearer signature than `█` but felt
-- distracting against the lightly-faded unfocused-insert sheet).
-- `_` lands as a quiet typewriter-style underline inside zellij's
-- hollow box: visible at a glance, not attention-grabbing.
--
-- Why this matters: the copy-on-select pipeline
-- (bin/copy-on-select + bin/clipboard-to-pane) lands a
-- selection from any pane into the draft while the draft is
-- unfocused. The user wants to see "this pane is the target" at a
-- glance.
--
-- The block blinks by toggling the extmark every BLINK_MS. Defaults
-- to 500 ms on / 500 ms off so the eye picks up motion without being
-- distracting. CursorMoved* refreshes the position so a paste-driven
-- cursor advance follows the block forward without waiting for the
-- next tick.
local pair_focus_block_ns = vim.api.nvim_create_namespace('pair_focus_block')
local pair_focus_buf = nil
local pair_focus_timer_id = nil
local pair_focus_visible = false
local PAIR_BLOCK_BLINK_MS = 500

-- Default to linking PairFocusBlock to Cursor; user themes can override.
vim.api.nvim_set_hl(0, 'PairFocusBlock', { link = 'Cursor', default = true })

-- Past-EOL bookkeeping. virt_text_pos='inline' past EOL turned out to
-- have a redraw race under zellij that stalled the blink after one
-- cycle. Instead, when the cursor sits past EOL on FocusLost we
-- temporarily append a real space to the line so the cursor is
-- *in-line*; everything else can use the well-trodden overlay path.
-- The space is removed on FocusGained. Two careful bits:
--   - autosave fires from another FocusLost autocmd that runs after
--     ours. To keep the persisted draft clean, the space-insert is
--     scheduled (vim.schedule) so autosave runs first on a clean
--     buffer; our pad lands a tick later.
--   - `modified` is restored around both insert and remove so the
--     dirty flag doesn't flicker.
local pair_focus_eol_padded = false
local pair_focus_eol_row    = nil

local function pair_eol_ensure_pad()
  if pair_focus_eol_padded then return end
  if not pair_focus_buf or not vim.api.nvim_buf_is_loaded(pair_focus_buf) then return end
  local win = vim.fn.bufwinid(pair_focus_buf)
  if win == -1 then return end
  local row, col = unpack(vim.api.nvim_win_get_cursor(win))
  local line = vim.api.nvim_buf_get_lines(pair_focus_buf, row - 1, row, false)[1] or ''
  if col < #line then return end -- already in-line
  local was_modified = vim.bo[pair_focus_buf].modified
  vim.api.nvim_buf_set_text(pair_focus_buf, row - 1, #line, row - 1, #line, { ' ' })
  vim.bo[pair_focus_buf].modified = was_modified
  pair_focus_eol_padded = true
  pair_focus_eol_row    = row
end

local function pair_eol_remove_pad()
  if not pair_focus_eol_padded then return end
  local row = pair_focus_eol_row
  pair_focus_eol_padded = false
  pair_focus_eol_row    = nil
  if not row or not pair_focus_buf or not vim.api.nvim_buf_is_loaded(pair_focus_buf) then
    return
  end
  if row > vim.api.nvim_buf_line_count(pair_focus_buf) then return end
  local line = vim.api.nvim_buf_get_lines(pair_focus_buf, row - 1, row, false)[1] or ''
  if line:sub(-1) == ' ' then
    local was_modified = vim.bo[pair_focus_buf].modified
    vim.api.nvim_buf_set_text(pair_focus_buf, row - 1, #line - 1, row - 1, #line, {})
    vim.bo[pair_focus_buf].modified = was_modified
  end
end

-- Render the cursor cell with an overlay extmark. Always toggle between
-- ⏺ and the underlying character so zellij's unfocused-pane blank-cell
-- paint never reaches the cell — we always have content there.
local function pair_block_render(show)
  if not pair_focus_buf or not vim.api.nvim_buf_is_loaded(pair_focus_buf) then return end
  local win = vim.fn.bufwinid(pair_focus_buf)
  if win == -1 then return end
  local row, col = unpack(vim.api.nvim_win_get_cursor(win))
  local line = vim.api.nvim_buf_get_lines(pair_focus_buf, row - 1, row, false)[1] or ''
  vim.api.nvim_buf_clear_namespace(pair_focus_buf, pair_focus_block_ns, 0, -1)
  -- If cursor moved past EOL during the unfocused window (e.g. paste
  -- jumped it there), top up the pad so we stay in-line.
  if col >= #line then
    pair_eol_ensure_pad()
    line = vim.api.nvim_buf_get_lines(pair_focus_buf, row - 1, row, false)[1] or ''
  end
  if col >= #line then return end -- defensive: pad failed somehow

  local glyph, hl
  if show then
    glyph, hl = '_', 'PairFocusBlock'
  else
    -- strcharpart on the byte-substring from col gives us the first
    -- grapheme cleanly, handling multi-byte UTF-8 (emoji, CJK, etc.).
    glyph = vim.fn.strcharpart(line:sub(col + 1), 0, 1)
    if glyph == '' or glyph == '\t' then glyph = ' ' end
    hl = 'Normal'
  end

  vim.api.nvim_buf_set_extmark(pair_focus_buf, pair_focus_block_ns, row - 1, col, {
    virt_text     = { { glyph, hl } },
    virt_text_pos = 'overlay',
    priority      = 200,
  })
end

local function pair_block_clear()
  if pair_focus_buf and vim.api.nvim_buf_is_loaded(pair_focus_buf) then
    vim.api.nvim_buf_clear_namespace(pair_focus_buf, pair_focus_block_ns, 0, -1)
  end
end

local function pair_block_tick()
  pair_focus_visible = not pair_focus_visible
  pcall(pair_block_render, pair_focus_visible)
end

local function pair_block_start()
  if pair_focus_timer_id then return end
  pair_focus_buf = vim.api.nvim_get_current_buf()
  -- Defer the actual work past the synchronous FocusLost autocmd chain
  -- so autosave (registered later, runs after us) sees a clean buffer.
  -- Our pad lands a tick afterwards.
  vim.schedule(function()
    if not pair_focus_buf then return end -- FocusGained may have raced
    pair_eol_ensure_pad()
    pair_focus_visible = true
    pair_block_render(true)
    pair_focus_timer_id = vim.fn.timer_start(PAIR_BLOCK_BLINK_MS, pair_block_tick,
      { ['repeat'] = -1 })
  end)
end

local function pair_block_stop()
  if pair_focus_timer_id then
    vim.fn.timer_stop(pair_focus_timer_id)
    pair_focus_timer_id = nil
  end
  pair_block_clear()
  pair_eol_remove_pad()
  pair_focus_buf = nil
  pair_focus_visible = false
end

vim.api.nvim_create_autocmd('FocusLost', {
  callback = function()
    local m = vim.fn.mode()
    if m == 'i' or m == 'ic' or m == 'ix' then
      pair_block_start()
    end
  end,
})

vim.api.nvim_create_autocmd('FocusGained', {
  callback = pair_block_stop,
})

-- Track cursor movement during the unfocused window — paste-driven
-- inserts advance the cursor and we want the block to follow without
-- waiting for the next blink.
vim.api.nvim_create_autocmd({ 'CursorMoved', 'CursorMovedI', 'TextChanged', 'TextChangedI' }, {
  callback = function()
    if pair_focus_timer_id then
      pcall(pair_block_render, pair_focus_visible)
    end
  end,
})

-- Disable nvim's right-click context menu. Default `mousemodel=popup_setpos`
-- pops up a "Copy/Paste/..." menu that's confusing inside the pair draft pane;
-- `extend` falls back to the vim-traditional behavior of extending the visual
-- selection on right-click.
vim.opt.mousemodel = 'extend'

-- Completion popup behavior. menuone: show popup even with one match.
-- noinsert,noselect: never auto-insert or auto-highlight a match — Enter
-- in the draft is overwhelmingly "newline", so accidental confirmation
-- would be disruptive. The user explicitly cycles with Tab/Shift-Tab.
vim.opt.completeopt = { 'menu', 'menuone', 'noinsert', 'noselect' }

-- Persistent undo so cleared content is recoverable
local undodir = vim.fn.expand('~/.local/share/pair/undo')
vim.fn.mkdir(undodir, 'p')
vim.opt.undodir = undodir
vim.opt.undofile = true

-- ---------------------------------------------------------------------------
-- helpers
-- ---------------------------------------------------------------------------

local function pair_tag()
  return os.getenv('PAIR_TAG') or os.getenv('PAIR_AGENT') or 'claude'
end

local function pair_data_dir()
  return os.getenv('PAIR_DATA_DIR')
      or (os.getenv('XDG_DATA_HOME') or vim.fn.expand('~/.local/share'))
         .. '/pair'
end

local function log_path_for_tag()
  return pair_data_dir() .. '/log-' .. pair_tag() .. '.md'
end

local function draft_path_for_tag()
  return pair_data_dir() .. '/draft-' .. pair_tag() .. '.md'
end

local function read_file(path)
  local f = io.open(path, 'r')
  if not f then return '' end
  local content = f:read('*a') or ''
  f:close()
  return content
end

local function write_file(path, content)
  vim.fn.mkdir(vim.fn.fnamemodify(path, ':h'), 'p')
  local f = io.open(path, 'w')
  if not f then return false end
  f:write(content)
  f:close()
  return true
end

local function append_log(body)
  local path = log_path_for_tag()
  vim.fn.mkdir(vim.fn.fnamemodify(path, ':h'), 'p')
  local f = io.open(path, 'a')
  if not f then return end
  f:write(os.date('## %Y-%m-%d %H:%M:%S') .. '\n\n')
  f:write(body)
  f:write('\n\n---\n\n')
  f:close()
end

-- Parse log-<tag>.md into a list of entry bodies, oldest first.
-- Entry shape (per append_log): "## YYYY-MM-DD HH:MM:SS\n\n<body>\n\n---\n\n".
-- Splitting on the entry separator yields parts; the trailing chunk is "" since
-- the file ends with the separator. Each non-empty part starts with the
-- timestamp header which we strip to recover just the body.
local function parse_log(text)
  local entries = {}
  if text == '' then return entries end
  local parts = vim.split(text, '\n\n---\n\n', { plain = true })
  for _, part in ipairs(parts) do
    if part ~= '' then
      local body = part:gsub('^## %S+ %S+\n\n', '', 1)
      table.insert(entries, body)
    end
  end
  return entries
end

local function read_history()
  return parse_log(read_file(log_path_for_tag()))
end

-- Rewrite the body of the n-th-most-recent log entry (n=1 is newest) in
-- place, preserving its timestamp header. Used to persist comment-only
-- edits made while navigating history — comments are stripped before the
-- agent sees them, so changing them is a no-op against the agent's view of
-- history. No-op if n is out of range or the file is missing/malformed.
local function write_history_entry(n, body)
  local path = log_path_for_tag()
  local text = read_file(path)
  if text == '' then return end
  local parts = vim.split(text, '\n\n---\n\n', { plain = true })
  local entries = {}
  for _, p in ipairs(parts) do
    if p ~= '' then table.insert(entries, p) end
  end
  local idx = #entries - n + 1
  if idx < 1 or idx > #entries then return end
  local header = entries[idx]:match('^(## %S+ %S+\n\n)')
  if not header then return end
  entries[idx] = header .. body
  write_file(path, table.concat(entries, '\n\n---\n\n') .. '\n\n---\n\n')
end

local function buffer_text()
  return table.concat(vim.api.nvim_buf_get_lines(0, 0, -1, false), '\n')
end

local function set_buffer_text(s)
  -- Replace whole buffer contents, mark unmodified, park cursor at end so the
  -- user can append immediately. nvim's autosave autocmd is gated on nav.pos,
  -- so this won't clobber draft-<tag>.md when we're showing a history entry.
  --
  -- Strip a single trailing "\n" from `s` before splitting: files we read off
  -- disk have nvim's :w-added trailing newline, but the user-visible buffer
  -- representation (and what buffer_text() returns) doesn't include it. Without
  -- this, set_buffer_text(read_file(draft)) produces a buffer with a spurious
  -- empty trailing line that grows with every write/read cycle.
  s = s or ''
  if s:sub(-1) == '\n' then s = s:sub(1, -2) end
  local lines = vim.split(s, '\n', { plain = true })
  vim.api.nvim_buf_set_lines(0, 0, -1, false, lines)
  vim.bo.modified = false
  local last = math.max(1, #lines)
  pcall(vim.api.nvim_win_set_cursor, 0, { last, #(lines[last] or '') })
end

-- Forward-declared navigation state. Accessed by send_and_clear and the
-- autosave autocmd (both before the nav helpers below). Initialized to the
-- "draft slot, clean buffer" state.
local nav = { pos = '*', baseline = '' }

-- Forward-declared layout-state mirror. layout_read (declared much later)
-- derives the rung from vim.o.lines; this in-memory copy lets earlier
-- callers (e.g. pair_ensure_visible_then) check the rung without a
-- forward-declaration shuffle. Updated by layout_write() at every
-- PairLayoutBigger/Smaller transition.
local pair_layout_state = 'small'

local function refresh_statusline()
  -- Defer to the next event-loop tick so the redraw fires *after* any side
  -- effects from the calling action have settled — e.g. send_and_clear's
  -- vim.fn.system shell-outs to zellij and the :w that follows. Without
  -- this, the trailing redraw work from those operations can blank the
  -- statusline immediately after our refresh, and it stays blank until
  -- the next user action triggers another redraw.
  vim.schedule(function() pcall(vim.cmd, 'redrawstatus') end)
end

-- (Removed: focus-loss spinner + auto-focus-grab. nvim used to yank focus
-- back to the draft pane 10s after losing it, with a braille countdown in
-- the statusline. The forced focus return was distracting mid-task in the
-- agent pane, so the whole feature is gone — FocusLost/FocusGained now only
-- manage the focus cursor.)

-- Brief inverted flash on the "N queued" statusline segment. Confirms a
-- queue-count change actually happened — Alt+q lands an item, send-from-+N
-- consumes one, Alt+BS deletes one. Without it the only visible change is
-- "buffer snapped/cleared," which is the same shape as a discard.
--
-- Both PairQueueCount and PairQueueZero are swapped so the flash is visible
-- whether the queue ends non-empty (PairQueueCount) or hits zero on this
-- transition (PairQueueZero). Uses IncSearch + 500ms to match paste-flash.
local function flash_queue_count()
  vim.api.nvim_set_hl(0, 'PairQueueCount', { link = 'IncSearch' })
  vim.api.nvim_set_hl(0, 'PairQueueZero',  { link = 'IncSearch' })
  refresh_statusline()
  vim.defer_fn(function()
    vim.api.nvim_set_hl(0, 'PairQueueCount', { link = 'WarningMsg' })
    vim.api.nvim_set_hl(0, 'PairQueueZero',  { link = 'Comment' })
    refresh_statusline()
  end, 500)
end

-- ---------------------------------------------------------------------------
-- queue store (issue #000015 §H2)
--
-- Per-tag directory `$DATA_DIR/queue-<tag>/` with one file per queued prompt.
-- Filenames are 6-digit zero-padded keys; sort order = display order (+1 is
-- the lowest key, +M is the highest). Keys start at 500000 in the middle of
-- the address space so push_front and push_back can grow either way without
-- collision in practical use.
-- ---------------------------------------------------------------------------

local QUEUE_KEY_FMT = '%06d'
local QUEUE_KEY_START = 500000

local function queue_dir()
  return pair_data_dir() .. '/queue-' .. pair_tag()
end

local function queue_path(key)
  return queue_dir() .. '/' .. key .. '.md'
end

local function queue_keys_sorted()
  vim.fn.mkdir(queue_dir(), 'p')
  local files = vim.fn.readdir(queue_dir(), function(name)
    return (name:match('^%d+%.md$') and 1 or 0)
  end)
  table.sort(files)
  local keys = {}
  for _, f in ipairs(files) do
    table.insert(keys, (f:gsub('%.md$', '')))
  end
  return keys
end

local function queue_count()
  return #queue_keys_sorted()
end

local function queue_read(key)
  return read_file(queue_path(key))
end

local function queue_write(key, body)
  return write_file(queue_path(key), body)
end

local function queue_remove(key)
  return os.remove(queue_path(key))
end

local function queue_push_back(body)
  local keys = queue_keys_sorted()
  local next_n = (#keys == 0) and QUEUE_KEY_START or (tonumber(keys[#keys]) + 1)
  local key = string.format(QUEUE_KEY_FMT, next_n)
  queue_write(key, body)
  return key
end

local function queue_push_front(body)
  local keys = queue_keys_sorted()
  local next_n = (#keys == 0) and QUEUE_KEY_START or (tonumber(keys[1]) - 1)
  local key = string.format(QUEUE_KEY_FMT, next_n)
  queue_write(key, body)
  return key
end

-- Display index N (1-based, +1 = front) → filename key.
local function queue_key_for_n(n)
  local keys = queue_keys_sorted()
  return keys[n]
end

-- Check if Neovim has a UI attached (false indicates headless mode, e.g. tests)
local function has_ui()
  return #vim.api.nvim_list_uis() > 0
end

local function send_esc_to_agent()
  -- ESC = 0x1b = 27. Claude reads this as "interrupt current stream".
  if not has_ui() then return end
  PairZellijTrace.action('draft.interrupt.focus-agent', { 'zellij', 'action', 'move-focus', 'up' })
  PairZellijTrace.action('draft.interrupt.esc', { 'zellij', 'action', 'write', '27' })
  PairZellijTrace.action('draft.interrupt.focus-draft', { 'zellij', 'action', 'move-focus', 'down' })
end

local function draftSendCommands(body, no_submit)
  local cmds = {
    { label = 'draft.send.focus-agent', argv = { 'zellij', 'action', 'move-focus', 'up' } },
    {
      label = 'draft.send.write-body',
      argv = { 'zellij', 'action', 'write-chars', body },
      opts = { redact = { [4] = body } },
    },
  }
  if no_submit then
    cmds[#cmds + 1] = { label = 'draft.send.newline', argv = { 'zellij', 'action', 'write', '13' } }
  else
    cmds[#cmds + 1] = { label = 'draft.send.submit', argv = { 'zellij', 'action', 'send-keys', 'Alt Enter' } }
  end
  cmds[#cmds + 1] = { label = 'draft.send.focus-draft', argv = { 'zellij', 'action', 'move-focus', 'down' } }
  return cmds
end

_G.PairDraftSendCommands = draftSendCommands

local function send_to_agent(body, no_submit)
  -- focus up to agent pane, type body, press Enter, focus back down.
  --
  -- We deliberately do NOT clear the agent's input first. The "[Image #N]"
  -- tokens that appear there after Alt+i Ctrl+V aren't visual echoes —
  -- they're the chip representation of the attached image bytes, and a
  -- readline kill (Ctrl+U / Cmd+Delete) detaches the underlying image
  -- along with the chip text. So we live with the chips: claude treats
  -- the chip in its input as the attachment, and the "[Image #N]" string
  -- in our body as a reference to it. Complementary, not duplicate.
  --
  -- Multi-line / large bodies get wrapped by zellij as bracketed paste
  -- (`\e[200~...\e[201~`). zellij's write-chars returns once the bytes
  -- are queued, not delivered — sending the submit immediately after
  -- can land inside the paste boundary and get treated as a literal
  -- newline rather than submit. Settle for ~100ms in that case so the
  -- agent has time to ingest the paste and return to the input prompt
  -- before we hit submit. Single-line sends skip the wait.
  --
  -- Submit is Alt+Enter, not plain Enter:
  -- pair-wrap's stdin translator rewrites incoming \r into the agent's
  -- "insert newline" sequence (claude: `\<Enter>`, codex/agy: \n),
  -- so a bare CR here would insert a newline rather than submit. Use
  -- zellij's semantic send-keys action for the modified chord instead
  -- of synthesizing it as raw ESC+CR bytes.
  --
  -- no_submit (Alt+Shift+Enter path): land the body in the agent's
  -- composer followed by a literal newline but DON'T submit. A bare CR
  -- (write 13) is exactly what pair-wrap rewrites into the agent's
  -- insert-newline sequence — the same byte the comment above warns is
  -- *not* a submit — so it leaves the cursor on a fresh line in the
  -- composer, ready for more input.
  if not has_ui() then return end
  local cmds = draftSendCommands(body, no_submit)
  PairZellijTrace.action(cmds[1].label, cmds[1].argv, cmds[1].opts)
  PairZellijTrace.action(cmds[2].label, cmds[2].argv, cmds[2].opts)
  if body:find('\n') or #body > 200 then
    vim.cmd('sleep 100m')
  end
  for i = 3, #cmds do
    PairZellijTrace.action(cmds[i].label, cmds[i].argv, cmds[i].opts)
  end
end

-- :PairReview <file> — PROPOSE a file for review (#66 M4a'). It does NOT open the
-- pane: it writes the review-target seam (status=proposed), runs the deterministic
-- readiness prep locally (track / new branch / resume), then sends the agent a
-- minimal ack request. Alt+c opens the pane once ready. `complete='file'` gives
-- :e-style tab-completion; Alt+c feeds ":PairReview " into this command line.
-- (Callback runs at runtime, after the do-block below sets `_G._pair_review`.)
vim.api.nvim_create_user_command('PairReview', function(opts)
  if _G._pair_review and _G._pair_review.propose then
    local ok = _G._pair_review.propose(opts.args)
    local status = ok and 'prepared ' or 'could not prepare '
    vim.notify('PairReview: ' .. status .. vim.fn.fnamemodify(opts.args, ':t')
      .. ' — press Alt+c to open when ready', ok and vim.log.levels.INFO or vim.log.levels.WARN)
  end
end, { nargs = 1, complete = 'file', desc = 'prepare a file for review (Alt+c opens when ready)' })

-- Review-workbench draft-side helpers (#66 M3). Wrapped in `do ... end` and shared
-- via the `_G._pair_review` table rather than added as file-level `local`s: init.lua's
-- main chunk is near Lua's hard 200-local-per-function ceiling (E5112), so new
-- top-level locals would break sourcing. The indicator block (below) reuses
-- `_pair_review.is_alive` from here, keeping the state-file contract DRY.
do
  local review_init_epoch = os.time()

  -- review-state file (nvim/review.lua writes it on VimEnter, removes it on exit).
  -- Line 1 = the review pane nvim's pid (liveness = `kill -0`); line 2 = the
  -- absolute doc path (the review-mode indicator reads it). Path from the shared
  -- seam module so this reader and the pane writer can't diverge (ARCH-DRY, I3).
  local nvim_dir = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
  local seam = dofile(nvim_dir .. 'review/seam.lua')

  -- The review-target (seam #6) is CONVERSATION-scoped: it carries the
  -- PAIR_SESSION_ID it was written under, and a reader IGNORES a target from a
  -- different session. So a fresh pair session (new PAIR_SESSION_ID) prompts
  -- (Alt+c → :PairReview), while an Alt+n restart that RESUMES the conversation
  -- (same id, re-loaded init) keeps its in-progress review. We do NOT clear on
  -- load — that would wrongly reset a resumed review. Pure → unit-testable. (#66 #6.)
  local function target_stale(t, session)
    return not (type(t) == 'table' and session and session ~= '' and t.session == session)
  end

  local function config_session_id(data_dir, tag, agent)
    local cf = io.open(data_dir .. '/config-' .. tag .. '-' .. agent .. '.json', 'r')
    if not cf then return nil end
    local body = cf:read('*a'); cf:close()
    local ok, parsed = pcall(vim.json.decode, body)
    if ok and type(parsed) == 'table' and parsed.session_id and parsed.session_id ~= '' then
      return parsed.session_id
    end
    return nil
  end

  local function descendant_pids(root)
    local out = vim.fn.systemlist({ 'ps', '-axo', 'pid=,ppid=' })
    local children = {}
    for _, line in ipairs(out) do
      local pid, ppid = line:match('^%s*(%d+)%s+(%d+)%s*$')
      if pid and ppid then
        children[ppid] = children[ppid] or {}
        table.insert(children[ppid], pid)
      end
    end
    local pids, queue, seen = {}, { root }, { [root] = true }
    local i = 1
    while i <= #queue do
      local pid = queue[i]; i = i + 1
      table.insert(pids, pid)
      for _, child in ipairs(children[pid] or {}) do
        if not seen[child] then
          seen[child] = true
          table.insert(queue, child)
        end
      end
    end
    return pids
  end

  local function live_codex_session_id(data_dir, tag)
    local pf = io.open(data_dir .. '/agent-pid-' .. tag, 'r')
    if not pf then return nil end
    local root = vim.trim(pf:read('*a') or ''); pf:close()
    if root == '' then return nil end
    for _, pid in ipairs(descendant_pids(root)) do
      for _, line in ipairs(vim.fn.systemlist({ 'lsof', '-p', pid, '-Fn' })) do
        local path = line:match('^n(.*/%.codex/sessions/.*/rollout%-.*%.jsonl)$')
        if path then
          local sid = path:match('([0-9a-fA-F]+%-[0-9a-fA-F]+%-[0-9a-fA-F]+%-[0-9a-fA-F]+%-[0-9a-fA-F]+)%.jsonl$')
          if sid then return sid end
        end
      end
    end
    return nil
  end

  local function current_session_id()
    local sid = vim.env.PAIR_SESSION_ID
    if sid and sid ~= '' then return sid end
    local data_dir = vim.env.PAIR_DATA_DIR
    if not data_dir or data_dir == '' then return nil end
    local tag = (vim.env.PAIR_TAG and vim.env.PAIR_TAG ~= '') and vim.env.PAIR_TAG or 'default'
    local agent = (vim.env.PAIR_AGENT and vim.env.PAIR_AGENT ~= '') and vim.env.PAIR_AGENT or 'claude'
    sid = config_session_id(data_dir, tag, agent)
    if sid then return sid end
    if agent == 'codex' then return live_codex_session_id(data_dir, tag) end
    return nil
  end

  local function state_file()
    return seam.open_state(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
  end

  -- Returns (alive, statefile, file). `file` is the reviewed doc's absolute path.
  local function is_alive()
    local sf = state_file()
    if not sf or vim.fn.filereadable(sf) ~= 1 then return false, sf end
    local lines = vim.fn.readfile(sf)
    local pid = tonumber(lines[1] or '')
    if not pid then return false, sf end
    vim.fn.system({ 'kill', '-0', tostring(pid) })
    return vim.v.shell_error == 0, sf, lines[2]
  end

  -- review-target (seam #6): {file, status: proposed|ready}. :PairReview proposes;
  -- the agent marks ready after prep; Alt+c reads it to decide prompt/open/wait.
  local function read_target()
    local p = seam.target_path(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
    if not p or vim.fn.filereadable(p) ~= 1 then return nil end
    local ok, t = pcall(vim.json.decode, table.concat(vim.fn.readfile(p), '\n'))
    if not (ok and type(t) == 'table' and t.file and t.file ~= '') then return nil end
    -- ignore a target from a different conversation (stale across a fresh session)
    if target_stale(t, current_session_id()) then
      -- Codex fresh starts learn their session id asynchronously. If :PairReview
      -- prepared a target before any id was discoverable, keep that same-nvim
      -- unscoped target readable; old unscoped targets remain stale.
      local stat = (vim.uv or vim.loop).fs_stat(p)
      local mtime = stat and stat.mtime and stat.mtime.sec or 0
      if not (t.session == '' and mtime >= review_init_epoch) then return nil end
    end
    return t
  end
  local function write_target(file, status)
    local p = seam.target_path(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
    if p then pcall(vim.fn.writefile,
      { vim.json.encode({ file = file, status = status, session = current_session_id() or '' }) }, p) end
  end
  -- :PairReview proposes a target, then performs deterministic local prep via the
  -- readiness shell seam. The nvim still does NOT open the pane — Alt+c does, once
  -- the prep marks the target ready. The agent gets only a concise ack request.
  local function propose(file)
    local abs = vim.fn.fnamemodify(file, ':p')
    write_target(abs, 'proposed')
    local home = vim.env.PAIR_HOME or ''
    -- PAIR_REVIEW_READINESS_BIN (test seam) is a binary implementing the readiness
    -- CLI directly (`--prepare <abs>`); the default self-execs the single pair as
    -- `pair review readiness --prepare <abs>` (#104 M2).
    local override = vim.env.PAIR_REVIEW_READINESS_BIN
    local cmd
    if override and override ~= '' then
      cmd = { override, '--prepare', abs }
    else
      local pair = (home ~= '') and (home .. '/bin/pair') or 'pair'
      cmd = { pair, 'review', 'readiness', '--prepare', abs }
    end
    local out = table.concat(vim.fn.systemlist(cmd), '\n')
    local ok = vim.v.shell_error == 0
    if out ~= '' then send_to_agent(out) end
    if not ok then
      vim.notify('PairReview: prep failed — ' .. out, vim.log.levels.WARN)
    end
    return ok, out
  end

  -- Pure: the Alt+c decision. A live pane → flip visibility (hide/show). No live
  -- pane → driven by the review-target status: ready→open, proposed→wait (prep in
  -- progress), none→prompt (:PairReview file-select). Exposed for the headless test.
  local function toggle_action(alive, visible, target_status)
    if alive then return visible and 'hide' or 'show' end
    if target_status == 'ready' then return 'open' end
    if target_status == 'proposed' then return 'wait' end
    return 'prompt'
  end

  _G._pair_review = { state_file = state_file, is_alive = is_alive, toggle_action = toggle_action,
    read_target = read_target, write_target = write_target, propose = propose, target_stale = target_stale,
    current_session_id = current_session_id }
  _G._pair_review_toggle_action = toggle_action -- test alias

  -- Alt+c — the collaboration/review-workbench brain (#66 M3/M4b). Routed here through the draft
  -- nvim (Alt+d-style) so the branch happens in a real nvim, not a transient shell
  -- pane. A LIVE pane → flip visibility (`are-floating-panes-visible` →
  -- show/hide-floating-panes; never the toggle-floating-panes footgun). Otherwise
  -- branch on the review target (seam #6): ready→open via `pair review open`,
  -- proposed→"prep in progress", none→drop into `:PairReview ` (file-select). The
  -- review pane defines its own PairReviewToggle() (hide-self) for Alt+c from inside
  -- the focused floating pane. No has_ui() guard so the headless test records calls.
  function _G.PairReviewToggle()
    local alive, sf = is_alive()
    if alive then
      local vis = vim.fn.system({ 'zellij', 'action', 'are-floating-panes-visible' })
      if toggle_action(true, vis:match('true') ~= nil) == 'hide' then
        vim.fn.system({ 'zellij', 'action', 'hide-floating-panes' })
      else
        vim.fn.system({ 'zellij', 'action', 'show-floating-panes' })
      end
      return
    end
    local t = read_target()
    local action = toggle_action(false, false, t and t.status)
    if action == 'open' then
      local home = vim.env.PAIR_HOME or ''
      local pair = (home ~= '') and (home .. '/bin/pair') or 'pair'
      vim.fn.system({ pair, 'review', 'open', t.file })
      if vim.v.shell_error ~= 0 then
        vim.notify('PairReview: open failed — ' .. vim.fn.fnamemodify(t.file, ':t'), vim.log.levels.WARN)
      end
    elseif action == 'wait' then
      vim.notify('review prep in progress — check the agent pane', vim.log.levels.INFO)
    else -- 'prompt'
      if sf then pcall(os.remove, sf) end -- reap a stale/dead open-state file
      vim.api.nvim_feedkeys(':PairReview ', 'n', false)
    end
  end
end

-- Strip whole-line comments (^%s*===) before sending. Comments are stored
-- intact in draft/queue/log so they survive history navigation — only what
-- reaches the agent is cleaned. Leading and trailing blank lines left behind
-- by the strip are also dropped so the agent doesn't see a dangling head or
-- tail. (Leading matters because comment_lines preserves blanks between
-- sticky comments, which sit at the top of the next draft.)
local function strip_comments(body)
  local out = {}
  for line in (body .. '\n'):gmatch('([^\n]*)\n') do
    if not line:match('^%s*===') then
      table.insert(out, line)
    end
  end
  while #out > 0 and out[#out]:match('^%s*$') do
    table.remove(out)
  end
  while #out > 0 and out[1]:match('^%s*$') do
    table.remove(out, 1)
  end
  return table.concat(out, '\n')
end

-- Inverse of strip_comments: returns the comment lines (in order), preserving
-- blank lines that sit *between* comments so the sticky block keeps its
-- spacing across a send. Blank lines before the first comment or after the
-- last one are dropped — only interior spacing is structural. Used to extract
-- sticky === context from any sent body — see apply_sticky_to_star.
local function comment_lines(body)
  local out = {}
  local pending = {}
  local seen = false
  for line in (body .. '\n'):gmatch('([^\n]*)\n') do
    if line:match('^%s*===') then
      for _, b in ipairs(pending) do table.insert(out, b) end
      pending = {}
      table.insert(out, line)
      seen = true
    elseif seen and line:match('^%s*$') then
      table.insert(pending, line)
    else
      pending = {}
    end
  end
  return out
end

-- After any send, the just-sent body's === lines become *'s new sticky set.
-- The non-comment WIP portion of * is preserved (only its old comments are
-- replaced) so a send from -N or +N doesn't clobber a half-typed draft at *.
-- When the send originated from * itself, there is no separate WIP to keep
-- — the sent body was the WIP — so the result is just the stickies plus a
-- typing line. Caller is responsible for the buffer view AND for writing the
-- result to disk via `:w` — going through vim keeps b_mtime_read in sync, so
-- the next autosave doesn't trip the "file changed since reading it" prompt.
local function apply_sticky_to_star(sent_body, was_at_star)
  local star_path = draft_path_for_tag()
  local star_body = was_at_star and '' or (read_file(star_path) or '')
  local stickies   = comment_lines(sent_body)
  local non_comm   = strip_comments(star_body)

  local lines = {}
  for _, l in ipairs(stickies) do table.insert(lines, l) end
  if non_comm ~= '' then
    if #lines > 0 then table.insert(lines, '') end
    for line in (non_comm .. '\n'):gmatch('([^\n]*)\n') do
      table.insert(lines, line)
    end
  end
  -- Trailing blank so the cursor lands on a fresh row to type into.
  if #lines == 0 or lines[#lines] ~= '' then table.insert(lines, '') end

  return lines
end

-- ---------------------------------------------------------------------------
-- attach_image: Alt+i — capture-driven image attachment.
--
-- Two-phase, non-blocking. The slow part (waiting for the agent to render
-- its image marker) runs asynchronously so the user can keep typing.
--
-- Phase 1 (synchronous, <10ms):
--   1. Sanity-check the OS clipboard actually holds an image; if not, flash
--      "[no image in clipboard]" at cursor for 1s and bail.
--   2. Read pair-wrap's pid from $DATA_DIR/pair-wrap-pid-<tag>. Bail with a
--      restart hint if missing or dead — pair-wrap is the whole I/O path
--      for the agent pane, so the session needs to be restarted anyway.
--   3. SIGUSR1 pair-wrap to arm a capture window, then write Ctrl+V to the
--      agent pane via zellij. The agent renders its own marker into its
--      input area; pair-wrap tees the bytes into image-capture-<tag>.
--   4. Insert a `[Image]` placeholder at cursor, anchor it with an extmark,
--      advance cursor past it. User can keep typing immediately.
--
-- Phase 2 (async, up to 1s):
--   5. Poll image-capture-<tag>.done every 50ms via vim.defer_fn.
--   6. On hit: read the buffer, strip ANSI, regex-match the agent's marker,
--      replace the placeholder span (resolved through the extmark, so any
--      typing the user has done in the meantime is preserved correctly).
--   7. On timeout: leave the placeholder in place, notify the user.
--
-- The agent is the source of truth for the marker text — no local counter,
-- no per-agent format hardcoded. Claude renders `[Image #N]` (sequential),
-- agy renders `[Image 12981-2312]` (non-consecutive); the same regex
-- matches both.
-- ---------------------------------------------------------------------------

local PAIR_IMAGE_MARKER_RE = '%[Image[ #][^%]]+%]'

-- Normalize a captured marker back to its canonical, human-readable form.
-- Claude's TUI paints `[Image #N]` (with a space) but routinely emits a
-- cursor-positioning CSI between "Image" and "#" instead of an actual
-- space character. strip_ansi correctly removes the CSI, which collapses
-- the visual space and yields "[Image#N]" — same bytes as far as the
-- regex is concerned but the wrong shape when typed back into the agent
-- (claude expects the spaced form). Re-insert the space.
local function normalize_image_marker(marker)
  -- Idempotent: only matches the unspaced form.
  return (marker:gsub('^(%[Image)(#)', '%1 %2'))
end

local function clipboard_has_image()
  -- macOS: clipboard info enumerates the available types as "«class XXXX»"
  -- tokens; image data shows up under PNGf / TIFF / JPEG / GIFf / jp2.
  if vim.fn.has('mac') == 1 then
    local out = vim.fn.system({
      'osascript', '-e',
      'try\n  return (clipboard info) as string\nend try',
    })
    return out:find('PNGf') ~= nil
        or out:find('TIFF') ~= nil
        or out:find('JPEG') ~= nil
        or out:find('GIFf') ~= nil
        or out:find('jp2 ') ~= nil
  end
  -- Linux: prefer Wayland (wl-paste --list-types), fall back to X11 (xclip).
  if vim.fn.executable('wl-paste') == 1 then
    local out = vim.fn.system({ 'wl-paste', '--list-types' })
    return out:find('image/') ~= nil
  end
  if vim.fn.executable('xclip') == 1 then
    local out = vim.fn.system({ 'xclip', '-selection', 'clipboard', '-t', 'TARGETS', '-o' })
    return out:find('image/') ~= nil
  end
  -- Unknown platform: assume there's an image and let the agent decide.
  -- The Ctrl+V will be a no-op for the agent; capture will time out;
  -- user gets the "no marker detected" path instead of an early flash.
  return true
end

local function flash_at_cursor(text, hl_group, duration_ms)
  local bufnr = vim.api.nvim_get_current_buf()
  local ns = vim.api.nvim_create_namespace('pair_image_flash')
  local row, col = unpack(vim.api.nvim_win_get_cursor(0))
  local id = vim.api.nvim_buf_set_extmark(bufnr, ns, row - 1, col, {
    virt_text = { { text, hl_group } },
    virt_text_pos = 'inline',
  })
  vim.defer_fn(function()
    pcall(vim.api.nvim_buf_del_extmark, bufnr, ns, id)
  end, duration_ms)
end

local function strip_ansi(s)
  -- CSI sequences (SGR + everything else with the standard parameter shape).
  s = s:gsub('\27%[[%d;?]*[%a@-~]', '')
  -- OSC sequences, terminated by BEL or ST.
  s = s:gsub('\27%][^\7\27]*\7', '')
  s = s:gsub('\27%][^\7\27]*\27\\', '')
  -- Stray CR (the agent overdraws its input area heavily).
  s = s:gsub('\r', '')
  return s
end

local PAIR_IMAGE_PLACEHOLDER = '[Image]'
local PAIR_IMAGE_POLL_MS = 50
local PAIR_IMAGE_FIRST_POLL_MS = 20  -- short first deferral; catches fast renders
local PAIR_IMAGE_TIMEOUT_MS = 1000

-- Try to replace `[Image]` at `extmark_id` with `marker`. Returns true on
-- replacement, false if the placeholder span is no longer intact (user
-- deleted it, edited it, or buffer is gone). The extmark span tracks
-- buffer edits so concurrent typing before/after the placeholder is
-- preserved naturally.
--
-- On successful replacement, flashes the new marker with IncSearch for
-- 500ms — the visual confirmation that the async capture landed. Uses
-- pair's standard flash namespace, but deletes by extmark id (not
-- namespace-wide clear) so the flash doesn't compete with concurrent
-- quote-paste flashes living in the same namespace.
local function replace_placeholder(bufnr, ns, extmark_id, marker)
  if not vim.api.nvim_buf_is_valid(bufnr) then return false end
  local pos = vim.api.nvim_buf_get_extmark_by_id(bufnr, ns, extmark_id,
                                                 { details = true })
  if not pos or not pos[1] then return false end
  local row, col, details = pos[1], pos[2], pos[3] or {}
  local end_row, end_col = details.end_row, details.end_col
  if not end_row or not end_col then return false end
  local current = vim.api.nvim_buf_get_text(bufnr, row, col, end_row, end_col, {})[1]
  if current ~= PAIR_IMAGE_PLACEHOLDER then return false end

  -- Cursor-at-boundary fix. nvim_buf_set_text auto-adjusts the cursor when
  -- it lies *strictly past* the replaced range, but when the cursor sits
  -- exactly at the end column of the range (the common "Alt+i and don't
  -- type anything" case, where attach_image parked the cursor right after
  -- the placeholder's `]`), it gets left at the old col — which now lands
  -- *inside* the longer replacement text. Detect that case here and
  -- explicitly advance the cursor to the end of the new marker. Any other
  -- position (cursor moved away, or strictly past the placeholder because
  -- the user typed more) is left alone for nvim to adjust as usual.
  local win
  for _, w in ipairs(vim.api.nvim_list_wins()) do
    if vim.api.nvim_win_get_buf(w) == bufnr then win = w; break end
  end
  local advance_cursor = false
  if win then
    local cur_row, cur_col = unpack(vim.api.nvim_win_get_cursor(win))
    if cur_row - 1 == end_row and cur_col == end_col then
      advance_cursor = true
    end
  end

  vim.api.nvim_buf_set_text(bufnr, row, col, end_row, end_col, { marker })
  vim.api.nvim_buf_del_extmark(bufnr, ns, extmark_id)

  if advance_cursor then
    pcall(vim.api.nvim_win_set_cursor, win, { row + 1, col + #marker })
  end

  local flash_ns = vim.api.nvim_create_namespace('pair_flash')
  local flash_id = vim.api.nvim_buf_set_extmark(bufnr, flash_ns, row, col, {
    end_row = row,
    end_col = col + #marker,
    hl_group = 'IncSearch',
  })
  vim.defer_fn(function()
    if vim.api.nvim_buf_is_valid(bufnr) then
      pcall(vim.api.nvim_buf_del_extmark, bufnr, flash_ns, flash_id)
    end
  end, 500)
  return true
end

local function poll_capture(bufnr, ns, extmark_id, cap_path, done_path, started_ms, next_delay_ms)
  if vim.loop.now() - started_ms > PAIR_IMAGE_TIMEOUT_MS then
    vim.notify(
      'pair: capture timed out — [Image] placeholder left in place; edit manually if needed',
      vim.log.levels.WARN)
    -- Drop the extmark so future Alt+i runs don't accumulate ghosts;
    -- the literal "[Image]" text stays in the buffer.
    if vim.api.nvim_buf_is_valid(bufnr) then
      pcall(vim.api.nvim_buf_del_extmark, bufnr, ns, extmark_id)
    end
    return
  end
  if vim.fn.filereadable(done_path) == 1 then
    local content = ''
    local fh = io.open(cap_path, 'rb')
    if fh then
      content = fh:read('*a') or ''
      fh:close()
    end
    os.remove(cap_path)
    os.remove(done_path)
    local marker = strip_ansi(content):match(PAIR_IMAGE_MARKER_RE)
    if not marker then
      vim.notify(
        'pair: no image marker in agent output — [Image] placeholder left in place',
        vim.log.levels.WARN)
      if vim.api.nvim_buf_is_valid(bufnr) then
        pcall(vim.api.nvim_buf_del_extmark, bufnr, ns, extmark_id)
      end
      return
    end
    marker = normalize_image_marker(marker)
    if not replace_placeholder(bufnr, ns, extmark_id, marker) then
      vim.notify(
        'pair: [Image] placeholder was edited/removed before capture completed (got ' .. marker .. ')',
        vim.log.levels.WARN)
    end
    return
  end
  -- Schedule the next poll. First deferral defaults to PAIR_IMAGE_FIRST_POLL_MS
  -- (5ms) so a fast agent render gets picked up almost immediately; subsequent
  -- deferrals back off to PAIR_IMAGE_POLL_MS (50ms) to keep poll volume bounded
  -- over the full 1s window (~20 polls instead of ~200).
  vim.defer_fn(function()
    poll_capture(bufnr, ns, extmark_id, cap_path, done_path, started_ms, PAIR_IMAGE_POLL_MS)
  end, next_delay_ms or PAIR_IMAGE_POLL_MS)
end

local function attach_image()
  if not clipboard_has_image() then
    flash_at_cursor('[no image in clipboard]', 'WarningMsg', 1000)
    return
  end

  local tag = vim.env.PAIR_TAG or ''
  if tag == '' then
    vim.notify('pair: PAIR_TAG unset — not inside a pair session?',
               vim.log.levels.ERROR)
    return
  end
  local dd = pair_data_dir()
  local pid_path = dd .. '/pair-wrap-pid-' .. tag
  local cap_path = dd .. '/image-capture-' .. tag
  local done_path = cap_path .. '.done'

  -- Read pid (file I/O — microseconds). We defer the kill -0 alive check
  -- to the SIGUSR1 below; an explicit kill -0 here would fork a subprocess
  -- on the hot path before the placeholder is even visible.
  local pid
  do
    local fh = io.open(pid_path, 'r')
    if fh then
      local line = fh:read('*l')
      fh:close()
      if line then pid = line:match('%d+') end
    end
  end
  if not pid then
    vim.notify('pair: pair-wrap pid missing — restart the pair session (Alt+n)',
               vim.log.levels.ERROR)
    return
  end

  -- Insert the placeholder NOW, before any subprocess calls. nvim_buf_set_text
  -- is cheap (microseconds); what made it appear "late" was the ~100-200ms
  -- of synchronous vim.fn.system calls below — nvim doesn't redraw the buffer
  -- until the function returns, so the placeholder didn't paint until SIGUSR1
  -- + the three zellij actions had all blocked. By inserting first and
  -- deferring the rest via vim.schedule, the placeholder lands on screen on
  -- the next event-loop tick (within a frame, ~16ms) while the slow IPC runs
  -- afterwards.
  local bufnr = vim.api.nvim_get_current_buf()
  local row, col = unpack(vim.api.nvim_win_get_cursor(0))
  vim.api.nvim_buf_set_text(bufnr, row - 1, col, row - 1, col, { PAIR_IMAGE_PLACEHOLDER })
  local end_col = col + #PAIR_IMAGE_PLACEHOLDER
  local ns = vim.api.nvim_create_namespace('pair_image_pending')
  local extmark_id = vim.api.nvim_buf_set_extmark(bufnr, ns, row - 1, col, {
    end_row = row - 1,
    end_col = end_col,
    -- Gravity: text inserted at either edge should stay OUTSIDE the
    -- placeholder span. Start has right_gravity=true (default) so text
    -- inserted at the start col goes before the span; end has
    -- end_right_gravity=false so text at the end col goes after the span.
    -- This is what lets the user keep typing right after the placeholder
    -- — their text doesn't pollute the span, and replacement preserves it.
    right_gravity = true,
    end_right_gravity = false,
  })
  vim.api.nvim_win_set_cursor(0, { row, end_col })

  -- Defer the subprocess work. vim.schedule fires on the next event-loop
  -- iteration after this function returns, so the buffer redraw (which
  -- paints the placeholder) gets a chance to run first.
  vim.schedule(function()
    -- Pre-clean any stale sentinel from a previous Alt+i. The capture
    -- file itself doesn't need clearing — pair-wrap truncates it on write.
    os.remove(done_path)
    -- Arm capture. If pair-wrap is gone, SIGUSR1 fails — surface a clear
    -- error and drop the extmark, but leave the literal "[Image]" in the
    -- buffer for the user to edit/delete (consistent with the timeout
    -- path's posture: never silently mutate the user's text on failure).
    vim.fn.system({ 'kill', '-USR1', pid })
    if vim.v.shell_error ~= 0 then
      vim.notify(
        'pair: pair-wrap (pid ' .. pid .. ') not running — placeholder left in place; restart the pair session (Alt+n)',
        vim.log.levels.ERROR)
      if vim.api.nvim_buf_is_valid(bufnr) then
        pcall(vim.api.nvim_buf_del_extmark, bufnr, ns, extmark_id)
      end
      return
    end
    -- Send Ctrl+V to the agent pane. Order is fixed: window must be open
    -- (above) before the bytes flow back, or we miss the marker.
    if has_ui() then
      vim.fn.system('zellij action move-focus up')
      vim.fn.system('zellij action write 22')
      vim.fn.system('zellij action move-focus down')
    end
    poll_capture(bufnr, ns, extmark_id, cap_path, done_path,
                 vim.loop.now(), PAIR_IMAGE_FIRST_POLL_MS)
  end)
end

-- ---------------------------------------------------------------------------
-- PairPasteQuote: triggered from `pair clip clipboard-to-pane` after a copy_command
-- selection. The hand-off delivers the *raw* clipboard body via
-- $PAIR_DATA_DIR/quote-<tag>; we decide the formatting here based on where
-- the cursor is.
--
--   * col == 0  → quote-mode. The user is at the start of a line, treating
--                 the selection as a fresh block. Reflow with par, prefix
--                 every line with `> `, scroll the first inserted line to
--                 the top of the window (zt), flash the block, land on a
--                 single empty line below in insert mode.
--
--   * col >  0  → inline-mode. The user is mid-text and stitching the
--                 selection in at the cursor. Reflow with par so hard-wrapped
--                 lines collapse into one continuous run (paragraph breaks
--                 preserved); no prefix, no scroll. Flash the inserted span,
--                 leave the cursor in insert mode immediately after it.
--
-- A single namespace `pair_flash` carries the IncSearch highlight in both
-- modes; cleared after 500ms via vim.defer_fn.
-- ---------------------------------------------------------------------------

local pair_flash_ns = vim.api.nvim_create_namespace('pair_flash')

local function quote_path()
  local data_dir = os.getenv('PAIR_DATA_DIR')
                or (os.getenv('XDG_DATA_HOME') or vim.fn.expand('~/.local/share'))
                   .. '/pair'
  return data_dir .. '/quote-' .. pair_tag()
end

local function clear_flash_after(buf, ms)
  vim.defer_fn(function()
    if vim.api.nvim_buf_is_valid(buf) then
      vim.api.nvim_buf_clear_namespace(buf, pair_flash_ns, 0, -1)
    end
  end, ms)
end

local function reflow_par(body)
  -- par 1000 (large width) acts as a paragraph rejoin/reflow; safe to skip
  -- if par is missing or errors out.
  if vim.fn.executable('par') == 0 then return body end
  local out = vim.fn.system({ 'par', '1000' }, body)
  if vim.v.shell_error ~= 0 then return body end
  return out
end

-- Strip leading/trailing whitespace (spaces, tabs, newlines) from the whole
-- selection. Mouse drags often grab a stray space or newline at either end;
-- trimming makes the inline-paste separator logic predictable (it can rely
-- on body never ending in whitespace) and keeps quote blocks from rendering
-- with `>     ` indents or trailing spaces. Interior indentation is left
-- alone — only the global ends are trimmed.
local function trim_ends(s)
  s = s:gsub('^%s+', '')
  s = s:gsub('%s+$', '')
  return s
end

local function paste_as_quote(body, row)
  body = trim_ends(body)
  if body == '' then return end
  local reflowed = trim_ends(reflow_par(body))
  local quoted_lines = {}
  for line in (reflowed .. '\n'):gmatch('([^\n]*)\n') do
    quoted_lines[#quoted_lines + 1] = '> ' .. line
  end
  if #quoted_lines == 0 then return end

  local buf = vim.api.nvim_get_current_buf()
  local cur_line = vim.api.nvim_buf_get_lines(buf, row - 1, row, false)[1] or ''
  -- If the cursor's line is empty we replace it (so we don't end up with
  -- a leading blank above the quote); otherwise we insert above and let
  -- the existing line slide down.
  local insert_start, insert_end
  if cur_line == '' then
    insert_start, insert_end = row - 1, row
  else
    insert_start, insert_end = row - 1, row - 1
  end

  local payload = {}
  for _, l in ipairs(quoted_lines) do payload[#payload + 1] = l end
  payload[#payload + 1] = ''  -- the empty line the cursor will land on

  vim.api.nvim_buf_set_lines(buf, insert_start, insert_end, false, payload)

  local block_start = insert_start                  -- 0-indexed, inclusive
  local block_end   = insert_start + #quoted_lines  -- 0-indexed, exclusive
  local cursor_row  = block_end + 1                 -- 1-indexed empty line

  vim.api.nvim_win_set_cursor(0, { block_start + 1, 0 })
  vim.cmd('normal! zt')
  vim.api.nvim_win_set_cursor(0, { cursor_row, 0 })

  vim.api.nvim_buf_clear_namespace(buf, pair_flash_ns, 0, -1)
  for i = block_start, block_end - 1 do
    vim.api.nvim_buf_add_highlight(buf, pair_flash_ns, 'IncSearch', i, 0, -1)
  end
  clear_flash_after(buf, 500)

  vim.cmd('startinsert')
end

local function paste_inline(body, row, col)
  body = trim_ends(body)
  if body == '' then return end
  body = trim_ends(reflow_par(body))
  -- Inline insertions are followed by user-typed continuation. body has
  -- been trimmed, so it never ends in whitespace — append a single space
  -- as the separator so the user can start typing immediately.
  body = body .. ' '
  local lines = vim.split(body, '\n', { plain = true })

  local buf = vim.api.nvim_get_current_buf()
  vim.api.nvim_buf_set_text(buf, row - 1, col, row - 1, col, lines)

  local end_row, end_col
  if #lines == 1 then
    end_row = row - 1
    end_col = col + #lines[1]
  else
    end_row = row - 1 + #lines - 1
    end_col = #lines[#lines]
  end

  vim.api.nvim_buf_clear_namespace(buf, pair_flash_ns, 0, -1)
  vim.api.nvim_buf_set_extmark(buf, pair_flash_ns, row - 1, col, {
    end_row = end_row,
    end_col = end_col,
    hl_group = 'IncSearch',
  })
  clear_flash_after(buf, 500)

  -- Place cursor at the end of the inserted text, then enter insert mode.
  -- nvim normal-mode cursors clamp to (line length - 1), but startinsert
  -- promotes us to insert mode where end-of-line positioning works
  -- correctly — type-next-character lands at end_col as intended.
  vim.api.nvim_win_set_cursor(0, { end_row + 1, end_col })
  vim.cmd('startinsert')
end

function _G.PairPasteQuote()
  local f = io.open(quote_path(), 'r')
  if not f then return end
  local body = f:read('*a')
  f:close()
  if not body or body == '' then return end

  local row, col = unpack(vim.api.nvim_win_get_cursor(0))
  -- Defensive: nvim returns 1-indexed row but in some uninitialized states
  -- (e.g. just-opened headless instances) it can return 0. Clamp.
  if row < 1 then row = 1 end
  if col == 0 then
    paste_as_quote(body, row)
  else
    paste_inline(body, row, col)
  end
end

-- ---------------------------------------------------------------------------
-- send_and_clear: Alt+Return — send entire buffer, log, clear, reset.
-- With no_submit=true (Alt+Shift+Return) the body lands in the agent's
-- composer followed by a literal newline but is NOT submitted; everything
-- else (strip, log, queue handling, clear, reset to *) is identical.
-- ---------------------------------------------------------------------------

local function send_and_clear(no_submit)
  local body = buffer_text()
  if body:match('^%s*$') then return end
  -- Strip-then-check happens before any side effects: a comment-only buffer
  -- is a no-op send, so it must NOT consume a +N queue item or append to log.
  local stripped = strip_comments(body)
  if stripped:match('^%s*$') then return end

  local from_queue = (type(nav.pos) == 'table' and nav.pos.kind == 'queue')

  -- send-from-+N consumes that queue file. Resolve the selected item by its
  -- filename KEY *now*, before any queue mutation below. The key is a stable
  -- identity; the display index (nav.pos.n) is not — enqueueing the draft
  -- shifts every index by one. Removing by a stale index is the duplication
  -- bug: queue_key_for_n then points at the wrong file (or nil), the selected
  -- item never gets removed, and it ends up at both +N and -1.
  local selected_key = from_queue and queue_key_for_n(nav.pos.n) or nil

  -- Sending from the future queue while * holds an in-progress draft: park
  -- that draft as a real queue item (front) rather than leaving it dangling
  -- at *. The draft was autosaved to disk on the nav into the queue, so read
  -- it from there. Push before the removal so it survives regardless of order;
  -- keys don't move on insert, so selected_key stays valid. Comment-only /
  -- empty drafts have nothing to park — skip them (mirrors the send no-op rule).
  local enqueued_draft = false
  if from_queue then
    local star_body = read_file(draft_path_for_tag()) or ''
    if not strip_comments(star_body):match('^%s*$') then
      queue_push_front(star_body)
      enqueued_draft = true
    end
  end

  local consumed_queue = false
  if selected_key then
    queue_remove(selected_key)
    consumed_queue = true
  end

  local was_at_star = (nav.pos == '*')

  -- Log the unstripped body (the user's authored text, comments and all),
  -- send only the stripped version to the agent.
  append_log(body)
  send_to_agent(stripped, no_submit)

  -- Return to *. The just-sent body's === lines become the new sticky set
  -- for *, regardless of which slot we sent from. When sent from -N or +N,
  -- *'s WIP non-comment content is preserved (only its old comments are
  -- replaced) so we don't clobber a half-typed draft — UNLESS we just parked
  -- that draft into the queue, in which case * is now empty and should reset
  -- to the sent item's stickies + a fresh line (treat * as empty).
  nav.pos = '*'
  local lines = apply_sticky_to_star(body, was_at_star or enqueued_draft)
  vim.api.nvim_buf_set_lines(0, 0, -1, false, lines)
  pcall(vim.api.nvim_win_set_cursor, 0, { #lines, 0 })
  -- Persist via :w so vim's b_mtime_read tracks the on-disk mtime. Writing
  -- the file out-of-band (io.open) would leave nvim's recorded mtime stale,
  -- and the next autosave would trip the "file changed since reading it" prompt.
  vim.cmd('silent! write')
  nav.baseline = table.concat(lines, '\n')
  refresh_statusline()
  if consumed_queue then flash_queue_count() end
  vim.cmd('startinsert')
end

-- ---------------------------------------------------------------------------
-- as-you-type path completion (plugin-free, fzf-style on the basename)
--
-- TextChangedI/P fires per keystroke. We pull the path-shaped token at the
-- cursor (anything ending in `/foo` or starting with `~`/`./`), split it on
-- the last `/` into <dir>/<filter>, list <dir> via getcompletion, then fuzzy-
-- filter by <filter> via matchfuzzy. Results go straight to vim.fn.complete()
-- — bypassing <C-x><C-f> avoids feedkeys reentrancy.
--
-- Reload via `:luafile $PAIR_HOME/nvim/init.lua` (works because all autocmds
-- live in the `pair` augroup with clear=true).
-- ---------------------------------------------------------------------------

-- z= is a normal-mode "pick a fix" gesture, not a typing session. While set,
-- bare digits 1-9 pick a spell suggestion (spell_pick_digit) instead of
-- inserting, and CompleteDone returns the user to normal mode. Declared here
-- (above the completers) so path_complete / word_complete can clear it the
-- moment they replace the menu with a typed-token popup — that's the only way
-- the spell behavior could otherwise leak past a momentary popup. Set by
-- spell_suggest_popup; cleared on CompleteDone / InsertLeave too.
local spell_popup_active = false

-- Two ways to wrap completion strings into vim.fn.complete()'s dict form.
--
-- plain_items — no labels. The uniform mechanism for every as-you-type menu
-- (path / word / spell): all three render identically with no numbering, picked
-- by arrows / Tab / <C-y>. (The <M-1>..<M-9> keys below still pick the Nth item
-- of any popup; they're just no longer advertised.)
local function plain_items(words)
  local items = {}
  for i, w in ipairs(words) do items[i] = { word = w } end
  return items
end

-- indexed_items — prefixes the first 9 abbrs with `<prefix>N ` to advertise a
-- numbered quick-pick. Used ONLY by the explicit z= spell popup, which passes
-- '' so the label is a bare digit (1..9) and bare digits pick it (see
-- spell_pick_digit). .word stays clean so <C-y> / CompleteDone see the real
-- candidate; items past 9 are arrow-navigable.
local function indexed_items(words, label_prefix)
  label_prefix = label_prefix or ''
  local items = {}
  for i, w in ipairs(words) do
    if i <= 9 then
      items[i] = { word = w, abbr = label_prefix .. i .. ' ' .. w }
    else
      items[i] = { word = w }
    end
  end
  return items
end

-- Lua pattern for a path-ish token: word chars, slash, dot, dash, underscore,
-- tilde. Matches the longest such run anchored at end-of-prefix.
local PATH_TOKEN_RE = '([%w%./%-_~]+)$'

-- A token is a "path" iff the user explicitly indicated one with a leading
-- `/`, `~`, or `./` / `../` etc. Plain `bin/pair-wrap` is *not* a path here
-- — it's an entity that lives in the agent-output spans, handled by
-- word_complete. The explicit-prefix rule keeps the two completion systems
-- mutually exclusive at trigger time.
local function token_is_path(token)
  return token:match('^[/~]') ~= nil or token:match('^%.+/') ~= nil
end

local function path_complete()
  local line = vim.api.nvim_get_current_line()
  local col = vim.fn.col('.') - 1  -- 0-indexed cursor byte position
  if col == 0 then return end
  local before = line:sub(1, col)
  local token_start, _, token = before:find(PATH_TOKEN_RE)
  if not token then return end
  if not token_is_path(token) then return end

  local dir, filter = token:match('^(.*/)([^/]*)$')
  if not dir then dir, filter = '', token end

  local ok, matches = pcall(vim.fn.getcompletion, dir, 'file')
  if not ok or type(matches) ~= 'table' or #matches == 0 then return end
  if filter ~= '' then
    local ok2, fuzzy = pcall(vim.fn.matchfuzzy, matches, filter)
    if ok2 then matches = fuzzy end
  end
  if #matches == 0 then return end

  -- complete() col is 1-indexed start of the span being replaced. We're about
  -- to replace any spell popup with a typed-token menu, so bare digits go back
  -- to being literal (see spell_popup_active).
  spell_popup_active = false
  vim.fn.complete(token_start, plain_items(matches))
  return true
end

-- Word completion. Triggered alongside path_complete on every TextChangedI.
-- Two sources of candidates:
--   1. The current draft buffer — `[%w_]+` words the user has typed.
--   2. $PAIR_DATA_DIR/agent-output-<tag> — colored spans extracted from
--      the agent's output by pair-wrap. Each line is
--      `<color>\t<count>\t<span>`, where <color> is the SGR foreground id
--      ("36" for cyan, "5;75" for 256-color, "2;R;G;B" for RGB) and
--      <count> is the number of times the agent has emitted that span.
--      Filtering by color is essential: Claude paints code spans (paths,
--      commands, `<M-CR>`) in cyan, but also paints headers/dim-text in
--      other colors that we must reject.
--
-- Matching is prefix-anchored: a candidate qualifies iff it starts with
-- the typed prefix. Fuzzy matching produced too many false positives (e.g.
-- `tel` surfaced anything containing t, e, l in order); the prefix
-- discipline keeps the popup quiet until the user has actually typed the
-- start of something.
--
-- Ranking: agent spans are scored by `(count + α·picks) · 0.5^(rank/H)`
-- where rank is age in lines (0 = newest), picks is how many times the
-- user has accepted that span from a popup (tracked in agent-picks-<tag>),
-- α=PICK_WEIGHT, H=DECAY_HALFLIFE. Only the top POOL_CAP agent spans by
-- score are eligible — keeps the popup tight even with 1000 stored spans.
-- Draft-buffer words are always eligible (no cap) at a fixed mid score.
--
-- Trigger after 1 typed char; candidates filtered to 5+ chars. Override
-- the default color allowlist with PAIR_AGENT_SPAN_COLORS (csv of color
-- ids — inspect $PAIR_DATA_DIR/agent-output-<tag> to see what's emitted).
local WORD_TRIGGER_MIN = 1
local WORD_CANDIDATE_MIN = 5
local POOL_CAP = 100        -- agent spans eligible for completion
local DECAY_HALFLIFE = 300  -- in line-rank units; 0.5^(rank/H)
local PICK_WEIGHT = 5       -- one user pick worth this many agent emissions
local DRAFT_SCORE = 1.0     -- score assigned to draft-buffer words
-- Prefix charset includes `-`, `.`, `/`, `$`, `+`, `<`, `>`, `{`, `}`,
-- `[`, `]` so entity-style tokens get captured whole: `draft-<tag>.md`,
-- `pair-wrap`, `lessons.md`, `bin/pair-wrap`, `$PAIR_HOME`,
-- `${XDG_DATA_HOME}/pair/`, `Alt+Return`, `<M-CR>`, `[Image #1]`-ish
-- bracket tokens. `~` stays out — leading `~` triggers path_complete;
-- word_complete bails via token_is_path below when the prefix is
-- path-shaped.
local WORD_TOKEN_RE = '([%w_%-./$+<>{}%[%]]+)$'

-- Color allowlist for agent-output spans. Each agent paints in a different
-- palette; the default per-agent table covers the colors we've actually
-- observed. Users can override via PAIR_AGENT_SPAN_COLORS=<csv> (comma-
-- separated; semicolons inside an entry are part of the color id, e.g.
-- `2;177;185;249` for an RGB triple).
local AGENT_SPAN_DEFAULTS = {
  -- Claude Code (claude.ai's TUI): code spans painted in periwinkle RGB.
  -- Inspect $PAIR_DATA_DIR/agent-output-<tag> to update.
  claude = { '2;177;185;249' },
}

local AGENT_SPAN_COLORS = (function()
  local env = os.getenv('PAIR_AGENT_SPAN_COLORS')
  local set = {}
  if env and env ~= '' then
    for c in env:gmatch('[^,]+') do set[c] = true end
    return set
  end
  local agent = os.getenv('PAIR_AGENT') or 'claude'
  for _, c in ipairs(AGENT_SPAN_DEFAULTS[agent] or {}) do set[c] = true end
  return set
end)()

local function agent_output_path()
  return pair_data_dir() .. '/agent-output-' .. pair_tag()
end

-- ----- pick tracking -------------------------------------------------------
-- Per-tag file `agent-picks-<tag>`: lines `<count>\t<span>`, oldest first
-- (most recent picks at end). Cap at PICK_CAP entries; LRU eviction by
-- pick recency. Loaded lazily on first use; flushed debounced (500ms) so
-- a burst of picks costs one rename. word_complete consults `picks` to
-- weight agent spans (PICK_WEIGHT × picks added to emission count).
local PICK_CAP = 5000
local picks = {}            -- span -> int count
local pick_order = {}       -- span list, oldest-pick first
local picks_loaded = false
local picks_dirty = false
local picks_flush_timer = nil

local function agent_picks_path()
  return pair_data_dir() .. '/agent-picks-' .. pair_tag()
end

local function picks_load()
  if picks_loaded then return end
  picks_loaded = true
  local f = io.open(agent_picks_path(), 'r')
  if not f then return end
  for l in f:lines() do
    local n, span = l:match('^(%d+)\t(.+)$')
    if n and span then
      picks[span] = tonumber(n)
      pick_order[#pick_order + 1] = span
    end
  end
  f:close()
end

local function picks_flush()
  picks_flush_timer = nil
  if not picks_dirty then return end
  picks_dirty = false
  local path = agent_picks_path()
  local tmp = path .. '.tmp'
  local f = io.open(tmp, 'w')
  if not f then return end
  for _, span in ipairs(pick_order) do
    f:write(string.format('%d\t%s\n', picks[span] or 0, span))
  end
  f:close()
  os.rename(tmp, path)
end

local function picks_schedule_flush()
  if picks_flush_timer then return end
  picks_flush_timer = vim.loop.new_timer()
  picks_flush_timer:start(500, 0, vim.schedule_wrap(picks_flush))
end

local function picks_bump(span)
  if not span or span == '' then return end
  picks_load()
  if picks[span] then
    -- move-to-end: scan from tail since recent picks tend to repeat.
    for i = #pick_order, 1, -1 do
      if pick_order[i] == span then
        table.remove(pick_order, i)
        break
      end
    end
  end
  picks[span] = (picks[span] or 0) + 1
  pick_order[#pick_order + 1] = span
  while #pick_order > PICK_CAP do
    local oldest = table.remove(pick_order, 1)
    picks[oldest] = nil
  end
  picks_dirty = true
  picks_schedule_flush()
end

local function word_complete()
  local line = vim.api.nvim_get_current_line()
  local col = vim.fn.col('.') - 1
  if col == 0 then return end
  local before = line:sub(1, col)
  local token_start, _, prefix = before:find(WORD_TOKEN_RE)
  if not prefix or #prefix < WORD_TRIGGER_MIN then return end
  -- Explicitly-prefixed paths belong to path_complete; bail rather than
  -- clobber its popup. Mirrors path_complete's trigger condition.
  if token_is_path(prefix) then return end

  picks_load()

  -- Build agent-span pool with scores. File order = LRU recency
  -- (oldest first, newest last), so rank = total - i gives 0 for newest.
  local spans = {}
  local f = io.open(agent_output_path(), 'r')
  if f then
    for l in f:lines() do
      local color, count, span = l:match('^([^\t]+)\t(%d+)\t(.+)$')
      if color and AGENT_SPAN_COLORS[color] then
        spans[#spans + 1] = { span = span, count = tonumber(count) }
      end
    end
    f:close()
  end
  local total = #spans
  for i, s in ipairs(spans) do
    local rank = total - i
    local decay = 0.5 ^ (rank / DECAY_HALFLIFE)
    s.score = (s.count + PICK_WEIGHT * (picks[s.span] or 0)) * decay
  end
  table.sort(spans, function(a, b) return a.score > b.score end)
  for i = #spans, POOL_CAP + 1, -1 do spans[i] = nil end

  local plen = #prefix
  local prefix_lower = prefix:lower()
  local seen = { [prefix] = true }
  local matches = {}

  local function add(w, score)
    if #w >= WORD_CANDIDATE_MIN
       and w:sub(1, plen):lower() == prefix_lower
       and not seen[w] then
      seen[w] = true
      matches[#matches + 1] = { word = w, score = score }
    end
  end

  -- Draft-buffer words: always eligible, fixed mid score.
  for _, l in ipairs(vim.api.nvim_buf_get_lines(0, 0, -1, false)) do
    for w in l:gmatch('[%w_]+') do add(w, DRAFT_SCORE) end
  end

  -- Top-POOL_CAP agent spans by score.
  for _, s in ipairs(spans) do add(s.span, s.score) end

  if #matches == 0 then return end
  table.sort(matches, function(a, b) return a.score > b.score end)
  local words = {}
  for i, m in ipairs(matches) do words[i] = m.word end
  -- Replacing any spell popup with a typed-token menu → bare digits literal.
  spell_popup_active = false
  vim.fn.complete(token_start, plain_items(words))
  return true
end

-- z= replacement: instead of vim's default "Choose a number:" prompt,
-- pop up the standard completion menu populated with spellsuggest() results
-- so the user picks via Tab / CR like every other completion in the draft.
--
-- Implementation: find word bounds around the cursor (alpha + apostrophe),
-- check that spell flags it, move cursor to end-of-word, enter insert mode,
-- and call vim.fn.complete() with span = the misspelled word. Picking a
-- suggestion replaces the word; dismissing leaves it intact. (spell_popup_active
-- — the flag that turns bare digits into pickers — is declared above the
-- completers; see its comment.)
local function spell_suggest_popup()
  local row, col = unpack(vim.api.nvim_win_get_cursor(0))  -- col is 0-indexed
  local line = vim.api.nvim_get_current_line()
  if line == '' then return end
  -- 1-indexed positions in line. Start at col+1 (cursor's char), expand.
  local s, e = col + 1, col + 1
  if s > #line then s = #line; e = #line end
  while s > 1 and line:sub(s - 1, s - 1):match("[%a']") do s = s - 1 end
  while e <= #line and line:sub(e, e):match("[%a']") do e = e + 1 end
  local word = line:sub(s, e - 1)
  if word == '' then
    vim.notify('pair: no word at cursor', vim.log.levels.INFO)
    return
  end

  local bad = vim.fn.spellbadword(word)
  if not bad or bad[1] == '' then
    vim.notify('pair: "' .. word .. '" is not misspelled', vim.log.levels.INFO)
    return
  end

  local suggestions = vim.fn.spellsuggest(word, 12)
  if not suggestions or #suggestions == 0 then
    vim.notify('pair: no suggestions for "' .. word .. '"', vim.log.levels.INFO)
    return
  end

  -- Park cursor at end-of-word in 0-indexed col, enter insert mode, then
  -- (after the mode switch lands) trigger the completion popup. complete()
  -- replaces text from start_col to cursor — i.e. the misspelled word.
  vim.api.nvim_win_set_cursor(0, { row, e - 1 })
  spell_popup_active = true
  vim.cmd('startinsert')
  vim.schedule(function()
    vim.fn.complete(s, indexed_items(suggestions, ''))
  end)
end

-- As-you-type spell typeahead. Runs only as a fallback in run_completers,
-- after path_complete and word_complete have both declined — so a word with
-- real buffer/agent completions still shows those, and a word that matches
-- nothing AND is misspelled gets a spelling-fix menu instead. Mirrors the
-- nvim-cmp + cmp-spell behavior in the user's own config, built plugin-free
-- on spellsuggest().
--
-- Unlike spell_suggest_popup (the on-demand z= gesture), this does NOT set
-- spell_popup_active: we're mid-type, so bare digits must stay literal text
-- and CompleteDone must not bounce us back to normal mode. A suggestion is
-- accepted via CR / Tab / <C-y> / arrows.
--
-- Uses plain_items, same as the path/word menus: every as-you-type popup is
-- unlabelled (uniform, no numbering), picked by arrows / Tab / <C-y>. Bare
-- digits stay literal mid-type — only the explicit z= popup repurposes them.
local SPELL_TRIGGER_MIN = 4   -- min misspelled-word length before suggesting
local SPELL_MAX_SUGGEST = 9   -- max suggestions shown in the menu
local function spell_complete()
  local line = vim.api.nvim_get_current_line()
  local col = vim.fn.col('.') - 1   -- 0-indexed byte position of the cursor
  if col == 0 then return end
  -- Fire only at end-of-word: if the char under the cursor is alphabetic the
  -- cursor sits inside a word, and complete()'s replace span (start..cursor)
  -- would strand the tail. Bail so mid-word edits aren't mangled.
  if line:sub(col + 1, col + 1):match("[%a']") then return end
  -- The alphabetic run ending at the cursor is the word being typed.
  local s = col + 1
  while s > 1 and line:sub(s - 1, s - 1):match("[%a']") do s = s - 1 end
  local word = line:sub(s, col)
  if #word < SPELL_TRIGGER_MIN then return end

  local bad = vim.fn.spellbadword(word)
  if not bad or bad[1] == '' then return end   -- correctly spelled → nothing to do

  local suggestions = vim.fn.spellsuggest(word, SPELL_MAX_SUGGEST)
  if not suggestions or #suggestions == 0 then return end

  vim.fn.complete(s, plain_items(suggestions))
  return true
end

local function pum_visible()
  return vim.fn.pumvisible() == 1
end

local function pum_has_selection()
  -- complete_info().selected is -1 when nothing highlighted.
  return vim.fn.complete_info({ 'selected' }).selected ~= -1
end

-- Decide what <CR> feeds in insert mode given completion-popup state. Pure
-- (three booleans → a key string) so it's unit-testable without a live popup,
-- which needs a UI headless nvim lacks. Cases:
--   no popup                       → <CR>        plain newline (unchanged).
--   popup + selection              → <C-y>       accept the highlighted item.
--   popup, no selection, typing    → <C-e><CR>   dismiss the menu, THEN newline.
--   popup, no selection, momentary → <CR>        clean dismiss, NO newline.
-- The typing case is the fix (#65): under completeopt=noselect nothing is ever
-- auto-highlighted, so "popup up, nothing picked" is the common case — and a
-- bare <CR> there only closes the menu, swallowing the newline. <C-e> cancels
-- completion (keeping exactly what was typed), so the following <CR> is
-- processed as an ordinary newline. <CR> in the draft must ALWAYS break the line
-- when the user hasn't picked a completion.
--
-- `momentary` is the escape hatch: the insert <CR> map is a shared chokepoint,
-- and the normal-mode z= spell popup (spell_suggest_popup, gated by
-- spell_popup_active) is a transient picker whose contract is "dismiss leaves
-- the text intact" — it does NOT want a newline. For it we keep the old bare
-- <CR> (clean dismiss); only as-you-type draft completions want the newline.
-- Threaded as an arg (not read here) so cr_keys stays pure. (Caught in the #65
-- milestone review — the Spec's three-state table missed this second consumer.)
local function cr_keys(visible, has_selection, momentary)
  if not visible then return '<CR>' end
  if has_selection then return '<C-y>' end
  if momentary then return '<CR>' end
  return '<C-e><CR>'
end
-- Exposed for tests/cr-newline-test.sh (decision table); the live map below
-- feeds the real popup state in.
_G.PairCRKeys = cr_keys

-- ---------------------------------------------------------------------------
-- prompt history & queue navigation (issue #000015)
--
-- Position model (M1: history-only):
--   nav.pos = '*'                          -- the persistent draft slot
--          | { kind='history', n=N }       -- Nth-most-recent log entry; n=1 is newest
--          | { kind='queue',   n=N }       -- (M2+) Nth queue item; n=1 is +1 (front)
--
-- nav.baseline = the buffer content at the moment we last loaded the slot.
-- Used to detect dirtiness for the (M3+) edit-flow rules. In M1 we discard
-- dirty content with a warning.
--
-- See workshop/issues/000015-prompt-history-queue.md for the full spec.
-- ---------------------------------------------------------------------------

local function pos_label(pos)
  if pos == '*' then return '*' end
  if pos.kind == 'history' then return '-' .. pos.n end
  if pos.kind == 'queue'   then return '+' .. pos.n end
  return '?'
end

local function load_baseline_for_current_pos()
  if nav.pos == '*' then
    return read_file(draft_path_for_tag())
  end
  if nav.pos.kind == 'history' then
    local entries = read_history()
    -- entries[#entries] is most recent ⇒ pos.n=1.
    local idx = #entries - nav.pos.n + 1
    return entries[idx] or ''
  end
  if nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    return key and queue_read(key) or ''
  end
  return ''
end

-- Dirty only matters for -N: history is immutable, so an edit there is a
-- pending fork that must explicitly become a send / a queue entry / a discard.
-- * and +N are mutable (their edits autosave to the underlying file), so they
-- have no dirty concept from the user's perspective.
--
-- Comment-only / blank-line-only edits don't count as dirty: they vanish under
-- strip_comments before the agent sees them, so there's no fork to resolve.
-- They are persisted back into the log entry by autosave_current_slot, so
-- annotations on history survive navigation and nvim restart.
local function is_dirty_history_slot()
  return type(nav.pos) == 'table'
     and nav.pos.kind == 'history'
     and strip_comments(buffer_text()) ~= strip_comments(nav.baseline)
end

-- Statusline format:
--   " Alt: <- history H < pos[*][ HINT] > Q queued -> "
-- The flanking arrows hint Alt+← / Alt+→. The trailing "*" on `pos` shows
-- when on -N with an unsent fork. The HINT inside the brackets is contextual:
--   * or -N : " [q=queue]" — Alt+q parks the current buffer as +1.
--   +N      : " [⌫=del]"   — Alt+BS deletes the current queue item.
-- Suppressed on +N for [q=queue] because Alt+q from +N is "move-to-front",
-- a different mental action that doesn't grow the queue.
--
-- Wrapped in a pcall guard so any edge-case error in is_dirty_history_slot
-- / read_history / queue_count can't blank the bar — fall back to a minimal
-- safe string.
-- Right-aligned cheatsheet rendered at the end of the statusline. Listed
-- in priority order — when the terminal is too narrow for the full set,
-- entries drop from the bottom (lowest priority first) until what's left
-- fits in the available space. At a minimum we try to keep Alt+h so the
-- user always has a discoverable path to the full keybind help.
local PAIR_CHEATS = {
  { key = 'Alt+h',  label = 'help'   },
  { key = 'Alt+⏎',  label = 'send'   },
  { key = 'Alt+q',  label = 'queue'  },
  { key = 'Alt+x',  label = 'quit'   },
  { key = 'Alt+d',  label = 'detach' },
}

-- Display width of a statusline format string, ignoring vim's inline
-- format codes (highlight groups, alignment, truncation markers) since
-- those don't render as visible cells. strdisplaywidth handles the
-- multibyte glyphs (⏎, ⌫, etc).
local function pair_statusline_width(s)
  local stripped = s:gsub('%%#[^#]*#', '')
                    :gsub('%%%*', '')
                    :gsub('%%=', '')
                    :gsub('%%<', '')
  return vim.fn.strdisplaywidth(stripped)
end

local function pair_format_cheat(c)
  -- Key gets the actionable accent; label stays in the muted baseline.
  return string.format('%%#PairAltKey#%s%%* %s', c.key, c.label)
end

-- Build the right-aligned cheatsheet with progressive disclosure: walk
-- the priority list and accumulate as many entries as fit in `budget`
-- display columns. Returns the accumulated format string (possibly
-- empty if even the highest-priority entry doesn't fit).
local function pair_build_cheatsheet(budget)
  if budget <= 0 then return '' end
  local sep = '  '
  local out = ''
  for i, c in ipairs(PAIR_CHEATS) do
    local part = (i == 1 and '' or sep) .. pair_format_cheat(c)
    local candidate = out .. part
    if pair_statusline_width(candidate) > budget then break end
    out = candidate
  end
  return out
end

-- Ephemeral right-end notification (#58). When set it REPLACES the cheatsheet
-- with a flashed message (e.g. "change log ready · Alt+l"); pair_flash_notify
-- sets it + a green PairNotify highlight, then a ~2s timer clears it back to the
-- cheatsheet. Declared here so pair_compose_statusline — the single right-end
-- renderer — can short-circuit to it in every mode.
local pair_notify = nil
local pair_notify_timer = nil

-- The active notification rendered as a right-aligned green segment, appended
-- past `left`. Single source for the format codes (%= right-align, PairNotify
-- highlight) so the compose paths can't drift.
local function pair_notify_segment(left)
  return left .. '%=%#PairNotify# ' .. pair_notify .. ' %* '
end

-- Compose a statusline with the cheatsheet right-aligned past `left`.
local function pair_compose_statusline(left)
  -- An active ephemeral notification owns the right end until its flash timer
  -- reverts it — show the green PairNotify message instead of the cheatsheet.
  if pair_notify then
    return pair_notify_segment(left)
  end
  -- #66 M3/M4d: while a review is open, the review segment replaces the
  -- rightmost cheatsheet. It carries its own %= so mode/file stay near the
  -- compact nav cluster while 🤖 agent/human counts are right-aligned.
  -- Read from the timer-cached value (a global set by the review do-block) so
  -- this hot path never shells git.
  if _G._pair_review_segment then
    local seg = _G._pair_review_segment()
    if seg then return left .. seg .. ' ' end
  end
  -- 6-cell minimum margin between the variable left segment and the
  -- cheatsheet. Capping the cheatsheet's budget at (columns - left - 6)
  -- bounds left+right ≤ columns - 6, so vim's %= autopads at least 6
  -- spaces in the middle no matter how wide the terminal is.
  local budget = vim.o.columns - pair_statusline_width(left) - 6
  local cheats = pair_build_cheatsheet(math.max(0, budget))
  if cheats == '' then return left end
  return left .. '%=' .. cheats .. ' '
end

-- Flash an ephemeral message on the right end of the statusline, then revert to
-- the cheatsheet after ~2s. Drives the "change log ready" build-complete signal
-- (#58): the draft pane is always on screen, so the operator sees it even while
-- working in the agent pane after a triggered-and-left build. Models
-- flash_queue_count — swap the highlight + refresh now, defer the revert.
-- Re-arming cancels the prior timer so a second notification isn't cut short.
local function pair_flash_notify(text)
  pair_notify = text
  -- Green background to grab attention (operator-specified). Set per-flash so it
  -- survives a colorscheme reload that would otherwise clear the group.
  vim.api.nvim_set_hl(0, 'PairNotify', { fg = '#04260a', bg = '#3fb950', bold = true })
  refresh_statusline()
  if pair_notify_timer then
    pcall(function() pair_notify_timer:stop() end)
    pcall(function() pair_notify_timer:close() end)
  end
  pair_notify_timer = vim.defer_fn(function()
    pair_notify = nil
    pair_notify_timer = nil
    refresh_statusline()
  end, 2000)
end
-- Exposed for the headless statusline test (drives the notify render path).
_G.PairFlashNotify = pair_flash_notify

_G._pair_review_compact_left = function()
  local ok, result = pcall(function()
    local h = #read_history()
    local q = queue_count()
    local pos = pos_label(nav.pos)
    if is_dirty_history_slot() then pos = pos .. '*' end
    local pos_seg = string.format('%%#PairPosLabel#%s%%*', pos)
    return string.format('-%d < %s > +%d • ', h, pos_seg, q)
  end)
  return ok and result or ' pair • '
end

function _G.PairStatusline()
  -- Minimized rung: nvim is collapsed to this single statusline row, so
  -- the buffer is invisible and the usual history/queue/position
  -- cluster has nothing to refer to. Replace it with a hint that names
  -- the keybind that grows the pane back. (the cheatsheet is
  -- intentionally omitted — the row is meant to read as a single
  -- focused hint.)
  --
  -- Leading whitespace gives the terminal cursor (which lives on this
  -- row since the buffer has zero visible lines) a few blank cells to
  -- land on instead of overprinting the hint text. nvim re-emits ?25h
  -- on redraws and we can't reliably suppress that, so the cursor
  -- block stays visible — but on a leading space it's unobtrusive.
  if pair_layout_state == 'minimized' then
    local base = '    %#PairAltKey#Alt+↑%* for pair input box '
    -- Surface an active notification even when collapsed — the build-complete
    -- flash matters most when the operator has minimized the draft to work.
    if pair_notify then
      return pair_notify_segment(base)
    end
    return base
  end
  if _G._pair_review_segment and _G._pair_review_segment() then
    return pair_compose_statusline(_G._pair_review_compact_left())
  end
  if vim.fn.mode():sub(1, 1) == 'n' then
    -- Carry the position marker (*, -N, +N) into locked/normal mode — the
    -- insert-mode cluster below is gone here, so without this the user loses
    -- the "you are here" cue the moment they leave insert to navigate
    -- (Alt+←/→ work in both modes). Same PairPosLabel green block as the
    -- cluster; pcall-guarded like it so a nav-state hiccup can't blank the bar.
    local ok_pos, pos = pcall(function()
      local p = pos_label(nav.pos)
      if is_dirty_history_slot() then p = p .. '*' end
      return p
    end)
    local pos_seg = (ok_pos and pos ~= '')
      and string.format(' %%#PairPosLabel#%s%%*', pos) or ''
    return pair_compose_statusline(
      pos_seg .. '%#PairLocked# <LOCKED> input not accepted — press i to type %*'
    )
  end
  local ok, result = pcall(function()
    local h = #read_history()
    local q = queue_count()
    local pos = pos_label(nav.pos)
    if is_dirty_history_slot() then pos = pos .. '*' end
    -- Wrap the actionable glyph (q or ⌫) inside its hint with PairAltKey
    -- so the keypress is visible — the rest of the bracket text stays
    -- in the muted statusline baseline.
    local hint
    if type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
      hint = ' [%#PairAltKey#⌫%*=del]'
    else
      hint = ' [%#PairAltKey#q%*=queue]'
    end
    local pos_seg = string.format('%%#PairPosLabel#%s%%*', pos)
    local q_hl = (q > 0) and 'PairQueueCount' or 'PairQueueZero'
    local q_seg = string.format('%%#%s#%d queued%%*', q_hl, q)
    -- "Alt:", "<-", "->" — the cluster label and the nav-arrow keys —
    -- get the same actionable accent so the user can see at a glance
    -- which glyphs are pressable. Inline highlight markers in the format
    -- string need %% to survive string.format's own % handling.
    return string.format(
      ' %%#PairAltKey#Alt:%%* %%#PairAltKey#<-%%* history %d < %s%s > %s %%#PairAltKey#->%%* ',
      h, pos_seg, hint, q_seg
    )
  end)
  return pair_compose_statusline(ok and result or ' pair ')
end

-- Persist any pending edit on a mutable slot to its underlying file. For
-- -N, only comment-only edits are persisted in place (the agent never sees
-- comments, so changing them isn't a fork). A real fork — anything that
-- would change the stripped body — is left unsaved so the next go_to can
-- raise the leave-dirty-history prompt.
local function autosave_current_slot()
  if nav.pos == '*' then
    pcall(vim.cmd, 'silent! write')
  elseif type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    if key then queue_write(key, buffer_text()) end
  elseif type(nav.pos) == 'table' and nav.pos.kind == 'history' then
    local body = buffer_text()
    if body ~= nav.baseline
       and strip_comments(body) == strip_comments(nav.baseline) then
      write_history_entry(nav.pos.n, body)
      nav.baseline = body
    end
  end
end

-- Send the current buffer to the agent and return to *, preserving *'s
-- persistent draft. Used only by the dirty-`-N` prompt's Send branch
-- (send_and_clear has its own variant that handles the from-* case).
local function ship_buffer_and_reset(body)
  -- Mirror send_and_clear: log full body, send stripped. Skip both if the
  -- stripped result is empty so a comment-only fork doesn't pollute the log.
  local stripped = strip_comments(body)
  local actually_sent = not stripped:match('^%s*$')
  if actually_sent then
    append_log(body)
    send_to_agent(stripped)
  end
  nav.pos = '*'
  if actually_sent then
    -- Sent body's === lines become the new sticky set for * (mirrors
    -- send_and_clear). Comment-only no-op preserves * as-is.
    local lines = apply_sticky_to_star(body, false)
    vim.api.nvim_buf_set_lines(0, 0, -1, false, lines)
    pcall(vim.api.nvim_win_set_cursor, 0, { #lines, 0 })
    vim.cmd('silent! write')
    nav.baseline = table.concat(lines, '\n')
  else
    set_buffer_text(read_file(draft_path_for_tag()))
    nav.baseline = buffer_text()
  end
  refresh_statusline()
end

-- Prompt the user with the four-option "what now?" dialog when leaving a
-- dirty -N slot. Returns true if the caller should proceed with the original
-- navigation (i.e. user picked Discard), false otherwise (Send/Queue performed
-- the action and moved us to *; or Stay cancelled the nav).
--
-- Single-key prompt format:  (S)end, (Q)ueue, (D)iscard, [S]tay:
--   () marks the access key, [] marks the default. Send/Stay both start with
--   "S"; we resolve by binding S → Send (the user-typeable choice) and
--   Enter/ESC/anything-else → Stay (the safe default, including covering
--   accidental key presses).
local function leave_dirty_history()
  local body = buffer_text()
  vim.api.nvim_echo({
    { '(S)end, (Q)ueue, (D)iscard, [S]tay: ', 'Question' },
  }, false, {})
  local ok, c = pcall(vim.fn.getchar)
  -- Clear the prompt line so it doesn't linger under cmdheight=0.
  pcall(vim.api.nvim_echo, { { '' } }, false, {})
  if not ok then return false end

  local key = (type(c) == 'number') and vim.fn.nr2char(c) or tostring(c or '')
  key = key:lower()

  if key == 's' then
    ship_buffer_and_reset(body)
    return false
  elseif key == 'q' then
    queue_push_front(body)
    set_buffer_text(read_file(draft_path_for_tag()))
    nav.pos = '*'
    nav.baseline = buffer_text()
    refresh_statusline()
    flash_queue_count()
    return false
  elseif key == 'd' then
    return true
  else
    return false                       -- Stay (Enter, ESC, anything else)
  end
end

-- Move to a new position: save mutable slots, prompt on dirty -N, then load
-- the destination baseline. nav_left / nav_right just compute the target pos.
local function go_to(new_pos)
  if is_dirty_history_slot() then
    if not leave_dirty_history() then return end
  else
    autosave_current_slot()
  end
  nav.pos = new_pos
  set_buffer_text(load_baseline_for_current_pos())
  -- Re-read the baseline from the buffer so its representation matches
  -- buffer_text() exactly (set_buffer_text strips a trailing newline). This
  -- keeps the dirty check (`buffer_text() ~= nav.baseline`) honest.
  nav.baseline = buffer_text()
  refresh_statusline()
end

local function nav_left()
  if nav.pos == '*' then
    local entries = read_history()
    if #entries == 0 then return end
    go_to({ kind = 'history', n = 1 })
  elseif nav.pos.kind == 'history' then
    local entries = read_history()
    if nav.pos.n >= #entries then return end   -- clamp at oldest
    go_to({ kind = 'history', n = nav.pos.n + 1 })
  elseif nav.pos.kind == 'queue' then
    -- +N → +(N-1) → *
    if nav.pos.n <= 1 then
      go_to('*')
    else
      go_to({ kind = 'queue', n = nav.pos.n - 1 })
    end
  end
end

local function nav_right()
  if nav.pos == '*' then
    -- * → +1 if queue has items.
    if queue_count() == 0 then return end
    go_to({ kind = 'queue', n = 1 })
  elseif nav.pos.kind == 'history' then
    -- -N → -(N-1), with -1 → *.
    if nav.pos.n <= 1 then
      go_to('*')
    else
      go_to({ kind = 'history', n = nav.pos.n - 1 })
    end
  elseif nav.pos.kind == 'queue' then
    -- +N → +(N+1), clamp at queue size.
    local total = queue_count()
    if nav.pos.n >= total then return end
    go_to({ kind = 'queue', n = nav.pos.n + 1 })
  end
end

-- Boundary-jump: Shift+Alt+←/→ steps between exactly three landmarks —
-- -h (oldest history), * (draft), +q (back of queue). The newest-history
-- (-1) and queue-front (+1) edges are deliberately *not* stops: Alt+←/→
-- already walks one slot at a time, so Shift+Alt is the coarse "jump to the
-- far end / back to draft" gesture. An empty region contributes no landmark.
local function pos_rank(p)
  if p == '*' then return 0 end
  if p.kind == 'history' then return -p.n end   -- -1 ranks -1, -h ranks -h
  if p.kind == 'queue'   then return  p.n end   -- +1 ranks 1,  +q ranks q
  return 0
end

local function ordered_landmarks()
  local list = {}
  local h = #read_history()
  local q = queue_count()
  if h >= 1 then
    table.insert(list, { kind = 'history', n = h })            -- -h (leftmost)
  end
  table.insert(list, '*')
  if q >= 1 then
    table.insert(list, { kind = 'queue', n = q })             -- +q (rightmost)
  end
  return list
end

local function nav_boundary(direction)
  local landmarks = ordered_landmarks()
  local cur = pos_rank(nav.pos)
  if direction > 0 then
    for _, lm in ipairs(landmarks) do
      if pos_rank(lm) > cur then go_to(lm); return end
    end
  else
    for i = #landmarks, 1, -1 do
      if pos_rank(landmarks[i]) < cur then go_to(landmarks[i]); return end
    end
  end
end

-- Alt+BS — delete the current +N queue item without sending it. "Stay near":
-- after delete, items at +(N+1)..+M shift down by one, so the same +N slot
-- now displays what used to be next. Lets the user tap-tap to clean out a
-- run. If queue empties, fall back to *.
--
-- Confirmation defaults to Yes — light safeguard against Cmd+BS (delete-to-
-- line-start) being mistapped as Alt+BS. Enter/space/y all proceed; only 'n'
-- or Esc cancels, so tap-tap cleanup stays cheap.
local function delete_current_queue_item()
  if type(nav.pos) ~= 'table' or nav.pos.kind ~= 'queue' then return end
  local key = queue_key_for_n(nav.pos.n)
  if not key then return end
  vim.api.nvim_echo({
    { 'Delete this queue item? [Y]es, (n)o: ', 'WarningMsg' },
  }, false, {})
  local ok, c = pcall(vim.fn.getchar)
  pcall(vim.api.nvim_echo, { { '' } }, false, {})
  local function cancel() refresh_statusline(); return end
  if not ok then return cancel() end
  if c == 27 then return cancel() end -- Esc
  local ch = (type(c) == 'number') and vim.fn.nr2char(c) or tostring(c or '')
  if ch:lower() == 'n' then return cancel() end
  queue_remove(key)
  local total = queue_count()
  if total == 0 then
    nav.pos = '*'
  elseif nav.pos.n > total then
    nav.pos = { kind = 'queue', n = total }
  end
  -- nav.pos.n unchanged when there's still something at this slot — the
  -- shifted-down item takes its place.
  set_buffer_text(load_baseline_for_current_pos())
  nav.baseline = buffer_text()
  refresh_statusline()
  flash_queue_count()
end

-- Shift+Alt+BS — wipe history, draft, and queue. "Start anew" for a session
-- whose state has accumulated cruft. Hard delete (no archive) per design;
-- confirmation defaults to No so a stray tap can't nuke a session.
local function forget_all()
  vim.api.nvim_echo({
    { 'Erase history, draft, and queue? (y)es, [N]o: ', 'WarningMsg' },
  }, false, {})
  local ok, c = pcall(vim.fn.getchar)
  pcall(vim.api.nvim_echo, { { '' } }, false, {})
  if not ok then return end
  local key = (type(c) == 'number') and vim.fn.nr2char(c) or tostring(c or '')
  if key:lower() ~= 'y' then return end

  os.remove(log_path_for_tag())
  os.remove(draft_path_for_tag())
  for _, k in ipairs(queue_keys_sorted()) do
    queue_remove(k)
  end

  nav.pos = '*'
  vim.api.nvim_buf_set_lines(0, 0, -1, false, { '' })
  vim.cmd('silent! write')
  nav.baseline = ''
  refresh_statusline()
  vim.cmd('startinsert')
end

-- Alt+q — push current buffer to the FRONT of the queue (+1), return to *.
-- Uniform rule across source slots:
--   * source : "park this draft for later". Clear * after the push.
--   -N source: fork the (possibly edited) history entry into +1. * untouched.
--   +N source: move-to-front. Remove the source +N file before pushing, so the
--              same item ends up at +1 (with edits applied). * untouched.
local function queue_current()
  local body = buffer_text()
  if body:match('^%s*$') then return end

  if type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    if key then queue_remove(key) end
  end

  queue_push_front(body)

  -- Preserve the `===` sticky comments into * exactly like a send does (the
  -- queued item keeps its own comments verbatim). Without this, parking a
  -- draft from * would wipe the sticky header. apply_sticky_to_star reads *'s
  -- on-disk WIP when queuing from -N/+N, or treats * as empty when queuing
  -- from * itself.
  local was_at_star = (nav.pos == '*')
  nav.pos = '*'
  local lines = apply_sticky_to_star(body, was_at_star)
  vim.api.nvim_buf_set_lines(0, 0, -1, false, lines)
  pcall(vim.api.nvim_win_set_cursor, 0, { #lines, 0 })
  -- Persist via :w; shortmess+=W keeps the cmdline silent so the statusline
  -- doesn't get pushed off.
  vim.cmd('silent! write')

  nav.baseline = table.concat(lines, '\n')
  refresh_statusline()
  flash_queue_count()
  vim.cmd('startinsert')
end

vim.opt.laststatus = 2
vim.opt.statusline = '%!v:lua.PairStatusline()'

-- Soften the statusline appearance. nvim's default StatusLine highlight is
-- inverted/bold (looks like a stark contrasting bar), which feels out of
-- place against the editing buffer. Linking it to `Comment` picks up the
-- colorscheme's dimmed-text style — visible as "secondary info" without
-- being visually loud. ColorScheme autocmd reapplies on theme changes.
local function pair_apply_statusline_hl()
  vim.api.nvim_set_hl(0, 'StatusLine',   { link = 'Comment' })
  vim.api.nvim_set_hl(0, 'StatusLineNC', { link = 'Comment' })
  -- Pop the queued-count above the muted baseline when the queue is non-empty
  -- — it's the only segment that means "you have pending work to send."
  -- PairQueueZero matches the muted baseline normally, but exists as its own
  -- group so flash_queue_count can light it up on the N→0 transition too.
  vim.api.nvim_set_hl(0, 'PairQueueCount', { link = 'WarningMsg' })
  vim.api.nvim_set_hl(0, 'PairQueueZero',  { link = 'Comment' })
  -- Locked-mode banner reads at the same muted level as the rest of the
  -- statusline — the bg tint is the loud signal; the text just labels it.
  vim.api.nvim_set_hl(0, 'PairLocked', { link = 'Comment' })
  -- Pop the position marker (*, -N, +N) above the muted baseline so the
  -- "you are here" cue is unmistakable. Identifier is typically a cool
  -- accent (blue/cyan) — distinct from PairQueueCount's warning hue.
  -- Borrow DiffAdd's color so the position marker reads as a green "you
  -- are here" block. Many themes encode DiffAdd as fg=green + reverse so
  -- the rendered look is dark text on a green bg. `link` + extra attrs
  -- doesn't layer (link wins, attrs drop), so resolve and copy.
  local diffadd = vim.api.nvim_get_hl(0, { name = 'DiffAdd', link = false })
  vim.api.nvim_set_hl(0, 'PairPosLabel', {
    fg      = diffadd.fg,
    bg      = diffadd.bg,
    ctermfg = diffadd.ctermfg,
    ctermbg = diffadd.ctermbg,
    reverse = diffadd.reverse,
    bold    = true,
  })
  -- "Things the user can press" — the Alt: label, the <- / -> nav arrows,
  -- and the inline action glyphs (q, ⌫) inside the contextual hint. Lift
  -- them out of the muted Comment baseline with Special's accent + bold
  -- so they read as actionable at a glance. `link` + bold doesn't layer
  -- (link wins, bold drops), so resolve Special's fg and set explicitly.
  local special = vim.api.nvim_get_hl(0, { name = 'Special', link = false })
  vim.api.nvim_set_hl(0, 'PairAltKey', {
    fg      = special.fg,
    ctermfg = special.ctermfg,
    bold    = true,
  })
  -- The pinned `===` header in the winbar gets the diff "added" green so it
  -- reads as a distinct "this is the header" band, set apart from the grey
  -- compose text scrolling below it. We want green *text* across the bar (not
  -- a full-width green block), so borrow the foreground of the `Added` group —
  -- the same green a diff paints new lines with. (DiffAdd.fg is the *contrast*
  -- color sitting on a green bg — white in slate — so it's the wrong source
  -- for a fg-only treatment; Added carries the green in its fg directly.)
  local added = vim.api.nvim_get_hl(0, { name = 'Added', link = false })
  vim.api.nvim_set_hl(0, 'PairWinbar', {
    fg      = added.fg,
    ctermfg = added.ctermfg,
    bold    = true,
  })
end
pair_apply_statusline_hl()

-- ---------------------------------------------------------------------------
-- mode-tinted background — make non-insert modes visually obvious so a stray
-- middle-click paste / copy-on-select that lands in normal mode is caught
-- before keys get interpreted as commands instead of text.
-- ---------------------------------------------------------------------------
-- Insert mode keeps the colorscheme look. Locked mode (everything else)
-- swaps to a "dimmed sheet": lifted grey bg + every syntax color blended
-- toward that bg by FADE_ALPHA so the palette survives in muted form.
-- Applied via a highlight namespace so the colorscheme's default ns is
-- left untouched. Truecolor (termguicolors) is required and asserted at
-- top of file.
-- Two fade levels:
--   pair_locked_ns  — focus-lost AND not in insert mode (deeper grey-out at
--                     0.45). The pane isn't where the user is looking and
--                     isn't even editable, so push it well into the bg.
--   pair_normal_ns  — light grey-out at 0.80. Applied in two cases:
--                     (1) focused + normal mode: user is looking, fade says
--                         "not actively typing." (2) unfocused + insert mode:
--                         text is still going to flow in here (copy-on-
--                         select destination), keep it legible while signaling
--                         "not the foreground focus."
-- Focused + insert uses neither (no fade).
local pair_locked_ns = vim.api.nvim_create_namespace('pair_locked')
local pair_normal_ns = vim.api.nvim_create_namespace('pair_normal')

-- 0 = pure locked-bg (full grey-out); 1 = original color (no fade).
local PAIR_LOCKED_FADE = 0.45  -- focus-lost: pane is in the user's periphery
local PAIR_NORMAL_FADE = 0.80  -- focused normal mode: barely faded

-- Fallback fg used when a group has no defined fg (e.g. Normal under the
-- default scheme — its fg comes from the terminal default, which our blend
-- can't see). Without this, "plain" buffer text stays at the terminal's
-- bright default while every colored token gets faded, leaving Normal
-- text the *brightest* thing on the dimmed sheet.
local PAIR_LOCKED_DEFAULT_FG = 0xeeeeee

-- Generic 24-bit RGB blend. alpha=1 → rgb_a, alpha=0 → rgb_b.
local function pair_blend_rgb(rgb_a, rgb_b, alpha)
  local ar = math.floor(rgb_a / 65536) % 256
  local ag = math.floor(rgb_a / 256) % 256
  local ab = rgb_a % 256
  local br = math.floor(rgb_b / 65536) % 256
  local bg = math.floor(rgb_b / 256) % 256
  local bb = rgb_b % 256
  local nr = math.floor(ar * alpha + br * (1 - alpha) + 0.5)
  local ng = math.floor(ag * alpha + bg * (1 - alpha) + 0.5)
  local nb = math.floor(ab * alpha + bb * (1 - alpha) + 0.5)
  return nr * 65536 + ng * 256 + nb
end

-- Resolve insert-mode and locked-mode background colors from the active
-- colorscheme's `Normal` group instead of hardcoding hex values, so the
-- dimmed sheet stays visibly distinct from the editing bg under any
-- theme (light or dark). Insert mode reuses Normal.bg verbatim; the
-- locked sheet is Normal.bg shifted 14% toward 50% grey — this direction
-- automatically lifts on dark themes and darkens on light themes (both
-- move toward neutral). Re-run on init and on every ColorScheme change.
local pair_bg_insert       -- '#xxxxxx' string for nvim_set_hl
local pair_bg_locked_int   -- 24-bit int, used as blend target

local function pair_resolve_bgs()
  local normal = vim.api.nvim_get_hl(0, { name = 'Normal', link = false }) or {}
  -- Fallback when Normal.bg is undefined (terminal-default themes): pick
  -- a dim neutral that the original hardcoded value resolved to.
  local theme_bg = normal.bg or 0x1c1c1c
  pair_bg_insert     = string.format('#%06x', theme_bg)
  pair_bg_locked_int = pair_blend_rgb(theme_bg, 0x808080, 0.86)
end
pair_resolve_bgs()

-- Linear blend between an integer RGB color (24-bit) and pair_bg_locked_int.
-- alpha = 1 returns rgb unchanged; alpha = 0 returns the locked-sheet bg.
local function pair_blend_to_locked(rgb, alpha)
  return pair_blend_rgb(rgb, pair_bg_locked_int, alpha)
end

-- Snapshot every currently-defined highlight group and clone it into a
-- target namespace with fg/bg blended toward the locked sheet bg, while
-- preserving decorations (bold/italic/underline) so the colorscheme's
-- shape stays recognizable. Rebuilt on ColorScheme so new schemes /
-- late-loaded tree-sitter groups get covered. Cursor groups are skipped
-- so the cursor block stays visible against the dimmed sheet.
--
-- `reverse` is dropped: it swaps fg/bg, which would put a faded color in
-- the bg slot and break the uniform sheet. Groups that relied on reverse
-- (CurSearch, Visual on some schemes) will fall back to the resolved
-- fg/bg without inversion — still legible, just not flipped.
local function pair_build_faded_ns(ns, alpha)
  for name in pairs(vim.api.nvim_get_hl(0, {})) do
    -- Skip cursor groups so the cursor block + the focus-lost
    -- PairFocusBlock indicator stay visible against the faded sheet.
    -- Without the PairFocusBlock exclusion, the blinking block ended
    -- up the same shade as its background and the animation was
    -- invisible in the unfocused-insert state.
    if name ~= 'Cursor' and name ~= 'lCursor' and name ~= 'TermCursor'
       and name ~= 'PairFocusBlock' then
      local hl = vim.api.nvim_get_hl(0, { name = name, link = false }) or {}
      local entry = {
        bold          = hl.bold,
        italic        = hl.italic,
        underline     = hl.underline,
        undercurl     = hl.undercurl,
        strikethrough = hl.strikethrough,
      }
      entry.fg = pair_blend_to_locked(hl.fg or PAIR_LOCKED_DEFAULT_FG, alpha)
      -- Group-specific bg (Visual, Search, etc.): fade toward locked bg
      -- so highlighters stay visible but muted. No bg → uniform sheet.
      if hl.bg then
        entry.bg = pair_blend_to_locked(hl.bg, alpha)
      else
        entry.bg = pair_bg_locked_int
      end
      vim.api.nvim_set_hl(ns, name, entry)
    end
  end
end
local function pair_build_fade_namespaces()
  pair_resolve_bgs() -- refresh in case colorscheme changed
  pair_build_faded_ns(pair_locked_ns, PAIR_LOCKED_FADE)
  pair_build_faded_ns(pair_normal_ns, PAIR_NORMAL_FADE)
end
pair_build_fade_namespaces()

-- Focus state mirror. Four sheet states:
--   focused   + insert  → no fade (full color)
--   focused   + normal  → light fade (pair_normal_ns at 0.80)
--   unfocused + insert  → light fade (pair_normal_ns at 0.80) — pane is a
--                         copy-on-select destination, text needs to stay
--                         readable while signaling "not the foreground"
--   unfocused + normal  → deep fade  (pair_locked_ns at 0.45)
-- Default true: nvim usually starts focused (and if it doesn't, the
-- first FocusLost will correct us).
local pair_has_focus = true

local function pair_apply_mode_bg(mode)
  local in_insert = (mode ~= 'n')
  if not pair_has_focus then
    pair_build_fade_namespaces() -- catch any groups defined since last build
    -- Insert-mode-unfocused promoted to the light fade so the user can
    -- read what's in the draft while attending to the other pane.
    vim.api.nvim_set_hl_ns(in_insert and pair_normal_ns or pair_locked_ns)
  elseif mode == 'n' then
    pair_build_fade_namespaces()
    vim.api.nvim_set_hl_ns(pair_normal_ns)
  else
    pair_resolve_bgs()
    vim.api.nvim_set_hl_ns(0)
    vim.api.nvim_set_hl(0, 'Normal',      { bg = pair_bg_insert })
    vim.api.nvim_set_hl(0, 'NormalNC',    { bg = pair_bg_insert })
    vim.api.nvim_set_hl(0, 'EndOfBuffer', { bg = pair_bg_insert })
  end
end
-- Coalesce + defer: read mode on the next event-loop tick so transient
-- mode flips (e.g. `:normal! zt` inside a Lua callback that ends with
-- `startinsert`) don't strand us in the locked namespace. Synchronous
-- ModeChanged would see the intermediate 'n', and the trailing startinsert
-- only takes effect after the current tick — so by the time our scheduled
-- callback runs, vim.fn.mode() reports the settled mode.
local pair_mode_bg_pending = false
local function pair_schedule_mode_bg()
  if pair_mode_bg_pending then return end
  pair_mode_bg_pending = true
  vim.schedule(function()
    pair_mode_bg_pending = false
    pair_apply_mode_bg(vim.fn.mode():sub(1, 1))
  end)
end
vim.api.nvim_create_autocmd('ModeChanged', {
  callback = pair_schedule_mode_bg,
})
vim.api.nvim_create_autocmd('FocusLost', {
  callback = function()
    pair_has_focus = false
    pair_schedule_mode_bg()
  end,
})
vim.api.nvim_create_autocmd('FocusGained', {
  callback = function()
    pair_has_focus = true
    pair_schedule_mode_bg()
  end,
})
vim.api.nvim_create_autocmd('ColorScheme', {
  callback = pair_build_fade_namespaces,
})

-- ---------------------------------------------------------------------------
-- Scrollback-comment pickup (#000018). When the user drops 🤖[] markers in
-- the Alt+/ scrollback viewer and quits, scrollback.lua writes a sidecar
-- block to `$DATA_DIR/scrollback-pending-<tag>.md`. On every FocusGained
-- the draft pane checks for this file and lands the content via one of
-- two paths depending on the current slot:
--
--   `*` slot     — append directly into the buffer via nvim_buf_set_lines
--                  + `:silent! write`. We don't go through autoread+
--                  checktime because file mtime resolution is sub-second
--                  and the change can land in the same tick the buffer
--                  was loaded, defeating the auto-reload trigger.
--   `-N` / `+N` — append to draft-<tag>.md on disk; the next nav-to-`*`
--                  reads from disk and picks it up naturally. No state
--                  needed in nvim — the file IS the queue.
--
-- The sidecar is removed only after a successful land (catch failed I/O
-- so we don't lose comments to a "picked up N" toast that's lying).
local function pair_pending_path()
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
  return data_dir .. '/scrollback-pending-' .. tag .. '.md'
end

local function pair_draft_file()
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
  return data_dir .. '/draft-' .. tag .. '.md'
end

local function pair_pickup_scrollback_pending()
  local pending = pair_pending_path()
  local f = io.open(pending, 'r')
  if not f then return end
  local content = f:read('*a')
  f:close()
  if not content or content == '' then
    os.remove(pending)
    return
  end

  local count = 0
  for _ in content:gmatch('\n> ') do count = count + 1 end

  local landed = false
  if nav.pos == '*' then
    -- On the draft slot: append directly into the buffer + autosave.
    local lines = vim.split(content, '\n', { plain = true })
    while #lines > 0 and lines[#lines] == '' do
      table.remove(lines)  -- trim trailing empty(s) from the final \n
    end
    -- The sidecar always begins with a leading empty as a separator.
    -- Drop it when the buffer's last line is also empty so we don't
    -- end up with a double blank.
    local last_existing = vim.api.nvim_buf_get_lines(0, -2, -1, false)[1] or ''
    if last_existing == '' and lines[1] == '' then
      table.remove(lines, 1)
    end
    local block_start = vim.api.nvim_buf_line_count(0)  -- 0-idx, first inserted
    local ok = pcall(vim.api.nvim_buf_set_lines, 0, -1, -1, false, lines)
    if ok then
      pcall(vim.cmd, 'silent! write')
      -- Visual feedback identical to copy-on-select's quote-paste:
      -- scroll the first non-empty inserted line to the top of the
      -- draft pane, then flash every inserted line via the shared
      -- pair_flash_ns + IncSearch hl. 500ms clear matches the rest of
      -- pair's flash idiom (queue-count blink, paste highlight). Skip
      -- startinsert — the user may have come back to read, not type.
      local first_visible = block_start
      for i, l in ipairs(lines) do
        if l ~= '' then first_visible = block_start + i - 1; break end
      end
      pcall(vim.api.nvim_win_set_cursor, 0, { first_visible + 1, 0 })
      pcall(vim.cmd, 'normal! zt')
      vim.api.nvim_buf_clear_namespace(0, pair_flash_ns, 0, -1)
      for i = block_start, block_start + #lines - 1 do
        vim.api.nvim_buf_add_highlight(0, pair_flash_ns, 'IncSearch', i, 0, -1)
      end
      clear_flash_after(0, 500)
      landed = true
    end
  else
    -- Not on `*`: append to the draft FILE so the next navigation back
    -- to the draft slot reads it. nvim's slot-load reads from disk, so
    -- this is the right hand-off shape for the off-slot case.
    local draft = pair_draft_file()
    local d = io.open(draft, 'a')
    if d then
      local wrote = pcall(function() d:write(content); d:close() end)
      if wrote then landed = true end
    end
  end

  if landed then
    os.remove(pending)
    vim.notify(string.format('🤖 picked up %d scrollback comment(s)', count),
               vim.log.levels.INFO)
  else
    -- Preserve sidecar so the next FocusGained tries again rather than
    -- silently swallowing the user's comments.
    vim.notify('🤖 pickup failed; sidecar kept at ' .. pending,
               vim.log.levels.WARN)
  end
end

vim.api.nvim_create_autocmd('FocusGained', {
  callback = function() pcall(pair_pickup_scrollback_pending) end,
})

-- Backstop the FocusGained pickup with a libuv fs_event watcher. When
-- scrollback.lua writes the sidecar in VimLeavePre, zellij doesn't
-- always forward a focus event back to the draft pane fast enough
-- (sometimes 5–10s of delay observed when closing a floating pane),
-- so markers appear to "vanish" until the user happens to mouse or
-- keypress into the draft. The watcher fires within ms of the rename
-- and calls the same pickup function, single source of truth for the
-- landing logic.
local function pair_start_pending_fs_watch()
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
  local target = 'scrollback-pending-' .. tag .. '.md'
  local handle = vim.loop.new_fs_event()
  if not handle then return end
  local ok = pcall(function()
    handle:start(data_dir, {}, vim.schedule_wrap(function(err, filename)
      if err then return end
      -- Many other files churn in $PAIR_DATA_DIR (agent-output, draft,
      -- config…); skip events that name a different file. Where the
      -- platform doesn't report a filename (filename == nil), fall
      -- through and let the pickup short-circuit on its own existence
      -- check.
      if filename and filename ~= target then return end
      pcall(pair_pickup_scrollback_pending)
    end))
  end)
  if not ok then handle:close() end
end
pair_start_pending_fs_watch()

-- Watch $PAIR_DATA_DIR for the change-log "build complete" marker (#58). The
-- detached distiller drops "changelog-<tag>-<agent>.ready" only when a triggered
-- build actually changed the log; we flash the statusline + delete the marker
-- (one-shot). A low-frequency timer poll, NOT fs_event: macOS FSEvents is
-- unreliable from nvim here (it surfaces EMFILE with a nil filename), and the
-- scrollback-pending fs_event watcher only gets away with it because a
-- FocusGained fallback covers the miss. This signal has no such fallback — its
-- whole job is to flash while the operator works in the *agent* pane (the draft
-- statusline stays on screen), so it can't depend on focus. One fs_stat every
-- 2s is negligible; the ≤2s latency is invisible against a slow background build.
-- Resolve the change-log session id (#63): the env var bin/pair exports when the
-- id is known at launch (claude-fresh / any resume), else the per-tag config the
-- session watcher writes (codex/agy discover it async). Mirrors the env->config
-- order in bin/pair-changelog-open so the polled .ready path matches the base the
-- opener builds. A focused reader, not pair_read_saved_config() -- that one is
-- defined later in this file (Lua local-function ordering) and also reads the
-- agent-<tag> file, which is overkill here.
local function pair_changelog_session_id(data_dir, tag, agent)
  local sid = vim.env.PAIR_SESSION_ID
  if sid and sid ~= '' then return sid end
  local cf = io.open(data_dir .. '/config-' .. tag .. '-' .. agent .. '.json', 'r')
  if not cf then return nil end
  local body = cf:read('*a'); cf:close()
  local ok, parsed = pcall(vim.json.decode, body)
  if ok and type(parsed) == 'table' and parsed.session_id and parsed.session_id ~= '' then
    return parsed.session_id
  end
  return nil
end

local function pair_start_changelog_ready_watch()
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
  local agent = vim.env.PAIR_AGENT or 'claude'
  vim.fn.timer_start(2000, function()
    -- Re-resolve each tick: a codex/agy id may land in the config mid-session.
    local sid = pair_changelog_session_id(data_dir, tag, agent)
    local base = data_dir .. '/changelog-' .. tag .. '-' .. agent
    if sid then base = base .. '-' .. sid end
    local marker = base .. '.ready'
    if not vim.loop.fs_stat(marker) then return end
    os.remove(marker) -- one-shot: consume the marker so the flash fires once
    pair_flash_notify('✓ change log ready · Alt+l')
  end, { ['repeat'] = -1 })
end
pair_start_changelog_ready_watch()

pair_apply_mode_bg(vim.fn.mode():sub(1, 1))

-- ---------------------------------------------------------------------------
-- quit-blocker — fat-finger guard for muscle-memory :wq / :q / ZZ etc.
-- ---------------------------------------------------------------------------
-- This nvim instance is the pair draft pane, not a standalone editor. The
-- correct exits are Alt+x (full quit) or Alt+d (detach); a stray :wq would
-- kill the draft pane mid-session and orphan zellij's layout. We rewrite
-- the common quit verbs as a no-op that echoes the right path. Saves still
-- happen via autosave, so swallowing the `:w` part of `:wq` costs nothing.
function _G.PairQuitWarn()
  vim.api.nvim_echo({
    { 'pair: ', 'Question' },
    { 'use Alt+x to quit, or Alt+d to detach', 'WarningMsg' },
  }, false, {})
end

-- Match the WHOLE typed command exactly (cmdline ==# 'q' etc.) so this only
-- fires for bare quits, not e.g. `:qall` typed character-by-character or a
-- substitute pattern that happens to contain 'q'. The `<expr>` form lets us
-- branch on getcmdtype() so command-mode-only triggers fire.
local quit_verbs = {
  'q', 'q!', 'wq', 'wq!', 'quit', 'quit!',
  'qa', 'qa!', 'qall', 'qall!',
  'wqa', 'wqa!', 'wqall', 'wqall!',
  'x', 'x!', 'xa', 'xa!', 'xall', 'xall!',
  'exit', 'exit!',
}
for _, v in ipairs(quit_verbs) do
  vim.cmd(string.format(
    [[cnoreabbrev <expr> %s getcmdtype() == ':' && getcmdline() ==# %q ? 'lua PairQuitWarn()' : %q]],
    v, v, v
  ))
end

-- Normal-mode shortcuts that bypass the cmdline (and thus the abbreviations).
vim.keymap.set('n', 'ZZ', function() PairQuitWarn() end, { silent = true, desc = 'pair: quit blocked' })
vim.keymap.set('n', 'ZQ', function() PairQuitWarn() end, { silent = true, desc = 'pair: quit blocked' })

-- ---------------------------------------------------------------------------
-- Alt+x / Alt+d confirm prompts
-- ---------------------------------------------------------------------------
-- The zellij keybindings for Alt+x (full quit) and Alt+d (detach) route
-- here instead of running the action directly. Both are easy to fat-finger
-- — Alt+x is unrecoverable (kills the zellij session and its processes)
-- and Alt+d drops the user out of a long-running attached session. The
-- zellij side moves focus to nvim, ESCs into normal mode, and runs one of
-- these via cmdline; vim.fn.confirm pops a modal Y/N (default No), and the
-- action only fires on Yes. Y is shelled out via vim.fn.system because
-- nvim has no direct zellij IPC and re-binding zellij keybindings to first
-- check a flag is more state than this is worth.
-- If the user fires a confirm-requiring keybind while the rung is
-- minimized, the modal prompt would land on a 1-row pane where nothing
-- is visible. Step up to small first so the prompt renders, then defer
-- the actual prompt one event-loop tick — zellij's resize after
-- swap-layout reaches nvim asynchronously, and vim.fn.confirm reads
-- window dimensions when it's called.
local function pair_ensure_visible_then(fn)
  if pair_layout_state == 'minimized' and _G.PairLayoutBigger then
    _G.PairLayoutBigger()
    vim.defer_fn(fn, 100)
  else
    fn()
  end
end

-- Compact "Nu" duration: `45s` `12m` `3.2h` `5d`. Used in the confirm
-- modals so the session-id line carries a "this session is X old, last
-- touched Y ago" hint without ballooning into a sentence.
local function humanize_dur(secs)
  if secs < 0 then secs = 0 end
  if secs < 60 then return string.format('%ds', secs) end
  if secs < 3600 then return string.format('%dm', math.floor(secs / 60)) end
  if secs < 86400 then return string.format('%.1fh', secs / 3600) end
  return string.format('%.1fd', secs / 86400)
end

-- Resolve the on-disk session file for (agent, sid) and return
-- "(<age> old, <idle> idle)" — or nil if the file can't be found
-- (uncaptured id, agent we don't have a path resolver for, etc.).
-- Only called from the confirm modals, so the cost (one stat for
-- claude; a find for codex) is paid at most once
-- per Alt+x / Alt+n press.
local function session_age_hint(agent, sid)
  if not sid or sid == '' then return nil end
  local home = vim.env.HOME or ''
  local path
  if agent == 'claude' then
    local cwd = vim.env.PWD or vim.fn.getcwd()
    local enc = cwd:gsub('[./]', '-')
    path = home .. '/.claude/projects/' .. enc .. '/' .. sid .. '.jsonl'
    if vim.fn.filereadable(path) ~= 1 then path = nil end
  elseif agent == 'codex' then
    local cmd = 'find ' .. vim.fn.shellescape(home .. '/.codex/sessions')
      .. " -type f -name '*" .. sid .. "*.jsonl' 2>/dev/null | head -1"
    local h = io.popen(cmd)
    if h then path = h:read('*l'); h:close() end
  end
  if not path or path == '' then return nil end
  local h = io.popen('stat -f "%B %m" ' .. vim.fn.shellescape(path) .. ' 2>/dev/null')
  if not h then return nil end
  local out = h:read('*l')
  h:close()
  if not out then return nil end
  local birth, mtime = out:match('^(%d+) (%d+)$')
  if not birth then return nil end
  local now = os.time()
  return string.format('(%s old, %s idle)',
                       humanize_dur(now - tonumber(birth)),
                       humanize_dur(now - tonumber(mtime)))
end

-- Read the per-(tag,agent) saved config so the Alt+x prompt can show the
-- user what they're about to detach from for the future `pair resume
-- <tag>` path. Returns nil when the tag isn't set, the agent file is
-- missing, or the JSON parse fails — in which case the prompt falls back
-- to the bare confirmation.
local function pair_read_saved_config()
  local tag = vim.env.PAIR_TAG
  if not tag or tag == '' then return nil end
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')

  local af = io.open(data_dir .. '/agent-' .. tag, 'r')
  if not af then return nil end
  local agent = af:read('*l')
  af:close()
  if not agent or agent == '' then return nil end

  local cfg = { tag = tag, agent = agent }
  local cf = io.open(data_dir .. '/config-' .. tag .. '-' .. agent .. '.json', 'r')
  if cf then
    local body = cf:read('*a')
    cf:close()
    local ok, parsed = pcall(vim.json.decode, body)
    if ok and type(parsed) == 'table' then
      cfg.args       = parsed.args
      cfg.session_id = parsed.session_id
    end
  end
  return cfg
end

function _G.PairConfirmQuit()
  pair_ensure_visible_then(function()
    local prompt = 'Quit pair session? This kills the session and all its processes.'
    local cfg = pair_read_saved_config()
    if cfg then
      local args_line
      if type(cfg.args) == 'table' and #cfg.args > 0 then
        args_line = table.concat(cfg.args, ' ')
      else
        args_line = '<none>'
      end
      local sid_line = cfg.session_id and cfg.session_id ~= '' and cfg.session_id or '<not captured>'
      local age = session_age_hint(cfg.agent, cfg.session_id)
      if age then sid_line = sid_line .. '  ' .. age end
      prompt = prompt
        .. '\n\nResumable later via `pair resume ' .. cfg.tag .. '`:'
        .. '\n  agent:      ' .. cfg.agent
        .. '\n  args:       ' .. args_line
        .. '\n  session id: ' .. sid_line
    end
    local ans = vim.fn.confirm(prompt, '&Yes\n&No', 2)
    if ans == 1 then
      vim.fn.system({ 'pair', 'quit' })
    end
  end)
end

function _G.PairConfirmDetach()
  pair_ensure_visible_then(function()
    local ans = vim.fn.confirm('Detach from this pair session?', '&Yes\n&No', 2)
    if ans == 1 then
      if has_ui() then
        vim.fn.system({ 'zellij', 'action', 'detach' })
      end
    end
  end)
end

-- Shared between Alt+n (PairConfirmRestart) and Shift+Alt+N
-- (PairConfirmRestartNewSession). Differs in whether `pair restart`
-- is invoked with --new-session and what the prompt says.
--
--   Alt+n         — pure pair reload; agent session is preserved
--                   (resumed via --resume <id> / resume <id>).
--   Shift+Alt+N   — same tag + agent + args, but a fresh agent
--                   conversation (saved config is dropped).
--
-- Both confirms offer an extra (R)ename option (#000022 M2): the
-- restart choreography becomes kill → `pair rename <old> <new>` →
-- re-exec with PAIR_FORCE_TAG=<new>, so the agent conversation
-- (resume) or fresh session (new-session) is preserved under the
-- new tag. The rename runs in handle_restart_marker after the kill;
-- pre-validation here via `pair rename --restart-check` so the user
-- gets immediate feedback on bad tags / collisions instead of
-- discovering them post-kill.
local function pair_rename_prompt(current_tag)
  -- Loop until the user enters a valid new tag (returned) or cancels
  -- with empty input / Esc (returns nil).
  while true do
    -- Pre-fill with current_tag so the user can edit in place rather
    -- than retype the prefix of a related name.
    local input = vim.fn.input({
      prompt = 'New tag: ',
      default = current_tag,
      cancelreturn = '',
    })
    if input == nil or input == '' then return nil end
    -- Strip optional pair- prefix to match `pair resume` / `pair rename`
    -- accepting either form.
    local new_tag = input:gsub('^pair%-', '')
    if new_tag == current_tag then
      vim.api.nvim_echo({ { '\nnew tag matches current tag; nothing to do', 'WarningMsg' } }, false, {})
      return nil
    end
    if not new_tag:match('^[%w_-]+$') then
      vim.api.nvim_echo({ { '\ninvalid tag (allowed: letters, digits, dash, underscore) — try again', 'WarningMsg' } }, false, {})
    else
      local out = vim.fn.system({ 'pair', 'rename', '--restart-check', current_tag, new_tag })
      if vim.v.shell_error == 0 then
        return new_tag
      end
      vim.api.nvim_echo({ { '\n' .. (out:gsub('%s+$', '')) .. ' — try again', 'WarningMsg' } }, false, {})
    end
  end
end

local function pair_confirm_restart_impl(new_session)
  pair_ensure_visible_then(function()
    local prompt
    if new_session then
      prompt = 'Continue work with a brand new session?'
    else
      prompt = 'Reload pair?'
    end
    local cfg = pair_read_saved_config()
    if cfg then
      local args_line
      if type(cfg.args) == 'table' and #cfg.args > 0 then
        args_line = table.concat(cfg.args, ' ')
      else
        args_line = '<none>'
      end
      prompt = prompt
        .. '\n\nRe-launching with:'
        .. '\n  agent: ' .. cfg.agent
        .. '\n  args:  ' .. args_line
      -- Show the session id only on the resume path — it's the load-
      -- bearing detail there. Hiding it on the new-session path avoids
      -- confusing the user into thinking the prior id will carry over.
      if not new_session and cfg.session_id and cfg.session_id ~= '' then
        local resume_line = cfg.session_id
        local age = session_age_hint(cfg.agent, cfg.session_id)
        if age then resume_line = resume_line .. '  ' .. age end
        prompt = prompt .. '\n  resume: ' .. resume_line
      end
    end
    local ans = vim.fn.confirm(prompt, '&Yes\n&No\n&Rename', 2)
    if ans ~= 1 and ans ~= 3 then return end

    local rename_to
    if ans == 3 then
      rename_to = pair_rename_prompt(pair_tag())
      if not rename_to then return end
    end

    local argv = { 'pair', 'restart' }
    if new_session then table.insert(argv, '--new-session') end
    if rename_to then
      table.insert(argv, '--rename-to')
      table.insert(argv, rename_to)
    end
    vim.fn.system(argv)
  end)
end

function _G.PairConfirmRestart()           pair_confirm_restart_impl(false) end
function _G.PairConfirmRestartNewSession() pair_confirm_restart_impl(true)  end

-- Alt+Shift+C compaction (#55). Unlike the restart modals (which invoke
-- `pair restart` directly), creating a continuation needs the agent's
-- judgment — so this asks the AGENT (agent-agnostic prompt, no claude-only
-- skill name) to distill a continuation and then run `pair continue <slug>`,
-- which is context-aware: inside this live pane it parks the scrollback, marks
-- a continue= restart, and kills the session → the outer pair reincarnates the
-- same tag with a fresh conversation seeded from the doc.
--
-- The prompt DEFERS to the project's continuation DATATYPE procedure rather than
-- enumerating a section skeleton inline. The skeleton + authoring steps live in
-- the datatype (construct/datatype/continuation.md — ariadne#105: flush-first,
-- thread-arc/user-model, open-questions, lessons); an inline copy here would
-- drift out of sync with it — that drift WAS the bug pair#61 fixed, so do not
-- re-add a skeleton list. Keep it agent-agnostic: no skill name, no hardcoded path.
local COMPACT_PROMPT = table.concat({
  'Compact this session:',
  "1. Write a continuation doc for this session NOW by following this project's",
  '   continuation DATATYPE procedure — first flush key exchanges to pensive,',
  '   then distill per that procedure and finalize with `pair continuation`',
  '   writer (workshop/continuation/). Choose a short slug.',
  '2. Then run:  pair continue <that-slug>',
  '   (or  pair-dev continue <that-slug>  if this is a dev checkout)',
  '   That restarts this session with a fresh conversation seeded from the',
  '   continuation. Do this as your immediate next action.',
}, '\n')

function _G.PairConfirmCompact()
  pair_ensure_visible_then(function()
    local ans = vim.fn.confirm(
      'Compact this session?\n\nThe agent will write a continuation, then'
        .. '\n`pair continue <slug>` restarts this tag fresh, seeded from it.',
      '&Yes\n&No', 2)
    if ans ~= 1 then return end
    send_to_agent(COMPACT_PROMPT)
  end)
end

-- :PairTTYRawPath / _G.PairTTYRawPath() — print THIS session's raw scrollback
-- path (the VT byte stream pair-wrap --scrollback-log captures; the substrate
-- `pair scrollback render` replays). It lives in the XDG data dir, NOT the repo,
-- and is RAW bytes, not cleaned text. Grab it mid-session while the file is
-- live — at Alt+x quit it's deleted unless preserved (see cleanup_quit_marker),
-- and the next same-tag launch O_TRUNCs it. The path is copied to the + register.
function _G.PairTTYRawPath()
  -- Raw getenv on purpose (NOT pair_tag(), whose 'claude' fallback would mask
  -- the unset case) so the guard below can report "not inside a pair session".
  local tag = os.getenv('PAIR_TAG')
  local agent = os.getenv('PAIR_AGENT')
  if not tag or tag == '' or not agent or agent == '' then
    vim.notify('pair: not inside a pair session (PAIR_TAG/PAIR_AGENT unset)', vim.log.levels.WARN)
    return
  end
  local raw = string.format('%s/scrollback-%s-%s.raw', pair_data_dir(), tag, agent)
  local sz = vim.fn.getfsize(raw)
  local note = (sz >= 0) and string.format(' (%d bytes)', sz) or ' (not present yet)'
  local copied = pcall(vim.fn.setreg, '+', raw)   -- may fail with no clipboard provider
  vim.notify('pair tty raw: ' .. raw .. note .. (copied and '  [copied to +]' or ''), vim.log.levels.INFO)
end
vim.api.nvim_create_user_command('PairTTYRawPath', function() _G.PairTTYRawPath() end, {})

-- ---------------------------------------------------------------------------
-- Layout sizing: minimized (statusline only) ↔ small (12 rows, initial) ↔ third (1/3).
-- ---------------------------------------------------------------------------
-- Two keys drive this: Alt+Up (PairLayoutBigger) and Alt+Down
-- (PairLayoutSmaller) step along the ladder, clamped at the ends.
--
-- Sizing is exact — zellij/layouts/main.kdl declares each rung as a
-- swap_tiled_layout with the desired draft-pane size. We step along the
-- ladder via `zellij action next-swap-layout` / `previous-swap-layout`,
-- which re-tiles the existing agent + nvim panes onto the target swap
-- layout. One IPC call per step, panes are preserved.
--
-- Cycle from default = [minimized, third]:
--   default(small) → next → minimized → next → third → next → wraps
--   default(small) → prev → third → prev → minimized → prev → wraps
-- So Alt+Down (smaller) maps to next-swap from {small, third},
-- and Alt+Up (bigger) maps to prev-swap from {small, minimized}.
-- The state machine in PairLayoutBigger / PairLayoutSmaller clamps at
-- the rung extremes so we never wrap past them.
local LAYOUT_STATE_FILE = (vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share'))
  .. '/pair/layout-mode-' .. (vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude')

-- Read the current rung from nvim's own pane height. The kdl pins each
-- rung to an exact size (1 / 12 / 33%), so vim.o.lines is a ground-truth
-- signal that can't drift from zellij's actual swap-layout position. We
-- previously tracked rung in a disk file and updated it after each step,
-- which permanently desynced if any single `zellij action <swap>` was
-- silently rejected — the clamp in PairLayoutBigger/Smaller would then
-- lock the user out of one end of the ladder. Reading actual height
-- means each press recomputes from reality and is self-correcting.
local function layout_read()
  local h = vim.o.lines
  if h <= 2 then return 'minimized' end
  if h <= 12 then return 'small' end
  return 'third'
end

local function layout_write(s)
  -- Mirrors the rung into pair_layout_state (the in-memory copy other
  -- callers read — e.g. pair_ensure_visible_then) and the
  -- on-disk file (diagnostic only — layout_read derives from vim.o.lines
  -- now, so disk drift is harmless).
  pair_layout_state = s
  local f = io.open(LAYOUT_STATE_FILE, 'w')
  if f then f:write(s); f:close() end
end

local LAYOUT_LADDER = { minimized = 1, small = 2, third = 3 }
local LAYOUT_BY_LEVEL = { 'minimized', 'small', 'third' }

local function zellij_swap(direction)
  -- direction = 'next' or 'previous'
  if has_ui() then
    vim.fn.system({ 'zellij', 'action', direction .. '-swap-layout' })
  end
end

local function layout_step(from, to)
  if from == 'minimized' and to == 'small' then
    zellij_swap('previous')   -- minimized → default(small)
  elseif from == 'small' and to == 'minimized' then
    zellij_swap('next')       -- default(small) → minimized
  elseif from == 'small' and to == 'third' then
    zellij_swap('previous')   -- default(small) → third (last in cycle)
  elseif from == 'third' and to == 'small' then
    zellij_swap('next')       -- third → wraps to default(small)
  end
end

local function layout_goto(target)
  local cur = layout_read()
  local from = LAYOUT_LADDER[cur] or 1
  local to = LAYOUT_LADDER[target]
  if not to or from == to then
    -- Clamped at the ladder boundary (Alt+Up at third, or Alt+Down at
    -- minimized). The zellij keybind has already moved focus to nvim
    -- and forced normal mode (Ctrl-\ Ctrl-N), expecting the post-step
    -- recovery below to either move focus back (for minimized) or
    -- startinsert (for any expanded rung). Mirror that recovery here
    -- so the keystroke is a true no-op visually.
    if cur == 'minimized' then
      if has_ui() then
        vim.fn.system({ 'zellij', 'action', 'move-focus', 'up' })
      end
    else
      vim.cmd('startinsert')
    end
    return
  end
  -- Shrinking to minimized takes nvim to a single row, which can't hold the
  -- pinned-header winbar. Clear it now, before the zellij resize lands, so the
  -- resize never tries to render a winbar that doesn't fit (E36 → `-- More --`
  -- prompt that eats the next keystroke). pair_pin_header re-adds it on the
  -- way back up via VimResized.
  if target == 'minimized' then pcall(function() vim.wo.winbar = '' end) end
  local dir = to > from and 1 or -1
  for level = from, to - dir, dir do
    local next_level = level + dir
    layout_step(LAYOUT_BY_LEVEL[level], LAYOUT_BY_LEVEL[next_level])
  end
  layout_write(target)
  -- Refresh the statusline AFTER state has updated. Focus events fire
  -- before this point (zellij's keybind does MoveFocus Down → nvim
  -- FocusGained → refresh, all before our :lua call runs), so without
  -- this explicit refresh the statusline reads the previous state and
  -- the minimized hint sticks around when leaving minimized.
  refresh_statusline()
  -- Landing in minimized: nvim is now a single-row statusline strip and
  -- the user can't usefully interact with it. Shift focus to the agent
  -- pane so they can keep working.
  --
  -- Landing in small/third: the zellij keybind that triggered us escaped
  -- to normal mode (Ctrl-\ Ctrl-N) before invoking this lua. That's
  -- correct for minimized (where the locked sheet is the desired look),
  -- but for any expanded rung the user is here to type — startinsert
  -- puts the buffer into insert mode automatically.
  if target == 'minimized' then
    if has_ui() then
      vim.fn.system({ 'zellij', 'action', 'move-focus', 'up' })
    end
  else
    vim.cmd('startinsert')
  end
end

function _G.PairLayoutBigger()
  local cur = LAYOUT_LADDER[layout_read()] or 1
  layout_goto(LAYOUT_BY_LEVEL[math.min(cur + 1, #LAYOUT_BY_LEVEL)])
end

function _G.PairLayoutSmaller()
  local cur = LAYOUT_LADDER[layout_read()] or 1
  layout_goto(LAYOUT_BY_LEVEL[math.max(cur - 1, 1)])
end

-- Seed the in-memory mirror at startup. zellij boots into the size=12
-- draft pane (see zellij/layouts/main.kdl), and the in-memory mirror is
-- only used by callers that don't want to call layout_read; layout_read
-- itself reads vim.o.lines so it doesn't need this.
layout_write('small')

-- Alt+b — open the scrollback viewer already positioned on the previous
-- user prompt. This is a one-key shortcut for "Alt+/ then Alt+b": it opens
-- the same floating pane Alt+/ does (geometry mirrored from the Alt+/ bind
-- in zellij/config.kdl), but passes `--jump prev` so `pair scrollback open`
-- exports PAIR_SCROLLBACK_JUMP and nvim/scrollback.lua jumps to the prior
-- prompt right after positioning — identical to pressing Alt+b inside the
-- viewer. `zellij run` panes inherit the session env (PAIR_DATA_DIR etc.),
-- so the script resolves its inputs the same way the Alt+/ Run does. The
-- new floating pane takes focus, landing the user in the viewer.
local function pair_scrollback_prev_prompt()
  vim.fn.system({
    'zellij', 'run', '--floating', '--close-on-exit', '--name', 'scrollback',
    '--width', '100%', '--height', '100%', '--x', '0', '--y', '0',
    '--', 'pair', 'scrollback', 'open', '--jump', 'prev',
  })
end

-- ---------------------------------------------------------------------------
-- keymaps
-- ---------------------------------------------------------------------------

vim.keymap.set({ 'n', 'i' }, '<M-CR>', send_and_clear,
  { silent = true, desc = 'pair: send buffer + clear' })

vim.keymap.set({ 'n', 'i' }, '<S-M-CR>', function() send_and_clear(true) end,
  { silent = true, desc = 'pair: append buffer to agent (newline, no send) + clear' })

vim.keymap.set({ 'n', 'i' }, '<M-b>', pair_scrollback_prev_prompt,
  { silent = true, desc = 'pair: open scrollback on previous prompt (Alt+/ then Alt+b)' })

vim.keymap.set({ 'n', 'i' }, '<M-i>', attach_image,
  { silent = true, desc = 'pair: attach clipboard image (Ctrl+V to agent + ref)' })

-- Ctrl+C forwards ESC to the agent. send_esc_to_agent doesn't touch the draft's mode,
-- so in insert mode
-- you stay in insert (overriding <C-c>'s usual leave-insert) and in normal
-- mode the pending-command cancel is given up — both deliberate, so a reflexive
-- Ctrl+C interrupts the agent's stream without disrupting your draft.
vim.keymap.set({ 'n', 'i' }, '<C-c>', send_esc_to_agent,
  { silent = true, desc = 'pair: send ESC to agent (interrupt stream)' })

vim.keymap.set({ 'n', 'i' }, '<M-Left>', nav_left,
  { silent = true, desc = 'pair: navigate to older history entry' })

vim.keymap.set({ 'n', 'i' }, '<M-Right>', nav_right,
  { silent = true, desc = 'pair: navigate toward draft / queue' })

vim.keymap.set({ 'n', 'i' }, '<S-M-Left>',  function() nav_boundary(-1) end,
  { silent = true, desc = 'pair: jump to previous region boundary' })
vim.keymap.set({ 'n', 'i' }, '<S-M-Right>', function() nav_boundary( 1) end,
  { silent = true, desc = 'pair: jump to next region boundary' })

vim.keymap.set({ 'n', 'i' }, '<M-q>', queue_current,
  { silent = true, desc = 'pair: queue current draft for later (back of queue)' })

-- Alt+BS deletes the current +N queue item whenever the cursor is parked on a
-- queue slot — in BOTH normal and insert mode, so the gesture doesn't change
-- meaning mid-edit (#62). delete_current_queue_item self-guards to +N, so off
-- the queue the insert-mode binding falls back to <C-U> — kill from the cursor
-- to the start of the line (the macOS Cocoa convention scrollback.lua already
-- uses for <M-BS>/<M-Del>) — keeping that editing convenience on the * draft and
-- history slots. Normal-mode off the queue is a no-op (the function returns).
vim.keymap.set('n', '<M-BS>', delete_current_queue_item,
  { silent = true, desc = 'pair: delete the current +N queue item' })
vim.keymap.set('i', '<M-BS>', function()
  if type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    delete_current_queue_item()
  else
    vim.api.nvim_feedkeys(
      vim.api.nvim_replace_termcodes('<C-U>', true, false, true), 'n', false)
  end
end, { silent = true, desc = 'pair: delete +N queue item, else kill to line start' })

vim.keymap.set({ 'n', 'i' }, '<S-M-BS>', forget_all,
  { silent = true, desc = 'pair: erase history + draft + queue (with confirm)' })

vim.keymap.set('i', '<Tab>', function()
  return pum_visible() and '<C-n>' or '<Tab>'
end, { expr = true, desc = 'pair: cycle completion or insert tab' })

vim.keymap.set('i', '<S-Tab>', function()
  return pum_visible() and '<C-p>' or '<S-Tab>'
end, { expr = true, desc = 'pair: reverse-cycle completion or shift-tab' })

vim.keymap.set('i', '<CR>', function()
  -- spell_popup_active marks the momentary z= picker (clean dismiss, no newline);
  -- everything else is as-you-type draft completion (dismiss + newline on #65).
  return cr_keys(pum_visible(), pum_has_selection(), spell_popup_active)
end, { expr = true, desc = 'pair: accept selected completion, else dismiss popup (+newline unless z=)' })

vim.keymap.set('i', '<LeftMouse>', function()
  if pum_visible() then
    local pos = vim.fn.pum_getpos()
    if pos and pos.row and pos.col and pos.width and pos.height then
      local mouse = vim.fn.getmousepos()
      local prow = pos.row + 1
      local pcol = pos.col + 1
      if mouse.screenrow >= prow and mouse.screenrow < prow + pos.height and
         mouse.screencol >= pcol and mouse.screencol < pcol + pos.width then
        local target_idx = mouse.screenrow - prow
        local info = vim.fn.complete_info({ 'selected' })
        local current_selected = info.selected
        if current_selected == -1 then
          return string.rep('<C-n>', target_idx + 1) .. '<C-y>'
        elseif target_idx > current_selected then
          return string.rep('<C-n>', target_idx - current_selected) .. '<C-y>'
        elseif target_idx < current_selected then
          return string.rep('<C-p>', current_selected - target_idx) .. '<C-y>'
        else
          return '<C-y>'
        end
      end
    end
  end
  return '<LeftMouse>'
end, { expr = true, desc = 'pair: select and confirm completion menu item on click' })

-- ---------------------------------------------------------------------------
-- autocmds — all under the `pair` augroup so :luafile reloads cleanly.
-- ---------------------------------------------------------------------------

local pair_aug = vim.api.nvim_create_augroup('pair', { clear = true })

-- autosave on transitions so disk and buffer agree. Routes to the right
-- file per slot: * → draft via :w; +N → queue file via queue_write. -N is
-- immutable (history can't be mutated), so its edits wait for the explicit
-- Send/Queue/Discard choice in leave_dirty_history.
vim.api.nvim_create_autocmd({ 'BufLeave', 'FocusLost', 'InsertLeave' }, {
  group = pair_aug,
  pattern = '*',
  callback = function()
    autosave_current_slot()
    -- The :silent! write inside autosave can occasionally blank the
    -- statusline under cmdheight=0. Re-fire the (deferred) redraw so
    -- it comes back without the user needing an Alt+← to nudge it.
    refresh_statusline()
  end,
})

-- Statusline depends on mode (the dirty-N* mark) and on focus state.
-- ModeChanged and FocusGained/FocusLost should both trigger a redraw,
-- but nvim doesn't always re-evaluate the statusline on these events
-- under cmdheight=0. Defensive explicit refresh.
vim.api.nvim_create_autocmd({ 'ModeChanged', 'FocusGained', 'FocusLost' }, {
  group = pair_aug,
  callback = function() refresh_statusline() end,
})

-- Re-apply the soft statusline highlight on colorscheme changes (each
-- :colorscheme implicitly runs :hi clear, blowing away our link).
vim.api.nvim_create_autocmd('ColorScheme', {
  group = pair_aug,
  callback = pair_apply_statusline_hl,
})

-- start at end of buffer in insert mode — drafting is the default activity,
-- so don't make the user press `i` after every fresh launch.
vim.api.nvim_create_autocmd('VimEnter', {
  group = pair_aug,
  callback = function()
    vim.cmd('normal! G')
    vim.cmd('startinsert!')
  end,
})

-- Insert-mode-only keymap that triggers PairPasteQuote. This is what
-- bin/clipboard-to-pane sends (as a single Ctrl-_, ASCII 31) after a
-- mouse selection. Defining the keymap *only* in insert mode is the gate:
-- if nvim is in normal mode (e.g. browsing prompt history), Ctrl-_ hits
-- its default — a no-op-ish revins toggle — and PairPasteQuote simply
-- doesn't fire. No buffer mutation, no policy code.
vim.keymap.set('i', '<C-_>', function() PairPasteQuote() end,
  { silent = true, desc = 'pair: insert mouse-selected quote (insert mode only)' })

-- Auto-paired brackets and quotes in insert mode.
--
-- Hand-rolled minimal autopair: each opener inserts its closer and parks
-- the cursor between them; each closer "jumps out" if it would otherwise
-- duplicate the existing closer; backspace on an empty pair deletes both
-- halves. No syntax awareness, no string-detection cleverness — just a
-- one-char lookahead/lookbehind.
--
-- Next-char gate: only auto-insert the closer when the cursor sits at
-- end-of-line or in front of whitespace. If a real character follows, the
-- user is typing in front of existing text (e.g. `(` before `foo`), where a
-- trailing closer would just be in the way — so drop in the bare opener.
-- Applies to every pair, brackets and quotes alike.
--
-- Quote rule: if the next char is the same quote, jump over it (covers
-- typing the closer of a pair we just inserted). Otherwise, skip pairing
-- when the previous char is a word char, so apostrophes in "don't" /
-- "can't" stay single.
local PAIR_OPEN_TO_CLOSE = {
  ['('] = ')', ['['] = ']', ['{'] = '}',
  ['"'] = '"', ["'"] = "'", ['`'] = '`',
}
local PAIR_QUOTES = { ['"'] = true, ["'"] = true, ['`'] = true }

local function pair_cursor_chars()
  local line = vim.api.nvim_get_current_line()
  local col  = vim.api.nvim_win_get_cursor(0)[2]  -- 0-based byte column
  local prev = col > 0     and line:sub(col, col)         or ''
  local next = col < #line and line:sub(col + 1, col + 1) or ''
  return prev, next
end

local function pair_insert_open(open)
  local close = PAIR_OPEN_TO_CLOSE[open]
  local prev, next = pair_cursor_chars()
  if PAIR_QUOTES[open] then
    if next == open          then return '<C-G>U<Right>' end
    if prev:match('[%w_]')   then return open            end
  end
  -- Only pair when at EOL or in front of whitespace; if a real character
  -- follows, insert the bare opener so we don't strand a closer mid-word.
  if next ~= '' and not next:match('%s') then return open end
  -- <C-G>U keeps the inline cursor move from breaking undo into two units,
  -- so a single `u` undoes the whole opener+closer insertion.
  return open .. close .. '<C-G>U<Left>'
end

local function pair_insert_close(close)
  local _, next = pair_cursor_chars()
  if next == close then return '<C-G>U<Right>' end
  return close
end

local function pair_backspace()
  local prev, next = pair_cursor_chars()
  if prev ~= '' and PAIR_OPEN_TO_CLOSE[prev] == next then
    return '<BS><Del>'
  end
  return '<BS>'
end

for open, close in pairs(PAIR_OPEN_TO_CLOSE) do
  vim.keymap.set('i', open, function() return pair_insert_open(open) end,
    { silent = true, expr = true, desc = 'pair: autopair ' .. open })
  -- Closers for non-quote pairs only — quote keys map to the opener handler
  -- above, which already does the jump-over check.
  if not PAIR_QUOTES[open] then
    vim.keymap.set('i', close, function() return pair_insert_close(close) end,
      { silent = true, expr = true, desc = 'pair: jump over ' .. close })
  end
end
vim.keymap.set('i', '<BS>', pair_backspace,
  { silent = true, expr = true, desc = 'pair: smart-delete empty pair' })

-- <M-1>..<M-9>: quick-pick the Nth visible completion item from any popup.
-- The as-you-type menus (path/word/spell) are unlabelled now (uniform, no
-- numbering), so this is an unadvertised power-key — and it also swallows Alt+N
-- so the terminal's Esc+N sequence can't break insert mode. Outside the popup
-- these keys are no-ops — returning '' from an expr keymap leaves the buffer
-- unchanged. (The z= popup advertises its own bare-digit picking separately.)
--
-- Mechanism: feed `<C-n>` / `<C-p>` to land selection on item N, then
-- `<C-y>` to accept. We don't replace text manually; vim's accept handler
-- already knows the span passed to complete() and substitutes correctly.
local function pair_pick_completion(n)
  if vim.fn.pumvisible() == 0 then return '' end
  local info = vim.fn.complete_info({ 'items', 'selected' })
  if not info.items or not info.items[n] then return '' end
  local cn = vim.api.nvim_replace_termcodes('<C-n>', true, false, true)
  local cp = vim.api.nvim_replace_termcodes('<C-p>', true, false, true)
  local cy = vim.api.nvim_replace_termcodes('<C-y>', true, false, true)
  -- selected is 0-indexed (-1 = nothing selected, the noselect default).
  -- Treat -1 as "before item 0" so steps from -1 to target=0 is +1.
  local current = info.selected
  local target = n - 1
  local steps = (current >= 0) and (target - current) or (target + 1)
  local keys = ''
  if steps > 0 then keys = string.rep(cn, steps)
  elseif steps < 0 then keys = string.rep(cp, -steps) end
  return keys .. cy
end

for i = 1, 9 do
  vim.keymap.set('i', '<M-' .. i .. '>',
    function() return pair_pick_completion(i) end,
    { silent = true, expr = true, desc = 'pair: pick completion item ' .. i })
end

-- Bare digits pick a spell suggestion — but ONLY while the z= spell popup is
-- up (spell_popup_active + a visible menu). Everywhere else a digit is just a
-- digit, so the expr returns the literal key. CompleteDone then drops us back
-- to normal mode (the popup's <C-y> accept fires it).
local function spell_pick_digit(n)
  if spell_popup_active and vim.fn.pumvisible() == 1 then
    return pair_pick_completion(n)
  end
  return tostring(n)
end

for i = 1, 9 do
  vim.keymap.set('i', tostring(i),
    function() return spell_pick_digit(i) end,
    { silent = true, expr = true, desc = 'pair: pick spell suggestion ' .. i })
end

-- Fire on both events: TextChangedI when popup is hidden, TextChangedP when
-- popup is visible — refreshing the menu as the user types more characters.
-- path_complete handles slash/tilde tokens; word_complete kicks in for plain
-- alphanumeric tokens >= 6 chars. Their token regexes are mutually exclusive
-- (path needs `/` or `~`, word excludes both), so at most one calls complete().
-- spell_complete is the fallback: only when both decline does it offer
-- spelling fixes for a misspelled word-in-progress. Each completer returns
-- true once it calls complete(), so run_completers short-circuits and at most
-- one menu is built per keystroke.
--
-- Burst-debounce: paste dumps hundreds of TextChangedI events within a few
-- ms; running both completion handlers (each scans the buffer + agent file)
-- on every char stalls nvim and looks like flaky paste. Detection threshold
-- is 20ms — well above human typing cadence (worst-case ~40ms at 200wpm),
-- well below paste IO (~1-2ms). Bursts get coalesced into a single deferred
-- run 30ms after the last event.
local complete_last_fire = 0
local complete_pending = nil
local function run_completers()
  -- The explicit z= gesture (spell_suggest_popup) owns the popup while it's
  -- active, so the as-you-type completers must stay out of its way. Without
  -- this guard, startinsert's TextChangedI drives spell_complete to pop its own
  -- menu before z='s scheduled complete() runs; z= then replaces that menu, and
  -- the replacement's CompleteDone is misread by the teardown as "popup
  -- dismissed" → a scheduled stopinsert closes the just-opened z= menu (the
  -- first-z= menu flash).
  if spell_popup_active then return end
  if path_complete() then return end
  if word_complete() then return end
  spell_complete()
end
vim.api.nvim_create_autocmd({ 'TextChangedI', 'TextChangedP' }, {
  group = pair_aug,
  callback = function()
    local now = vim.loop.now()
    local delta = now - complete_last_fire
    complete_last_fire = now
    if complete_pending then
      complete_pending:stop()
      complete_pending:close()
      complete_pending = nil
    end
    if delta < 20 then
      complete_pending = vim.loop.new_timer()
      complete_pending:start(30, 0, vim.schedule_wrap(function()
        if complete_pending then
          complete_pending:close(); complete_pending = nil
        end
        run_completers()
      end))
      return
    end
    run_completers()
  end,
})

vim.keymap.set('n', 'z=', spell_suggest_popup,
  { silent = true, desc = 'pair: spell suggestions in completion popup' })

-- Track which words the user accepts from the completion popup. The pick
-- count feeds back into word_complete's ranking (PICK_WEIGHT). Fires for
-- every completion (path/word/spell); only word_complete consults picks,
-- but path/spell picks don't hurt — they just sit unused in the file.
-- v.completed_item is `{}` on cancel, so the empty-word guard handles it.
vim.api.nvim_create_autocmd('CompleteDone', {
  group = pair_aug,
  callback = function()
    local item = vim.v.completed_item
    if item and type(item) == 'table' and item.word then
      picks_bump(item.word)
    end
    -- z= forced insert mode purely to host the popup; once the suggestion is
    -- accepted (or dismissed) hand the user back the normal mode they came
    -- from. stopinsert can't run mid-CompleteDone, so defer it.
    if spell_popup_active then
      spell_popup_active = false
      vim.schedule(function() vim.cmd('stopinsert') end)
    end
  end,
})

-- Safety net: if the spell popup is torn down without a CompleteDone (e.g.
-- the popup never materialized), make sure the flag never survives into a
-- later, ordinary insert session where bare digits must stay literal.
vim.api.nvim_create_autocmd('InsertLeave', {
  group = pair_aug,
  callback = function() spell_popup_active = false end,
})

-- "Ghost cursor" while the nvim pane is unfocused. zellij hides the real
-- terminal cursor on FocusLost, leaving the insertion point invisible.
-- Mark the position with a glyph chosen by mode so the indicator mirrors
-- the focused-state cursor:
--   normal-mode unfocused : ▯ (outline of █, the focused block cursor)
--   insert-mode unfocused : ¦ (broken version of |, the focused bar cursor)
local pair_focus_ns = vim.api.nvim_create_namespace('pair_focus_cursor')

local function pair_apply_focus_cursor_hl()
  -- Tie to `Comment` so the glyph picks up the colorscheme's dimmed-text
  -- color — visible but subdued. Reapplied on ColorScheme since :hi clear
  -- (which colorschemes implicitly run) blows highlights away.
  vim.api.nvim_set_hl(0, 'PairFocusCursor', { link = 'Comment' })
end
pair_apply_focus_cursor_hl()

local function pair_show_focus_cursor()
  local mode  = vim.api.nvim_get_mode().mode:sub(1, 1)
  local glyph = (mode == 'i') and '¦' or '▯'
  local row1, col = unpack(vim.api.nvim_win_get_cursor(0))
  local row = row1 - 1
  vim.api.nvim_buf_clear_namespace(0, pair_focus_ns, 0, -1)
  pcall(vim.api.nvim_buf_set_extmark, 0, pair_focus_ns, row, col, {
    virt_text     = { { glyph, 'PairFocusCursor' } },
    virt_text_pos = 'overlay',
    priority      = 200,
  })
end

local function pair_hide_focus_cursor()
  vim.api.nvim_buf_clear_namespace(0, pair_focus_ns, 0, -1)
end

vim.api.nvim_create_autocmd('FocusLost', {
  group = pair_aug,
  callback = function()
    pair_show_focus_cursor()
    -- A delayed full redraw catches the case where zellij's focus-change
    -- rendering fires after our immediate refresh_statusline (which only
    -- defers one event-loop tick). 80ms is comfortably above one terminal
    -- frame and unobtrusive.
    vim.defer_fn(function() pcall(vim.cmd, 'redraw!') end, 80)
  end,
})
vim.api.nvim_create_autocmd('FocusGained', {
  group = pair_aug,
  callback = function()
    pair_hide_focus_cursor()
    refresh_statusline()
  end,
})
vim.api.nvim_create_autocmd('ColorScheme', { group = pair_aug, callback = pair_apply_focus_cursor_hl })

-- ---------------------------------------------------------------------------
-- Auto-orientation slug — dispose side (issue #000027)
-- ---------------------------------------------------------------------------
-- The Stop hook's `cmd/pair-slug` proposes a `=== <branch> | <focus> ===`
-- slug to slug-proposed-<tag>. We apply it to draft line 1 when safe and
-- mirror the effective line 1 back to slug-<tag> (the proposer's `prev`).
-- The buffer-safety decision lives in the pure, headless-tested nvim/slug.lua
-- (`make test-lua`); this is just the fs_event + file-IO wiring.
local pair_slug = nil
do
  local dir = debug.getinfo(1, 'S').source:match('@?(.*/)')
  if dir then
    local ok, mod = pcall(dofile, dir .. 'slug.lua')
    if ok then pair_slug = mod end
  end
end

-- :PairDoctor — agent-agnostic pair-doctor entry (issue #000048). The payload is
-- built by the pure, headless-tested nvim/doctor.lua; this is the IO seam: read
-- $PAIR_HOME and hand the absolute-pathed instruction to whatever agent is
-- running (works under any agent, unlike a Claude skill; works from any cwd,
-- since the paths are $PAIR_HOME-absolute). Auto-sends via send_to_agent.
do
  local doctor = nil
  local dir = debug.getinfo(1, 'S').source:match('@?(.*/)')
  if dir then
    local ok, mod = pcall(dofile, dir .. 'doctor.lua')
    if ok then doctor = mod end
  end
  local function pair_doctor()
    if not doctor then
      vim.notify('PairDoctor: nvim/doctor.lua failed to load.', vim.log.levels.ERROR)
      return
    end
    local body = doctor.payload(vim.env.PAIR_HOME)
    if not body then
      vim.notify('PairDoctor: PAIR_HOME unset (run inside a pair session).',
        vim.log.levels.ERROR)
      return
    end
    send_to_agent(body)
  end
  vim.api.nvim_create_user_command('PairDoctor', pair_doctor,
    { desc = 'Ask the agent to run pair-doctor and propose harness-drift fixes' })
  _G.PairDoctor = pair_doctor
end

local slug_pending = false -- a proposal arrived mid-insert; apply on InsertLeave

local function pair_slug_draft_buf()
  local want = draft_path_for_tag()
  for _, b in ipairs(vim.api.nvim_list_bufs()) do
    if vim.api.nvim_buf_is_loaded(b) and vim.api.nvim_buf_get_name(b) == want then
      return b
    end
  end
  return nil
end

-- Mirror line 1 to slug-<tag> so the proposer reads the user's edits as `prev`
-- next Stop (soft policy — the model biases toward keeping the user's
-- wording). Only slug-shaped / empty line 1 is mirrored; the user's prompt
-- text is never persisted as `prev`.
local function pair_slug_mirror_line1()
  local buf = pair_slug_draft_buf()
  if not buf then return end
  local line1 = (vim.api.nvim_buf_get_lines(buf, 0, 1, false))[1] or ''
  if line1 ~= '' and line1:sub(1, 3) ~= '===' then return end
  -- Idempotency: an apply rewrites line 1 via nvim_buf_set_lines, which itself
  -- fires TextChanged → this mirror. The apply already persisted the same
  -- value to slug-<tag>, so skip when unchanged — avoids churning the file
  -- (and the proposer's `prev`) on every machine apply.
  local path = pair_data_dir() .. '/slug-' .. pair_tag()
  local cur = read_file(path):match('^[^\n]*') or ''
  if cur == line1 then return end
  write_file(path, line1 .. '\n')
end

local function pair_slug_reconcile()
  if not pair_slug then return end
  local dd, tag = pair_data_dir(), pair_tag()
  local proposed = read_file(dd .. '/slug-proposed-' .. tag):match('^[^\n]*') or ''
  if proposed == '' then return end
  local buf = pair_slug_draft_buf()
  if not buf then return end
  local line1 = (vim.api.nvim_buf_get_lines(buf, 0, 1, false))[1] or ''
  -- Defer ONLY when the cursor is actually on a non-empty line 1 — i.e. the
  -- user is editing the slug itself — so the rewrite never lands under their cursor.
  -- Empty line 1 is initialization, not an edit-in-progress; apply immediately
  -- and let slug.lua move the cursor to the blank prompt line below.
  -- The draft pane sits in insert mode almost permanently while composing on
  -- line 2+, so gating on insert-mode-at-all (the original check) deferred
  -- every live proposal forever; gating on the line-1 cursor is the real
  -- safety condition. Re-runs on InsertLeave/CursorMoved off line 1.
  local win = vim.fn.bufwinid(buf)
  if line1 ~= '' and win ~= -1 and vim.api.nvim_win_get_cursor(win)[1] == 1 then
    slug_pending = true
    return
  end
  local _, prev = pair_slug.apply(buf, proposed)
  if prev == '' or prev:sub(1, 3) == '===' then
    write_file(dd .. '/slug-' .. tag, prev .. '\n')
  end
end

do
  local handle = vim.loop.new_fs_event()
  if handle then
    local target = 'slug-proposed-' .. pair_tag()
    local ok = pcall(function()
      handle:start(pair_data_dir(), {}, vim.schedule_wrap(function(err, filename)
        if err then return end
        if filename and filename ~= target then return end
        pcall(pair_slug_reconcile)
      end))
    end)
    if not ok then handle:close() end
  end
end

-- Re-run a deferred apply once the cursor leaves line 1 (the defer condition).
-- CursorMoved/CursorMovedI fire as the user moves down to compose; InsertLeave
-- covers leaving insert mode while still on line 1.
vim.api.nvim_create_autocmd({ 'CursorMoved', 'CursorMovedI', 'InsertLeave' }, {
  group = pair_aug,
  callback = function()
    if slug_pending then
      slug_pending = false
      pcall(pair_slug_reconcile)
    end
  end,
})

-- Debounced mirror of the user's line-1 edits (see pair_slug_mirror_line1).
do
  local timer = vim.loop.new_timer()
  vim.api.nvim_create_autocmd({ 'TextChanged', 'TextChangedI' }, {
    group = pair_aug,
    callback = function()
      if vim.bo.filetype ~= 'markdown' or not timer then return end
      timer:stop()
      timer:start(400, 0, vim.schedule_wrap(function() pcall(pair_slug_mirror_line1) end))
    end,
  })
end

-- Pick up any proposal already on disk at startup (e.g. a Stop fired while
-- the draft pane was restarting). Deferred to VimEnter: running it inline
-- during init races the draft-buffer load, so pair_slug_draft_buf() finds
-- nothing and the slug isn't applied until the next Stop.
vim.api.nvim_create_autocmd('VimEnter', {
  group = pair_aug,
  once = true,
  callback = function() pcall(pair_slug_reconcile) end,
})

-- Review-mode bar content (#66 M3/M4d) — the compact review segment shown in the
-- pair (draft) view while a review is open: `-H < pos > +Q • 🪄 Mode • file •
-- ... 🤖 A/H` (A agent / H human round commits). This just builds the review
-- string; PairStatusline supplies the compact history/queue prefix. Supersedes the
-- earlier line-1 `=== review … ===` indicator (line 1 is the user's to edit — a
-- bar is chrome, not buffer content). (`do ... end`: no file-level locals —
-- init.lua is at Lua's 200-local chunk ceiling; reuses `_pair_review.is_alive`.)
do
  local nvim_dir = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
  local seam = dofile(nvim_dir .. 'review/seam.lua')

  -- Rounds in the CURRENT review session only. The session lives on a
  -- `review/<slug>` branch (M4, agent-owned); count that slug's round commits.
  -- On any OTHER branch — including M3 render-only, where no review is active —
  -- return 0/0. (A repo's history can hold dozens of shipped reviews of OTHER docs:
  -- `review(a-blogging-workflow): …` etc. Counting repo-wide gave the spurious
  -- "25/28" — those are other docs' shipped rounds, not this session.)
  -- Slug-scoped, not just branch-scoped, so a merged prior review of a *different*
  -- doc reachable from HEAD doesn't leak in. (M4 refinement: merge-base..HEAD to
  -- also drop a prior shipped review of the SAME slug.)
  local function counts(file)
    local dir = vim.fn.fnamemodify(file, ':h')
    local branch = vim.fn.system({ 'git', '-C', dir, 'branch', '--show-current' })
    if vim.v.shell_error ~= 0 then return 0, 0 end
    local slug = vim.trim(branch):match('^review/(.+)$')
    if not slug then return 0, 0 end -- not in an active review (M3 / between reviews)
    local out = vim.fn.system({ 'git', '-C', dir, 'log', '--pretty=%s' })
    if vim.v.shell_error ~= 0 then return 0, 0 end
    local pa = '^review%(' .. vim.pesc(slug) .. '%):%s*agent%s'
    local ph = '^review%(' .. vim.pesc(slug) .. '%):%s*human%s'
    local a, h = 0, 0
    for line in out:gmatch('[^\n]+') do
      if line:match(pa) then a = a + 1
      elseif line:match(ph) then h = h + 1 end
    end
    return a, h
  end

  -- The compact draft-side review bar text. 🤖A = agent (robot) rounds, /H = human.
  local function bar_text(file)
    local a, h = counts(file)
    local m = seam.read_mode(pair_data_dir(), pair_tag())
    return string.format('🪄 %s • %s • 🤖 %d/%d', seam.mode_label(m), vim.fn.fnamemodify(file, ':t'), a, h)
  end
  _G._pair_review_bar = bar_text -- exposed for the headless test (plain, unstyled)

  -- Review-mode accent for the folded statusline segment (reapplied on ColorScheme
  -- since :colorscheme runs :hi clear).
  local function set_hl()
    vim.api.nvim_set_hl(0, 'PairReviewBar', { fg = '#56b6c2', bold = true, default = true })
  end
  set_hl()
  vim.api.nvim_create_autocmd('ColorScheme', { callback = set_hl })

  -- Cached statusline segment, recomputed off a timer — the statusline renders on
  -- nearly every keystroke/cursor move, so it must NEVER shell git inline. `nil`
  -- when no review is open; PairStatusline shows the normal cheatsheet then.
  local cached = nil
  local function recompute()
    local seg = nil
    if _G._pair_review then
      local alive, _, file = _G._pair_review.is_alive()
      if alive and file and file ~= '' then
        local a, h = counts(file)
        local m = seam.read_mode(pair_data_dir(), pair_tag())
        local label, name = seam.mode_label(m), vim.fn.fnamemodify(file, ':t')
        seg = string.format('%%#PairReviewBar#🪄 %s • %s •%%*%%=%%#PairReviewBar#🤖 %d/%d%%*',
          label:gsub('%%', '%%%%'), name:gsub('%%', '%%%%'), a, h)
      end
    end
    if seg ~= cached then
      cached = seg
      pcall(refresh_statusline) -- redraw on open / close / count change
    end
  end
  _G._pair_review_segment = function() return cached end

  local timer = vim.loop.new_timer()
  if timer then timer:start(800, 1500, vim.schedule_wrap(function() pcall(recompute) end)) end
end

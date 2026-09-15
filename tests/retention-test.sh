#!/usr/bin/env bash
set -euo pipefail
repo=$(cd "$(dirname "$0")/.." && pwd)
scratch=$(mktemp -d "${TMPDIR:-/tmp}/pair-retention.XXXXXX")
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/data"
go build -o "$scratch/bin/pair-go" ./cmd/pair-go
ln -s pair-go "$scratch/bin/pair"
cat > "$scratch/check.lua" <<'LUA'
local root = assert(vim.env.PAIR_DATA_DIR)
local module = dofile(assert(vim.env.RETENTION_MODULE))
local guard = assert(module.setup('integration-reader'))
local function state()
  local files = vim.fn.glob(root .. '/.retention/owners/*.json', false, true)
  assert(#files == 1)
  return vim.json.decode(table.concat(vim.fn.readfile(files[1]), '\n'))
end
local s = state()
assert(#s.processes == 1 and s.processes[1].process.pid == vim.fn.getpid(), 'registered CLI instead of editor')
local target = root .. '/draft-test.md'
assert(guard:write(target, 'changed\n', function() return '' end, function(body)
  assert(#state().intents == 1, 'write began without durable intent')
  local f = assert(io.open(target, 'w')); f:write(body); f:close(); return true
end))
s = state()
assert(not s.intents or #s.intents == 0)
local at = s.activity.last_use
assert(guard:write(target, 'changed\n', function() return 'changed\n' end, function() error('unchanged write') end))
assert(state().activity.last_use == at, 'unchanged content refreshed use')
assert(guard:close())
s = state()
assert(not s.processes or #s.processes == 0)
print('retention real CLI/editor integration passed')
LUA
env -u PAIR_SCOPE_KEY -u PAIR_RETENTION_START_ID PATH="$scratch/bin:$PATH" PAIR_DATA_DIR="$scratch/data" PAIR_TAG=test PAIR_RETENTION_PROTOCOL=1 RETENTION_MODULE="$repo/nvim/retention.lua" nvim -l "$scratch/check.lua"
mkdir -p "$scratch/preview"
printf '%s\n' 'untouched' > "$scratch/preview/draft-preview.md"
"$scratch/bin/pair" gc --root "$scratch/preview" --json > "$scratch/preview.json"
jq -e '.storage.migration_complete == false' "$scratch/preview.json" >/dev/null
[ ! -e "$scratch/preview/.retention" ]
[ "$(cat "$scratch/preview/draft-preview.md")" = untouched ]
printf '%s\n' 'public pair gc preview leaves storage unchanged'

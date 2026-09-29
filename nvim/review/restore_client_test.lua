package.path = './nvim/?.lua;' .. package.path
local ok, module = pcall(require, 'review.restore_client')
assert(ok, 'draft restoration client must exist')
local root = vim.fn.tempname(); vim.fn.mkdir(root, 'p')
local identity = { status='resolved', repo='/repo', branch='review/a', head='sha-a', file="a's.md" }
local context = { repo=identity.repo, branch=identity.branch, file=identity.file, activation='active-a' }
local function scenario(config)
  config = config or {}
  local sf=root..'/pane.open'
  local metadata={version=1,endpoint='/private/socket',token='secret',session='session-a',context=context}
  if config.foreign then metadata.session='session-b' end
  if config.pane ~= false then
    vim.fn.writefile({tostring(vim.fn.getpid()),'/repo/old.md',config.legacy and '' or vim.json.encode(metadata)}, sf)
  else vim.fn.delete(sf) end
  local effects, calls, notifications, published, resolve_count = {}, {}, {}, nil, 0
  local session='session-a'
  local client = module.new({
    state_file=function() return sf end,
    session=function() return session end,
    read_target=function() return config.target end,
    write_target=function(file,status,id) published={file=file,status=status,identity=id}; effects[#effects+1]='publish' end,
    pair_bin=function() return '/pair' end,
    cwd=function() return '/checkout' end,
    hide=function() effects[#effects+1]='hide' end,
    prompt=function() effects[#effects+1]='prompt' end,
    notify=function(msg) notifications[#notifications+1]=msg end,
    run=function(cmd, opts, cb)
      calls[#calls+1]=cmd
      local result={code=0,stdout='',stderr=''}
      if vim.tbl_contains(cmd,'--resolve') then
        resolve_count=resolve_count+1
        local observed=vim.deepcopy(identity)
        if config.nonreview then observed.status='non_review'
        elseif config.missing and not vim.tbl_contains(cmd,'--selected') then observed.status='missing'
        elseif config.drift and resolve_count > 1 then observed.head='changed' end
        result.stdout=vim.json.encode(observed)
        effects[#effects+1]='resolve'
      elseif vim.tbl_contains(cmd,'--remote-expr') then
        -- Evaluate the fixed expression locally to prove quotes in identities
        -- are transported as data, using the same Luaeval decoding contract.
        _G.PairReviewPane={restore=function(raw)
          local request=vim.json.decode(raw)
          assert(request.token=='secret' and request.session=='session-a')
          assert(request.identity.file==identity.file)
          return {ok=not config.blocked,error='pending review request',same=not config.switched,identity=identity,context=context}
        end}
        result.stdout=vim.fn.eval(cmd[#cmd])
        if config.sessiondrift then session='session-new' end
        if config.panedrift then metadata.token='new-token'; vim.fn.writefile({tostring(vim.fn.getpid()),'/repo/old.md',vim.json.encode(metadata)},sf) end
        if config.badack then local a=vim.json.decode(result.stdout); a.context.branch='review/wrong'; result.stdout=vim.json.encode(a) end
        if config.timeout then result={code=124,stderr='timeout'} end
        effects[#effects+1]='rpc'
      elseif vim.tbl_contains(cmd,'open') then
        assert(opts.env.PAIR_SESSION_ID=='session-a')
        assert(vim.json.decode(opts.env.PAIR_REVIEW_IDENTITY).head==identity.head)
        vim.fn.writefile({tostring(vim.fn.getpid()),'/repo/'..identity.file,vim.json.encode(metadata)},sf)
        effects[#effects+1]='open'
      elseif vim.tbl_contains(cmd,'are-floating-panes-visible') then result.stdout=config.visible==false and 'false' or 'true'; effects[#effects+1]='visibility'
      elseif vim.tbl_contains(cmd,'show-floating-panes') then effects[#effects+1]='show'
      else error('unexpected command '..vim.inspect(cmd)) end
      vim.schedule(function() cb(result) end)
      return {}
    end,
  })
  client:toggle()
  client:toggle() -- duplicate calls cannot start a second transaction
  assert(vim.wait(1500,function() return not client.busy end,10),'client did not finish')
  return {effects=effects,calls=calls,notifications=notifications,published=published,resolves=resolve_count}
end
local r=scenario()
assert(vim.deep_equal(r.effects,{'resolve','rpc','resolve','publish','visibility','hide'}),vim.inspect(r.effects))
r=scenario({switched=true})
assert(vim.deep_equal(r.effects,{'resolve','rpc','resolve','publish','show'}))
r=scenario({visible=false})
assert(r.effects[#r.effects]=='show')
r=scenario({pane=false})
assert(vim.deep_equal(r.effects,{'resolve','open','rpc','resolve','publish','show'}))
for _, mode in ipairs({'foreign','legacy','timeout','blocked','drift','sessiondrift','panedrift','badack'}) do
  r=scenario({[mode]=true})
  assert(not r.published and not vim.tbl_contains(r.effects,'show') and not vim.tbl_contains(r.effects,'open'),mode)
  assert(#r.notifications>0,mode..' must diagnose refusal')
end
r=scenario({nonreview=true,target={status='ready',file='/stale.md'}})
assert(vim.deep_equal(r.effects,{'resolve','prompt'}))
r=scenario({missing=true,target={status='ready',file='/repo/'..identity.file}})
assert(vim.deep_equal(r.effects,{'resolve','prompt'}),'generic target cannot authorize history-free restore')
r=scenario({missing=true,target={status='ready',file='/repo/'..identity.file,identity=identity}})
assert(r.published and r.resolves==4,'matching prepared selection must be independently verified twice')
local stale=vim.deepcopy(identity); stale.head='old'
r=scenario({missing=true,target={status='ready',identity=stale}})
assert(not r.published and r.resolves==1)
vim.fn.delete(root,'rf')
print('restore client tests passed')

<script>
  import { api } from '../lib/api.js'
  import { teams, loadTeams } from '../lib/stores.js'

  let { def, isNew, onsave, oncancel } = $props()

  let name = $state(def.name || '')
  let description = $state(def.description || '')
  let model = $state(def.model || '')
  let systemPrompt = $state(def.system_prompt || '')
  let maxTurns = $state(def.max_turns || 0)
  let maxConcurrentTools = $state(def.max_concurrent_tools || 0)
  let forceJson = $state(def.force_json || false)
  let scope = $state(def.scope || 'user')
  let team = $state(def.team || '')
  let soEnabled = $state(!!(def.structured_output))
  let soJSON = $state(def.structured_output ? JSON.stringify(def.structured_output, null, 2) : '')

  let availableServers = $state([])
  let availableAgents = $state([])
  let loading = $state(true)
  let enabledTools = $state({})
  let selectedServers = $state([])
  let expandedServer = $state(null)
  let subAgents = $state([])

  $effect(() => {
    loadAll()
    loadTeams()
  })

  async function loadAll() {
    try {
      loading = true
      const [servers, agents] = await Promise.all([
        api.get('/tools/servers/list'),
        api.get('/agents/list')
      ])
      availableServers = servers
      availableAgents = (agents || []).filter(a => a.name !== def.name)
      initFromDef()
    } catch (e) {
      console.error('Failed to load data', e)
    } finally {
      loading = false
    }
  }

  function initFromDef() {
    if (!def.tools || def.tools.length === 0) return
    const et = {}
    const sa = []
    for (const ref of def.tools) {
      const dot = ref.indexOf('.')
      if (dot === -1) {
        sa.push(ref)
      } else {
        const srv = ref.slice(0, dot)
        const tool = ref.slice(dot + 1)
        if (!et[srv]) et[srv] = new Set()
        et[srv].add(tool)
      }
    }
    enabledTools = et
    selectedServers = Object.keys(et)
    subAgents = sa
  }

  function addServer(name) {
    if (selectedServers.includes(name)) return
    selectedServers = [...selectedServers, name]
    if (!enabledTools[name]) {
      enabledTools = { ...enabledTools, [name]: new Set() }
    }
    expandedServer = name
  }

  function removeServer(name) {
    selectedServers = selectedServers.filter(s => s !== name)
    const et = { ...enabledTools }
    delete et[name]
    enabledTools = et
    if (expandedServer === name) expandedServer = null
  }

  function toggleTool(serverName, toolName) {
    const et = { ...enabledTools }
    if (!et[serverName]) et[serverName] = new Set()
    const set = new Set(et[serverName])
    if (set.has(toolName)) {
      set.delete(toolName)
    } else {
      set.add(toolName)
    }
    et[serverName] = set
    enabledTools = et
  }

  function addSubAgent(agentName) {
    if (subAgents.includes(agentName)) return
    subAgents = [...subAgents, agentName]
  }

  function removeSubAgent(agentName) {
    subAgents = subAgents.filter(a => a !== agentName)
  }

  function getServer(name) {
    return availableServers.find(s => s.name === name)
  }

  function getServerTools(name) {
    const srv = getServer(name)
    return srv ? srv.tools : []
  }

  function getOrphanedTools(name) {
    const et = enabledTools[name]
    if (!et || et.size === 0) return []
    const found = new Set(getServerTools(name).map(t => t.name))
    return [...et].filter(t => !found.has(t))
  }

  function enabledCount(name) {
    return enabledTools[name]?.size || 0
  }

  function totalEnabledCount() {
    let count = subAgents.length
    for (const s of selectedServers) {
      count += enabledTools[s]?.size || 0
    }
    return count
  }

  function availableServerOptions() {
    return availableServers.filter(s => !selectedServers.includes(s.name))
  }

  function availableAgentOptions() {
    return availableAgents.filter(a => !subAgents.includes(a.name))
  }

  function scopeLabel(srv) {
    if (!srv) return 'personal'
    return srv.scope || 'personal'
  }

  function isChecked(serverName, toolName) {
    return enabledTools[serverName]?.has(toolName) || false
  }

  function getAgentDesc(agentName) {
    const a = availableAgents.find(aa => aa.name === agentName)
    return a?.description || ''
  }

  function handleSave() {
    const tools = []
    for (const s of selectedServers) {
      const et = enabledTools[s]
      if (!et) continue
      for (const t of et) {
        tools.push(s + '.' + t)
      }
    }
    for (const a of subAgents) {
      tools.push(a)
    }

    let so = null
    if (soEnabled && soJSON.trim()) {
      try {
        so = JSON.parse(soJSON)
      } catch {
        alert('Invalid structured output JSON')
        return
      }
    }

    onsave({
      originalName: def.name,
      name,
      description,
      model,
      system_prompt: systemPrompt,
      tools,
      max_turns: maxTurns,
      max_concurrent_tools: maxConcurrentTools,
      force_json: forceJson,
      scope,
      team,
      structured_output: so,
      kind: 'agent',
    })
  }
</script>

<div class="form-page">
  <div class="form-header">
    <h2 class="form-title">{isNew ? 'New Agent' : 'Edit Agent'}</h2>
    <div class="form-header-actions">
      <button onclick={handleSave} class="sb-submit" style="width:auto;">Save</button>
      <button onclick={oncancel} class="cancel-btn">Cancel</button>
    </div>
  </div>

  <div class="form-body">
    <div class="form-group">
      <label class="form-label">Name</label>
      <input value={name} oninput={(e) => name = e.target.value} class="sb-input" placeholder="Agent name" />
    </div>

    <div class="form-group">
      <label class="form-label">Description</label>
      <input value={description} oninput={(e) => description = e.target.value} class="sb-input" placeholder="Brief description" />
    </div>

    <div class="form-group">
      <label class="form-label">Model</label>
      <input value={model} oninput={(e) => model = e.target.value} class="sb-input" placeholder="e.g. gpt-4o" />
    </div>

    <div class="form-group">
      <label class="form-label">System Prompt</label>
      <textarea value={systemPrompt} oninput={(e) => systemPrompt = e.target.value} class="sb-input form-textarea" placeholder="System prompt..." rows="6"></textarea>
    </div>

    <div class="form-group">
      <div class="tool-section-header">
        <label class="form-label" style="margin-bottom:0;">Tools & Sub-agents</label>
        {#if totalEnabledCount() > 0}
          <span class="tool-count-badge">{totalEnabledCount()} enabled</span>
        {/if}
      </div>

      {#if loading}
        <p class="tool-hint">Loading...</p>
      {:else}
        <div class="tool-section-label">Sub-agents</div>
        <p class="tool-hint">Use other agents as tools. Call them by name in the agent's system prompt.</p>

        {#if availableAgentOptions().length > 0}
          <div class="tool-add-row">
            <select class="sb-input tool-server-select" id="agent-select">
              <option value="">-- Select agent --</option>
              {#each availableAgentOptions() as a}
                <option value={a.name}>{a.name}</option>
              {/each}
            </select>
            <button class="sb-submit" style="width:auto; padding:9px 16px; font-size:0.82rem;" onclick={() => {
              const sel = document.getElementById('agent-select')
              if (sel && sel.value) { addSubAgent(sel.value); sel.value = '' }
            }}>Add</button>
          </div>
        {/if}

        {#if subAgents.length > 0}
          <div class="sub-agent-list">
            {#each subAgents as agentName}
              <div class="sub-agent-chip">
                <span class="sub-agent-icon">🤖</span>
                <div class="sub-agent-info">
                  <span class="sub-agent-name">{agentName}</span>
                  {#if getAgentDesc(agentName)}
                    <span class="sub-agent-desc">{getAgentDesc(agentName)}</span>
                  {/if}
                </div>
                <button class="sub-agent-remove" onclick={() => removeSubAgent(agentName)} title="Remove">&times;</button>
              </div>
            {/each}
          </div>
        {:else}
          <p class="tool-hint" style="margin-bottom:16px;">No sub-agents selected.</p>
        {/if}

        <div class="tool-divider"></div>

        <div class="tool-section-label">MCP Server Tools</div>

        {#if availableServerOptions().length > 0}
          <div class="tool-add-row">
            <select class="sb-input tool-server-select" id="server-select">
              <option value="">-- Select MCP server --</option>
              {#each availableServerOptions() as srv}
                <option value={srv.name}>{srv.name} ({srv.tools.length} tools)</option>
              {/each}
            </select>
            <button class="sb-submit" style="width:auto; padding:9px 16px; font-size:0.82rem;" onclick={() => {
              const sel = document.getElementById('server-select')
              if (sel && sel.value) { addServer(sel.value); sel.value = '' }
            }}>Add</button>
          </div>
        {:else if selectedServers.length === 0}
          <p class="tool-hint">No MCP servers available. Register one in the Tools tab first.</p>
        {/if}

        {#if selectedServers.length > 0}
          <div class="tool-servers-list">
            {#each selectedServers as srvName}
              {@const srv = getServer(srvName)}
              {@const tools = getServerTools(srvName)}
              {@const orphaned = getOrphanedTools(srvName)}
              {@const expanded = expandedServer === srvName}
              <div class="tool-server-card" class:expanded>
                <div class="tool-server-card-header" onclick={() => expandedServer = expanded ? null : srvName} onkeydown={(e) => { if (e.key === 'Enter') expandedServer = expanded ? null : srvName }} role="button" tabindex="0">
                  <div class="tool-server-card-main">
                    <span class="tool-server-status" class:connected={srv?.connected} class:disconnected={!srv?.connected}>{srv?.connected ? '●' : '○'}</span>
                    <span class="tool-server-name">{srvName}</span>
                    <span class="tool-server-scope scope-{scopeLabel(srv)}">{scopeLabel(srv)}</span>
                  </div>
                  <div class="tool-server-card-meta">
                    <span class="tool-server-count">{enabledCount(srvName)}/{tools.length + orphaned.length}</span>
                    <span class="tool-server-expand">{expanded ? '▾' : '▸'}</span>
                  </div>
                </div>

                {#if expanded}
                  <div class="tool-server-card-body">
                    <div class="tool-server-actions">
                      <button class="action-btn action-btn-delete" onclick={() => removeServer(srvName)}>Remove</button>
                      {#if tools.length > 0}
                        <button class="action-btn" onclick={() => {
                          const et = { ...enabledTools }
                          et[srvName] = new Set(tools.map(t => t.name))
                          enabledTools = et
                        }}>Select All</button>
                        <button class="action-btn" onclick={() => {
                          const et = { ...enabledTools }
                          et[srvName] = new Set()
                          enabledTools = et
                        }}>Deselect All</button>
                      {/if}
                    </div>

                    {#if tools.length > 0}
                      <div class="tool-checkboxes">
                        {#each tools as tool}
                          <label class="tool-checkbox-label">
                            <input
                              type="checkbox"
                              checked={isChecked(srvName, tool.name)}
                              onchange={() => toggleTool(srvName, tool.name)}
                            />
                            <span class="tool-checkbox-name">{tool.name}</span>
                            <span class="tool-checkbox-desc">{tool.description || ''}</span>
                          </label>
                        {/each}
                      </div>
                    {:else}
                      <p class="tool-hint" style="padding:8px 0;">No tools discovered from this server.</p>
                    {/if}

                    {#if orphaned.length > 0}
                      <div class="tool-orphaned-section">
                        <span class="tool-orphaned-label">Previously enabled (no longer on server):</span>
                        {#each orphaned as tool}
                          <label class="tool-checkbox-label tool-orphaned">
                            <input
                              type="checkbox"
                              checked={true}
                              onchange={() => toggleTool(srvName, tool)}
                            />
                            <span class="tool-checkbox-name">{tool}</span>
                          </label>
                        {/each}
                      </div>
                    {/if}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    </div>

    <div class="form-row">
      <div class="form-group">
        <label class="form-label">Max Turns</label>
        <input value={maxTurns} oninput={(e) => maxTurns = e.target.valueAsNumber || 0} type="number" class="sb-input" placeholder="0 = unlimited" />
      </div>
      <div class="form-group">
        <label class="form-label">Max Concurrent Tools</label>
        <input value={maxConcurrentTools} oninput={(e) => maxConcurrentTools = e.target.valueAsNumber || 0} type="number" class="sb-input" placeholder="0 = unlimited" />
      </div>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label class="form-label">Scope</label>
        <select value={scope} onchange={(e) => scope = e.target.value} class="sb-input">
          <option value="user">personal</option>
          <option value="team">team</option>
          <option value="global">global</option>
        </select>
      </div>
      {#if scope === 'team'}
        <div class="form-group">
          <label class="form-label">Team</label>
          <select value={team} onchange={(e) => team = e.target.value} class="sb-input">
            <option value="">-- select team --</option>
            {#each $teams as t}
              <option value={t}>{t}</option>
            {/each}
          </select>
        </div>
      {/if}
    </div>

    <div class="form-group">
      <label class="form-check">
        <input checked={forceJson} onchange={(e) => forceJson = e.target.checked} type="checkbox" />
        Force JSON output
      </label>
    </div>

    <div class="form-group">
      <label class="form-check">
        <input checked={soEnabled} onchange={(e) => soEnabled = e.target.checked} type="checkbox" />
        Structured Output
      </label>
      {#if soEnabled}
        <textarea value={soJSON} oninput={(e) => soJSON = e.target.value} class="sb-input form-textarea mono" placeholder='"name": "", "schema": null, "strict": false' rows="8" style="margin-top:8px;"></textarea>
      {/if}
    </div>
  </div>
</div>

<style>
  .form-page {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
  }
  .form-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
  }
  .form-header-actions {
    display: flex;
    gap: 8px;
  }
  .form-title {
    font-size: 1.2rem;
    font-weight: 700;
    margin: 0;
  }
  .cancel-btn {
    padding: 9px 16px;
    background: var(--bg-card);
    color: var(--text-muted);
    border: 1px solid var(--border);
    border-radius: 8px;
    font-family: inherit;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }
  .cancel-btn:hover { background: var(--bg-sidebar); color: var(--text-base); }
  .form-body {
    max-width: 680px;
  }
  .form-group {
    margin-bottom: 16px;
  }
  .form-label {
    display: block;
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--text-muted);
    margin-bottom: 6px;
  }
  .form-textarea {
    resize: vertical;
    min-height: 80px;
    font-family: inherit;
  }
  .form-textarea.mono {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.8rem;
  }
  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .form-check {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.82rem;
    color: var(--text-base);
    cursor: pointer;
  }
  .form-check input[type="checkbox"] {
    accent-color: var(--purple-solid);
  }

  /* Tool picker */
  .tool-section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
  }
  .tool-count-badge {
    font-size: 0.72rem;
    font-weight: 600;
    color: var(--purple-solid);
    background: var(--purple-soft);
    padding: 2px 8px;
    border-radius: 10px;
  }
  .tool-section-label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 4px;
  }
  .tool-divider {
    border-top: 1px solid var(--border);
    margin: 16px 0 12px 0;
  }
  .tool-add-row {
    display: flex;
    gap: 8px;
    align-items: center;
    margin-bottom: 12px;
  }
  .tool-server-select {
    flex: 1;
  }
  .tool-hint {
    font-size: 0.78rem;
    color: var(--text-muted);
    margin: 4px 0;
    line-height: 1.4;
  }

  /* Sub-agent chips */
  .sub-agent-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 4px;
  }
  .sub-agent-chip {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 8px;
  }
  .sub-agent-icon {
    font-size: 0.9rem;
    flex-shrink: 0;
  }
  .sub-agent-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
    flex: 1;
  }
  .sub-agent-name {
    font-size: 0.82rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .sub-agent-desc {
    font-size: 0.72rem;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .sub-agent-remove {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 0.95rem;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
    flex-shrink: 0;
  }
  .sub-agent-remove:hover { color: #ef4444; }

  /* Server cards */
  .tool-servers-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .tool-server-card {
    border: 1px solid var(--border);
    border-radius: 8px;
    overflow: hidden;
    background: var(--bg-card);
  }
  .tool-server-card.expanded {
    border-color: var(--purple-soft);
  }
  .tool-server-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    cursor: pointer;
    user-select: none;
    transition: background 0.1s;
  }
  .tool-server-card-header:hover {
    background: var(--bg-sidebar);
  }
  .tool-server-card-main {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .tool-server-status {
    font-size: 0.6rem;
  }
  .tool-server-status.connected { color: #22c55e; }
  .tool-server-status.disconnected { color: var(--text-muted); }
  .tool-server-name {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .tool-server-scope {
    font-size: 0.65rem;
    font-weight: 600;
    text-transform: uppercase;
    padding: 1px 6px;
    border-radius: 4px;
  }
  .tool-server-scope.scope-personal { background: var(--bg-sidebar); color: var(--text-muted); }
  .tool-server-scope.scope-team { background: #1e3a5f; color: #60a5fa; }
  .tool-server-scope.scope-global { background: #3b1f5e; color: #c084fc; }
  .tool-server-scope.scope-user { background: var(--bg-sidebar); color: var(--text-muted); }
  .tool-server-card-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.78rem;
    color: var(--text-muted);
  }
  .tool-server-count {
    font-variant-numeric: tabular-nums;
  }
  .tool-server-expand {
    font-size: 0.7rem;
  }
  .tool-server-card-body {
    border-top: 1px solid var(--border);
    padding: 12px;
  }
  .tool-server-actions {
    display: flex;
    gap: 6px;
    margin-bottom: 10px;
  }
  .tool-checkboxes {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 240px;
    overflow-y: auto;
  }
  .tool-checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 4px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.1s;
  }
  .tool-checkbox-label:hover {
    background: var(--bg-sidebar);
  }
  .tool-checkbox-label input[type="checkbox"] {
    accent-color: var(--purple-solid);
    flex-shrink: 0;
  }
  .tool-checkbox-name {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-base);
    font-family: 'SF Mono', 'Fira Code', monospace;
    white-space: nowrap;
  }
  .tool-checkbox-desc {
    font-size: 0.75rem;
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .tool-orphaned-section {
    margin-top: 10px;
    border-top: 1px dashed var(--border);
    padding-top: 8px;
  }
  .tool-orphaned-label {
    font-size: 0.72rem;
    color: var(--text-muted);
    display: block;
    margin-bottom: 4px;
  }
  .tool-orphaned {
    opacity: 0.65;
  }
  .tool-orphaned .tool-checkbox-name {
    text-decoration: line-through;
    color: var(--text-muted);
  }

  .action-btn {
    padding: 4px 10px;
    background: var(--bg-sidebar);
    color: var(--text-muted);
    border: 1px solid var(--border);
    border-radius: 5px;
    font-family: inherit;
    font-size: 0.75rem;
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
  }
  .action-btn:hover { background: var(--border); color: var(--text-base); }
  .action-btn-delete { color: #ef4444; border-color: #ef444440; }
  .action-btn-delete:hover { background: #ef444418; color: #ef4444; }
</style>

<script>
  import { api } from '../lib/api.js'
  import ServerModal from '../components/ServerModal.svelte'

  let servers = $state([])
  let mcpServers = $state([])
  let selectedTools = $state([])
  let yamlOutput = $state('')
  let loading = $state(true)
  let showModal = $state(false)
  let editingServer = $state(null)
  let expandedServer = $state(null)
  let serverLoading = $state(false)

  $effect(() => {
    loadAll()
  })

  async function loadAll() {
    loading = true
    await Promise.all([loadTools(), loadMCPServers()])
    loading = false
  }

  async function loadTools() {
    try {
      servers = await api.get('/tools/list')
    } catch (e) {
      console.error('Failed to load tools', e)
    }
  }

  async function loadMCPServers() {
    try {
      mcpServers = await api.get('/tools/servers/list')
    } catch (e) {
      console.error('Failed to load MCP servers', e)
    }
  }

  function toggleTool(qualifiedName) {
    if (selectedTools.includes(qualifiedName)) {
      selectedTools = selectedTools.filter(t => t !== qualifiedName)
    } else {
      selectedTools = [...selectedTools, qualifiedName]
    }
  }

  async function generateYAML() {
    try {
      const result = await api.post('/tools/generate', selectedTools)
      yamlOutput = result.yaml || ''
    } catch (e) {
      console.error('Failed to generate YAML', e)
    }
  }

  function openCreateModal() {
    editingServer = null
    showModal = true
  }

  function openEditModal(srv) {
    editingServer = srv
    showModal = true
  }

  function closeModal() {
    showModal = false
    editingServer = null
  }

  function onServerCreated() {
    closeModal()
    loadMCPServers()
    loadTools()
  }

  function toggleExpand(name) {
    expandedServer = expandedServer === name ? null : name
  }

  async function deleteServer(name) {
    if (!confirm(`Delete MCP server "${name}"? This will disconnect it.`)) return
    try {
      await api.del('/tools/servers/' + name)
      if (expandedServer === name) expandedServer = null
      loadMCPServers()
      loadTools()
    } catch (e) {
      console.error('Failed to delete server', e)
      alert('Failed to delete: ' + e.message)
    }
  }

  async function refreshServer(name) {
    serverLoading = true
    try {
      await api.post('/tools/servers/' + name + '/refresh', {})
      setTimeout(() => {
        loadMCPServers()
        loadTools()
      }, 1500)
    } catch (e) {
      console.error('Failed to refresh', e)
    } finally {
      serverLoading = false
    }
  }

  async function setToolScope(serverName, toolName, scope) {
    try {
      await api.put('/tools/servers/' + serverName + '/tools/' + encodeURIComponent(toolName), { scope })
      loadMCPServers()
    } catch (e) {
      console.error('Failed to set tool scope', e)
    }
  }

  function scopeBadge(scope) {
    if (scope === 'global') return 'global'
    if (scope === 'team') return 'team'
    return 'personal'
  }
</script>

<div class="tools-page">
  <div class="page-header">
    <h2 class="page-title">Tools</h2>
    <span class="page-sub">{selectedTools.length} selected</span>
  </div>

  {#if loading}
    <p style="color:var(--text-muted); padding:20px;">Loading...</p>
  {:else}
    <div class="servers-section">
      <div class="section-header">
        <h3 class="section-title">MCP Servers</h3>
        <button onclick={openCreateModal} class="sb-submit" style="width:auto; padding:7px 16px; font-size:0.82rem;">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
            <path d="M12 5v14M5 12h14"/>
          </svg>
          Register Server
        </button>
      </div>

      {#if mcpServers.length === 0}
        <div class="empty-servers">
          <p>No registered MCP servers. Register one to discover its tools.</p>
        </div>
      {:else}
        <div class="server-cards">
          {#each mcpServers as srv}
            <div class="server-card" class:expanded={expandedServer === srv.name}>
              <div class="server-card-header" onclick={() => toggleExpand(srv.name)} onkeydown={(e) => { if (e.key === 'Enter') toggleExpand(srv.name) }} role="button" tabindex="0">
                <div class="server-card-main">
                  <span class="server-name">{srv.name}</span>
                  <span class="server-url">{srv.url}</span>
                </div>
                <div class="server-card-meta">
                  <span class="status-dot" class:connected={srv.connected} class:disconnected={!srv.connected}></span>
                  <span class="scope-badge scope-{scopeBadge(srv.scope)}">{scopeBadge(srv.scope)}</span>
                  <span class="tool-count">{srv.tools.length} tools</span>
                  <span class="expand-arrow">{expandedServer === srv.name ? '▾' : '▸'}</span>
                </div>
              </div>

              {#if expandedServer === srv.name}
                <div class="server-card-body">
                  <div class="server-card-actions">
                    <button class="action-btn" onclick={() => openEditModal(srv)}>Edit</button>
                    <button class="action-btn action-btn-refresh" onclick={() => refreshServer(srv.name)} disabled={serverLoading}>Refresh</button>
                    <button class="action-btn action-btn-delete" onclick={() => deleteServer(srv.name)}>Delete</button>
                  </div>

                  {#if srv.tools.length > 0}
                    <div class="tools-table-wrap">
                      <table class="tools-table">
                        <thead>
                          <tr>
                            <th>Tool</th>
                            <th>Description</th>
                            <th>Scope</th>
                          </tr>
                        </thead>
                        <tbody>
                          {#each srv.tools as tool}
                            <tr>
                              <td class="tool-name-cell">{tool.name}</td>
                              <td class="tool-desc-cell">{tool.description || '—'}</td>
                              <td class="tool-scope-cell">
                                <select class="scope-select" value={tool.scope} onchange={(e) => setToolScope(srv.name, tool.name, e.target.value)}>
                                  <option value="user">personal</option>
                                  <option value="team">team</option>
                                  <option value="global">global</option>
                                </select>
                                {#if tool.scope_source === 'tool'}
                                  <span class="override-badge">override</span>
                                {:else}
                                  <span class="inherit-badge">inherits</span>
                                {/if}
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {:else}
                    <p class="no-tools">No tools discovered. Click Refresh to re-discover.</p>
                  {/if}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="tools-layout">
      <div class="tool-server-list">
        <div class="section-header">
          <h3 class="section-title">Browse Tools</h3>
        </div>
        {#if servers.length === 0}
          <div class="empty-state">
            <p class="empty-title">No tools available</p>
          </div>
        {:else}
          {#each servers as server}
            <div class="server-group">
              <h3 class="server-name">{server.name}</h3>
              {#each server.tools as tool}
                <label class="tool-item" class:selected={selectedTools.includes(tool.qualified_name)}>
                  <input
                    type="checkbox"
                    checked={selectedTools.includes(tool.qualified_name)}
                    onchange={() => toggleTool(tool.qualified_name)}
                  />
                  <div class="tool-info">
                    <span class="tool-name">{tool.name}</span>
                    {#if tool.description}
                      <span class="tool-desc">{tool.description}</span>
                    {/if}
                  </div>
                </label>
              {/each}
            </div>
          {/each}
        {/if}
      </div>

      <div class="tool-output-panel">
        <div class="output-header">
          <h3 class="output-title">Agent Config</h3>
          <button
            onclick={generateYAML}
            class="sb-submit"
            style="width:auto;"
            disabled={selectedTools.length === 0}
          >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="M7 10l5 5 5-5"/><path d="M12 15V3"/>
            </svg>
            Generate YAML
          </button>
        </div>
        {#if yamlOutput}
          <div class="yaml-output">
            <pre>{yamlOutput}</pre>
          </div>
        {:else}
          <div class="yaml-empty">
            Select tools and click "Generate YAML" to see the agent configuration snippet.
          </div>
        {/if}
        {#if selectedTools.length > 0}
          <div class="selected-tools">
            <div class="sb-section-label" style="padding:12px;">Selected tools ({selectedTools.length})</div>
            {#each selectedTools as t}
              <div class="selected-tool">{t}</div>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>

{#if showModal}
  <ServerModal server={editingServer} onclose={closeModal} oncreated={onServerCreated} />
{/if}

<style>
  .tools-page {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
  }
  .page-header {
    display: flex;
    align-items: baseline;
    gap: 12px;
    margin-bottom: 24px;
  }
  .page-title {
    font-size: 1.2rem;
    font-weight: 700;
    margin: 0;
  }
  .page-sub {
    font-size: 0.8rem;
    color: var(--text-muted);
  }

  .servers-section {
    margin-bottom: 32px;
  }
  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }
  .section-title {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0;
  }

  .empty-servers {
    padding: 24px;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.82rem;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
  }

  .server-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .server-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
    transition: border-color 0.15s;
  }
  .server-card.expanded {
    border-color: oklch(59.1% 0.249 292.7 / 0.5);
  }
  .server-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    cursor: pointer;
    user-select: none;
    transition: background 0.1s;
  }
  .server-card-header:hover {
    background: rgba(255,255,255,0.02);
  }
  .server-card-main {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .server-name {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .server-url {
    font-size: 0.76rem;
    color: var(--text-muted);
    font-family: 'SF Mono', 'Fira Code', monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .server-card-meta {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }
  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .status-dot.connected {
    background: #22c55e;
    box-shadow: 0 0 6px rgba(34,197,94,0.5);
  }
  .status-dot.disconnected {
    background: #71717a;
  }
  .scope-badge {
    font-size: 0.68rem;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .scope-badge.scope-personal {
    background: oklch(55% 0.25 292.7 / 0.12);
    color: var(--purple);
  }
  .scope-badge.scope-team {
    background: oklch(55% 0.2 150 / 0.12);
    color: oklch(50% 0.18 150);
  }
  .scope-badge.scope-global {
    background: oklch(55% 0.2 50 / 0.12);
    color: oklch(55% 0.2 50);
  }
  .tool-count {
    font-size: 0.76rem;
    color: #52525b;
  }
  .expand-arrow {
    font-size: 0.85rem;
    color: #52525b;
    width: 16px;
    text-align: center;
  }

  .server-card-body {
    border-top: 1px solid var(--border);
    padding: 12px 16px;
  }
  .server-card-actions {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }
  .action-btn {
    padding: 4px 10px;
    background: none;
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.75rem;
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }
  .action-btn:hover {
    background: var(--bg-sidebar);
    color: var(--text-base);
  }
  .action-btn-delete {
    border-color: var(--red-border);
    color: var(--red-text);
  }
  .action-btn-delete:hover { background: var(--red-dim); }
  .action-btn:disabled { opacity: 0.45; cursor: not-allowed; }

  .tools-table-wrap {
    border: 1px solid var(--border);
    border-radius: 8px;
    overflow: hidden;
  }
  .tools-table {
    width: 100%;
    border-collapse: collapse;
  }
  .tools-table th {
    text-align: left;
    padding: 8px 12px;
    font-size: 0.7rem;
    font-weight: 600;
    color: #52525b;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--border);
    background: var(--bg-sidebar);
  }
  .tools-table td {
    padding: 8px 12px;
    font-size: 0.78rem;
    border-bottom: 1px solid var(--border);
    color: var(--text-muted);
  }
  .tools-table tr:last-child td { border-bottom: none; }
  .tool-name-cell {
    color: var(--text-base) !important;
    font-weight: 500;
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.76rem !important;
  }
  .tool-desc-cell {
    max-width: 300px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .tool-scope-cell {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .scope-select {
    background: var(--bg-sidebar);
    border: 1px solid var(--border);
    border-radius: 5px;
    color: var(--text-base);
    font-family: inherit;
    font-size: 0.72rem;
    padding: 2px 6px;
    outline: none;
    cursor: pointer;
  }
  .override-badge {
    font-size: 0.62rem;
    font-weight: 600;
    padding: 1px 5px;
    border-radius: 3px;
    background: oklch(55% 0.25 292.7 / 0.12);
    color: var(--purple);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }
  .inherit-badge {
    font-size: 0.62rem;
    font-weight: 600;
    padding: 1px 5px;
    border-radius: 3px;
    background: rgba(0,0,0,0.15);
    color: #52525b;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }
  .no-tools {
    text-align: center;
    color: #52525b;
    font-size: 0.78rem;
    padding: 16px;
    margin: 0;
  }

  .tools-layout {
    display: grid;
    grid-template-columns: 1fr 320px;
    gap: 24px;
    align-items: start;
  }
  .tool-server-list {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .server-group {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
  }
  .server-group .server-name {
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--text-muted);
    padding: 12px 16px;
    margin: 0;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--border);
  }
  .tool-item {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 16px;
    cursor: pointer;
    transition: background 0.12s;
    border-bottom: 1px solid var(--border);
  }
  .tool-item:last-child { border-bottom: none; }
  .tool-item:hover { background: rgba(255,255,255,0.02); }
  .tool-item.selected {
    background: var(--purple-dim);
  }
  .tool-item input[type="checkbox"] {
    margin-top: 2px;
    accent-color: var(--purple-solid);
    flex-shrink: 0;
  }
  .tool-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .tool-info .tool-name {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .tool-info .tool-desc {
    font-size: 0.78rem;
    color: var(--text-muted);
    line-height: 1.4;
  }

  .tool-output-panel {
    position: sticky;
    top: 24px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
  }
  .output-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
  }
  .output-title {
    font-size: 0.85rem;
    font-weight: 600;
    margin: 0;
  }
  .yaml-output {
    padding: 16px;
  }
  .yaml-output pre {
    background: rgba(0,0,0,0.3);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 14px;
    font-size: 0.8rem;
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: var(--text-base);
    overflow-x: auto;
    margin: 0;
  }
  .yaml-empty {
    padding: 24px 16px;
    color: #52525b;
    font-size: 0.82rem;
    text-align: center;
  }
  .selected-tool {
    padding: 6px 12px;
    font-size: 0.78rem;
    color: var(--text-muted);
    border-bottom: 1px solid var(--border);
  }
  .selected-tool:last-child { border-bottom: none; }
  .sb-section-label {
    font-size: 0.72rem;
    font-weight: 600;
    color: #52525b;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--border);
  }
</style>

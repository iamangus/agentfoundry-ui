<script>
  import { api } from '../lib/api.js'
  import ServerModal from '../components/ServerModal.svelte'

  let mcpServers = $state([])
  let loading = $state(true)
  let showModal = $state(false)
  let editingServer = $state(null)
  let expandedServer = $state(null)
  let serverLoading = $state(false)
  let refreshTimer = $state(null)

  $effect(() => {
    loadMCPServers()
    return () => {
      if (refreshTimer) clearTimeout(refreshTimer)
    }
  })

  async function loadMCPServers() {
    loading = true
    try {
      mcpServers = await api.get('/tools/servers/list')
    } catch (e) {
      console.error('Failed to load MCP servers', e)
    } finally {
      loading = false
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
  }

  function toggleExpand(id) {
    expandedServer = expandedServer === id ? null : id
  }

  async function deleteServer(id) {
    if (!confirm(`Delete MCP server "${mcpServers.find(s => s.id === id)?.name}"? This will disconnect it.`)) return
    try {
      await api.del('/tools/servers/' + encodeURIComponent(id))
      if (expandedServer === id) expandedServer = null
      loadMCPServers()
    } catch (e) {
      console.error('Failed to delete server', e)
      alert('Failed to delete: ' + e.message)
    }
  }

  async function refreshServer(id) {
    serverLoading = true
    try {
      await api.post('/tools/servers/' + encodeURIComponent(id) + '/refresh', {})
      if (refreshTimer) clearTimeout(refreshTimer)
      refreshTimer = setTimeout(() => {
        refreshTimer = null
        loadMCPServers()
      }, 1500)
    } catch (e) {
      console.error('Failed to refresh', e)
    } finally {
      serverLoading = false
    }
  }

  async function setToolScope(serverID, toolName, scope) {
    try {
      await api.put('/tools/servers/' + encodeURIComponent(serverID) + '/tools/' + encodeURIComponent(toolName), { scope })
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
    <div class="section-header">
      <h3 class="section-title">MCP Servers</h3>
      <button onclick={openCreateModal} class="sb-submit" style="width:auto; padding:8px 18px; font-size:0.82rem;">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
          <path d="M12 5v14M5 12h14"/>
        </svg>
        Register Server
      </button>
    </div>
  </div>

  {#if loading}
    <p style="color:var(--text-muted); padding:20px;">Loading...</p>
  {:else if mcpServers.length === 0}
    <div class="empty-state">
      <p class="empty-title">No registered MCP servers</p>
      <p class="empty-desc">Register an MCP server to discover its tools. Tools can then be assigned to agents from the agent create/edit form.</p>
    </div>
  {:else}
    <div class="server-cards">
      {#each mcpServers as srv}
        <div class="server-card" class:expanded={expandedServer === srv.id}>
          <div class="server-card-header" onclick={() => toggleExpand(srv.id)} onkeydown={(e) => { if (e.key === 'Enter') toggleExpand(srv.id) }} role="button" tabindex="0">
            <div class="server-card-main">
              <span class="server-name">{srv.name}</span>
              <span class="server-url">{srv.url}</span>
            </div>
            <div class="server-card-meta">
              <span class="status-dot" class:connected={srv.connected} class:disconnected={!srv.connected}></span>
              <span class="scope-badge scope-{scopeBadge(srv.scope)}">{scopeBadge(srv.scope)}</span>
              <span class="tool-count">{srv.tools.length} tools</span>
              <span class="expand-arrow">{expandedServer === srv.id ? '▾' : '▸'}</span>
            </div>
          </div>

          {#if expandedServer === srv.id}
            <div class="server-card-body">
              <div class="server-card-actions">
                <button class="action-btn" onclick={() => openEditModal(srv)}>Edit</button>
                <button class="action-btn action-btn-refresh" onclick={() => refreshServer(srv.id)} disabled={serverLoading}>Refresh</button>
                <button class="action-btn action-btn-delete" onclick={() => deleteServer(srv.id)}>Delete</button>
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
                            <select class="scope-select" value={tool.scope} onchange={(e) => setToolScope(srv.id, tool.name, e.target.value)}>
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

{#if showModal}
  <ServerModal server={editingServer} onclose={closeModal} oncreated={onServerCreated} />
{/if}

<style>
  .tools-page {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
  }
  .section-title {
    font-size: 1.05rem;
    font-weight: 700;
    color: var(--text-base);
    margin: 0;
  }

  .empty-state {
    padding: 40px 24px;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.82rem;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
  }
  .empty-title {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-base);
    margin: 0 0 8px 0;
  }
  .empty-desc {
    margin: 0;
    max-width: 480px;
    margin-inline: auto;
    line-height: 1.5;
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
</style>

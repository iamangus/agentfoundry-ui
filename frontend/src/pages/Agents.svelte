<script>
  import { api } from '../lib/api.js'
  import AgentForm from './AgentForm.svelte'

  let agents = $state([])
  let groups = $state({ personal: [], team: [], global: [] })
  let loading = $state(true)
  let editingAgent = $state(null)
  let creatingNew = $state(false)
  let showVersions = $state(null)
  let versions = $state([])
  let versionsLoading = $state(false)
  let formServers = $state([])
  let formProviders = $state([])
  let agentList = $state([])

  $effect(() => {
    loadAgents()
    loadFormData()
  })

  async function loadAgents() {
    try {
      agents = await api.get('/agents/list')
      groups = groupByScope(agents)
      agentList = agents || []
    } catch (e) {
      console.error('Failed to load agents', e)
    } finally {
      loading = false
    }
  }

  async function loadFormData() {
    try {
      const [servers, providers] = await Promise.all([
        api.get('/tools/servers/list'),
        api.get('/api/v1/providers')
      ])
      formServers = servers || []
      formProviders = providers || []
    } catch (e) {
      console.error('Failed to load form data', e)
    }
  }

  function groupByScope(list) {
    const sorted = [...list].sort((a, b) => a.name.localeCompare(b.name))
    const result = { personal: [], team: [], global: [] }
    for (const a of sorted) {
      if (a.scope === 'team') result.team.push(a)
      else if (a.scope === 'global') result.global.push(a)
      else result.personal.push(a)
    }
    return result
  }

  function startEdit(agent) {
    editingAgent = JSON.parse(JSON.stringify(agent))
    creatingNew = false
  }

  function startCreate() {
    editingAgent = { kind: 'agent', name: '', description: '', model: '', system_prompt: '', tools: [], max_turns: 0, max_concurrent_tools: 0, force_json: false, scope: '', team: '', structured_output: null, memory_enabled: false, memory_search_agent_id: '', memory_ingest_agent_id: '', pre_inference_processors: [], handoff_to: '', handoffs: [] }
    creatingNew = true
  }

  function cancelEdit() {
    editingAgent = null
    creatingNew = false
  }

  async function saveAgent(formData) {
    try {
      if (creatingNew) {
        await api.post('/agents/form', formData)
      } else {
        await api.put('/agents/' + formData.originalName, formData)
      }
      editingAgent = null
      creatingNew = false
      await loadAgents()
    } catch (e) {
      console.error('Failed to save agent', e)
      alert('Failed to save: ' + e.message)
    }
  }

  async function cloneAgent(name) {
    try {
      await api.post('/agents/' + encodeURIComponent(name) + '/clone', {})
      await loadAgents()
    } catch (e) {
      console.error('Failed to clone agent', e)
      alert('Failed to clone: ' + e.message)
    }
  }

  async function deleteAgent(name) {
    if (!confirm('Delete agent "' + name + '"?')) return
    try {
      await api.del('/agents/' + encodeURIComponent(name))
      await loadAgents()
    } catch (e) {
      console.error('Failed to delete agent', e)
    }
  }

  async function loadVersions(agentName) {
    showVersions = agentName
    versionsLoading = true
    try {
      const resp = await api.get('/agents/' + encodeURIComponent(agentName) + '/versions')
      versions = (resp.versions || []).sort((a, b) => new Date(b.last_modified) - new Date(a.last_modified))
    } catch (e) {
      console.error('Failed to load versions', e)
      versions = []
    } finally {
      versionsLoading = false
    }
  }

  function closeVersions() {
    showVersions = null
    versions = []
  }

  async function rollbackVersion(agentName, versionId) {
    if (!confirm('Restore this version? This will create a new save with this version\'s content.')) return
    try {
      await api.post('/agents/' + encodeURIComponent(agentName) + '/rollback?version_id=' + encodeURIComponent(versionId), null)
      await loadAgents()
      closeVersions()
    } catch (e) {
      console.error('Failed to rollback', e)
      alert('Rollback failed: ' + e.message)
    }
  }
</script>

{#if editingAgent}
  <AgentForm
    def={editingAgent}
    isNew={creatingNew}
    onsave={saveAgent}
    oncancel={cancelEdit}
    servers={formServers}
    providers={formProviders}
    agents={agentList}
  />
{:else}
  <div class="agents-page">
    <div class="page-header">
      <h2 class="page-title">Agents</h2>
      <button onclick={startCreate} class="sb-submit" style="width:auto;">
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" style="width:14px;height:14px;">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
        </svg>
        New Agent
      </button>
    </div>

    {#if loading}
      <p style="color:var(--text-muted); padding:20px;">Loading...</p>
    {:else}
      {#each Object.entries(groups) as [scope, list]}
        {#if list.length > 0}
          <div class="scope-section">
            <h3 class="scope-label">{scope}</h3>
            <div class="agent-grid">
              {#each list as agent}
                <div class="agent-card">
                  <div class="agent-card-header">
                    <span class="agent-name">{agent.name}</span>
                    <div class="agent-actions">
                      <button class="action-btn" onclick={() => loadVersions(agent.name)} title="Versions">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
                          <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
                        </svg>
                      </button>
                      <button class="action-btn" onclick={() => cloneAgent(agent.name)} title="Clone">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
                          <rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                        </svg>
                      </button>
                      <button class="action-btn" onclick={() => startEdit(agent)} title="Edit">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
                          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                        </svg>
                      </button>
                      <button class="action-btn danger" onclick={() => deleteAgent(agent.name)} title="Delete">
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
                          <path d="M3 6h18"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
                        </svg>
                      </button>
                    </div>
                  </div>
                  {#if agent.description}
                    <p class="agent-description">{agent.description}</p>
                  {/if}
                  <div class="agent-meta">
                    {#if agent.agent_id}<span class="meta-tag id-tag" title="Agent ID for API runs">{agent.agent_id}</span>{/if}
                    {#if agent.model}<span class="meta-tag">model: {agent.model}</span>{/if}
                    {#if agent.tools && agent.tools.length > 0}<span class="meta-tag">{agent.tools.length} tools</span>{/if}
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      {/each}

      {#if Object.values(groups).every(g => g.length === 0)}
        <div class="empty-state" style="padding-top:60px;">
          <div class="empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="8" r="4"/><path d="M4 20c0-4 3.6-7 8-7s8 3 8 7"/>
            </svg>
          </div>
          <p class="empty-title">No agents</p>
          <p class="empty-sub">Create your first agent to get started.</p>
        </div>
      {/if}
    {/if}
    {#if showVersions}
      <div class="modal-overlay" onclick={closeVersions}>
        <div class="version-modal" onclick={(e) => e.stopPropagation()}>
          <div class="version-modal-header">
            <h3>Revisions — {showVersions}</h3>
            <button class="action-btn" onclick={closeVersions} title="Close">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
                <path d="M18 6 6 18"/><path d="m6 6 12 12"/>
              </svg>
            </button>
          </div>
          <div class="version-list">
            {#if versionsLoading}
              <p class="version-loading">Loading...</p>
            {:else if versions.length === 0}
              <p class="version-empty">No revision history yet.</p>
            {:else}
              {#each versions as v}
                <div class="version-row" class:is-current={v.is_latest}>
                  <div class="version-info">
                    <span class="version-time">{new Date(v.last_modified).toLocaleString()}</span>
                    <span class="version-id">{v.version_id.slice(0, 8)}</span>
                    {#if v.is_latest}
                      <span class="version-current-badge">current</span>
                    {/if}
                  </div>
                  {#if !v.is_latest}
                    <button class="version-restore-btn" onclick={() => rollbackVersion(showVersions, v.version_id)}>
                      Restore
                    </button>
                  {/if}
                </div>
              {/each}
            {/if}
          </div>
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  .agents-page {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
  }
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 24px;
  }
  .page-title {
    font-size: 1.2rem;
    font-weight: 700;
    margin: 0;
  }
  .scope-section {
    margin-bottom: 28px;
  }
  .scope-label {
    font-size: 0.65rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: #52525b;
    margin: 0 0 12px 4px;
  }
  .agent-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 12px;
  }
  .agent-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 16px;
    transition: border-color 0.15s;
  }
  .agent-card:hover {
    border-color: oklch(59.1% 0.249 292.7 / 0.3);
  }
  .agent-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }
  .agent-name {
    font-size: 0.9rem;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 200px;
  }
  .agent-actions {
    display: flex;
    gap: 4px;
    opacity: 0;
    transition: opacity 0.15s;
  }
  .agent-card:hover .agent-actions { opacity: 1; }
  .action-btn {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-base);
    color: var(--text-muted);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.12s, color 0.12s;
  }
  .action-btn:hover { background: var(--purple-dim); color: var(--text-base); }
  .action-btn.danger:hover { background: var(--red-dim); color: var(--red-text); border-color: var(--red-border); }
  .agent-description {
    font-size: 0.82rem;
    color: var(--text-muted);
    margin: 0 0 8px;
    line-height: 1.5;
  }
  .agent-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .meta-tag {
    font-size: 0.7rem;
    font-weight: 500;
    background: var(--purple-dim);
    color: oklch(70% 0.2 292.0);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.25);
    padding: 2px 8px;
    border-radius: 12px;
  }
  .id-tag {
    font-family: monospace;
    font-size: 0.65rem;
    user-select: all;
    cursor: copy;
  }
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .version-modal {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 12px;
    width: 440px;
    max-height: 480px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .version-modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px 12px;
    border-bottom: 1px solid var(--border);
  }
  .version-modal-header h3 {
    font-size: 0.95rem;
    font-weight: 600;
    margin: 0;
  }
  .version-list {
    overflow-y: auto;
    padding: 8px;
  }
  .version-loading, .version-empty {
    color: var(--text-muted);
    font-size: 0.82rem;
    padding: 20px;
    text-align: center;
  }
  .version-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    border-radius: 8px;
    transition: background 0.12s;
  }
  .version-row:hover { background: var(--bg-base); }
  .version-row.is-current { background: var(--purple-dim); }
  .version-info {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 0.78rem;
  }
  .version-time { color: var(--text-muted); }
  .version-id {
    font-family: monospace;
    font-size: 0.7rem;
    color: oklch(70% 0.2 292.0);
  }
  .version-current-badge {
    font-size: 0.65rem;
    font-weight: 600;
    background: oklch(59.1% 0.249 292.7 / 0.2);
    color: oklch(70% 0.2 292.0);
    padding: 1px 8px;
    border-radius: 10px;
  }
  .version-restore-btn {
    font-size: 0.72rem;
    font-weight: 500;
    background: var(--bg-base);
    color: var(--text-muted);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 3px 10px;
    cursor: pointer;
    transition: background 0.12s, color 0.12s;
  }
  .version-restore-btn:hover { background: var(--purple-dim); color: var(--text-base); }
</style>

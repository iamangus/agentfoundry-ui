<script>
  import { api } from '../lib/api.js'
  import { teams, loadTeams } from '../lib/stores.js'

  let { server = null, onclose, oncreated } = $props()

  let isEdit = $derived(!!server)
  let saving = $state(false)
  let error = $state('')

  let formName = $state('')
  let formUrl = $state('')
  let formTransport = $state('sse')
  let formHeaders = $state('')
  let formScope = $state('user')
  let formTeam = $state('')
  let toolOverrides = $state({})

  $effect(() => {
    loadTeams()
    if (server) {
      formName = server.name
      formUrl = server.url
      formTransport = server.transport || 'sse'
      formHeaders = server.headers ? Object.entries(server.headers).map(([k, v]) => `${k}: ${v}`).join('\n') : ''
      formScope = server.scope || 'user'
      formTeam = server.team || ''
      if (server.tool_overrides) {
        try {
          toolOverrides = typeof server.tool_overrides === 'string' ? JSON.parse(server.tool_overrides) : server.tool_overrides
        } catch {}
      }
    }
  })

  function close() {
    onclose()
  }

  function onkeydown(e) {
    if (e.key === 'Escape') close()
  }

  function parseHeaders() {
    const headers = {}
    if (formHeaders.trim()) {
      for (const line of formHeaders.trim().split('\n')) {
        const idx = line.indexOf(':')
        if (idx > 0) {
          const key = line.substring(0, idx).trim()
          const val = line.substring(idx + 1).trim()
          if (key) headers[key] = val
        }
      }
    }
    return headers
  }

  function getOverrideKeys() {
    return Object.keys(toolOverrides)
  }

  function getOverrideParams(key) {
    return toolOverrides[key] ? Object.keys(toolOverrides[key]) : []
  }

  function addOverride(target) {
    const ov = { ...toolOverrides }
    if (!ov[target]) ov[target] = {}
    ov[target] = { ...ov[target] }
    ov[target][''] = { value: '', force: false }
    toolOverrides = ov
  }

  function removeOverride(target, param) {
    const ov = { ...toolOverrides }
    if (ov[target]) {
      ov[target] = { ...ov[target] }
      delete ov[target][param]
      if (Object.keys(ov[target]).length === 0) delete ov[target]
    }
    toolOverrides = ov
  }

  function updateOverrideParam(target, oldParam, newParam) {
    const ov = { ...toolOverrides }
    if (ov[target] && ov[target][oldParam] && newParam !== oldParam) {
      ov[target] = { ...ov[target] }
      ov[target][newParam] = ov[target][oldParam]
      delete ov[target][oldParam]
      toolOverrides = ov
    }
  }

  function updateOverrideValue(target, param, val) {
    const ov = { ...toolOverrides }
    if (!ov[target]) ov[target] = {}
    ov[target] = { ...ov[target] }
    ov[target][param] = { ...(ov[target][param] || {}), value: val }
    toolOverrides = ov
  }

  function updateOverrideForce(target, param, force) {
    const ov = { ...toolOverrides }
    if (!ov[target]) ov[target] = {}
    ov[target] = { ...ov[target] }
    ov[target][param] = { ...(ov[target][param] || {}), force }
    toolOverrides = ov
  }

  function buildBody() {
    return {
      name: formName.trim(),
      url: formUrl.trim(),
      transport: formTransport,
      headers: parseHeaders(),
      scope: formScope,
      team: formScope === 'team' ? formTeam.trim() : '',
      tool_overrides: JSON.stringify(toolOverrides)
    }
  }

  async function save() {
    if (!formName.trim() || !formUrl.trim()) {
      error = 'Name and URL are required'
      return
    }
    saving = true
    error = ''
    try {
      await api.post('/tools/servers', buildBody())
      oncreated()
    } catch (e) {
      error = e.message || 'Failed to save'
    } finally {
      saving = false
    }
  }

  async function update() {
    if (!formName.trim() || !formUrl.trim()) {
      error = 'Name and URL are required'
      return
    }
    saving = true
    error = ''
    try {
      await api.put('/tools/servers/' + server.id, buildBody())
      oncreated()
    } catch (e) {
      error = e.message || 'Failed to update'
    } finally {
      saving = false
    }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-backdrop" onclick={close} onkeydown={onkeydown}>
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_interactive_supports_focus -->
  <div class="modal-card" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex="-1">
    <h3 class="modal-title">{isEdit ? 'Edit Server' : 'Register MCP Server'}</h3>

    {#if error}
      <div class="form-error">{error}</div>
    {/if}

    <label class="modal-label">Name</label>
    <input class="sb-input" bind:value={formName} placeholder="my-mcp-server" />

    <label class="modal-label">URL</label>
    <input class="sb-input" bind:value={formUrl} placeholder="http://localhost:8080/sse" />

    <label class="modal-label">Transport</label>
    <select class="sb-input" bind:value={formTransport}>
      <option value="sse">SSE</option>
      <option value="streamable-http">Streamable HTTP</option>
    </select>

    <label class="modal-label">Headers (one <code>Key: Value</code> per line)</label>
    <textarea class="sb-input sb-textarea" bind:value={formHeaders} placeholder="X-API-Key: your-key" rows="3"></textarea>

    <label class="modal-label">Scope</label>
    <select class="sb-input" bind:value={formScope}>
      <option value="user">Personal</option>
      <option value="team">Team</option>
      <option value="global">Global</option>
    </select>

    {#if formScope === 'team'}
      <label class="modal-label">Team name</label>
      <select class="sb-input" bind:value={formTeam}>
        <option value="">-- select team --</option>
        {#each $teams as t}
          <option value={t}>{t}</option>
        {/each}
      </select>
    {/if}

    <label class="modal-label">Tool Parameter Overrides</label>
    <p class="modal-hint">Override input values for tool parameters. Use <code>*</code> to target all tools, or a specific tool name. Value supports <code>$&#123;agentID&#125;</code>, <code>$&#123;agentName&#125;</code>, <code>$&#123;userSubject&#125;</code>.</p>
    {#each getOverrideKeys() as target}
      <div class="override-group">
        <div class="override-group-header">
          <span class="override-target-name">{target}</span>
          <button class="override-rm-group-btn" onclick={() => { const ov = { ...toolOverrides }; delete ov[target]; toolOverrides = ov }} title="Remove all for {target}">&times;</button>
        </div>
        {#each getOverrideParams(target) as param}
          <div class="override-row">
            <input class="sb-input override-param" placeholder="param" value={param} onblur={(e) => updateOverrideParam(target, param, e.target.value.trim())} oninput={null} />
            <input class="sb-input override-val" placeholder='${agentID}' value={toolOverrides[target][param]?.value || ''} oninput={(e) => updateOverrideValue(target, param, e.target.value)} />
            <label class="override-force">
              <input type="checkbox" checked={toolOverrides[target][param]?.force || false} onchange={(e) => updateOverrideForce(target, param, e.target.checked)} />
              force
            </label>
            <button class="override-rm-btn" onclick={() => removeOverride(target, param)} title="Remove">&times;</button>
          </div>
        {/each}
        <button class="override-add-btn" onclick={() => addOverride(target)}>+ Param</button>
      </div>
    {/each}
    <button class="override-add-group-btn" onclick={() => addOverride('*')}>+ Add override group</button>

    <div class="modal-actions">
      <button onclick={close} class="modal-btn modal-btn-cancel">Cancel</button>
      <button onclick={() => isEdit ? update() : save()} class="modal-btn modal-btn-create" disabled={saving}>
        {saving ? 'Saving...' : isEdit ? 'Update' : 'Register'}
      </button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.55);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .modal-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 24px;
    width: 440px;
    max-width: 90vw;
    max-height: 85vh;
    overflow-y: auto;
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
  }
  .modal-title {
    font-size: 1.05rem;
    font-weight: 700;
    margin: 0 0 16px 0;
  }
  .modal-label {
    display: block;
    font-size: 0.78rem;
    font-weight: 500;
    color: var(--text-muted);
    margin: 12px 0 6px 0;
  }
  .modal-label:first-of-type {
    margin-top: 0;
  }
  .modal-label code {
    font-size: 0.72rem;
    background: rgba(124,58,237,0.12);
    padding: 1px 4px;
    border-radius: 3px;
  }
  .sb-textarea {
    resize: vertical;
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.8rem;
  }
  .form-error {
    padding: 8px 12px;
    background: var(--red-dim);
    border: 1px solid var(--red-border);
    border-radius: 8px;
    color: var(--red-text);
    font-size: 0.82rem;
    margin-bottom: 12px;
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 20px;
  }
  .modal-btn {
    padding: 7px 16px;
    border-radius: 8px;
    font-family: inherit;
    font-size: 0.82rem;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.12s, opacity 0.12s;
    border: 1px solid var(--border);
    background: var(--bg-card);
    color: var(--text-base);
  }
  .modal-btn-cancel:hover { background: var(--bg-sidebar); }
  .modal-btn-create {
    background: var(--purple-solid);
    border-color: transparent;
    color: #fff;
  }
  .modal-btn-create:hover { opacity: 0.85; }
  .modal-btn-create:disabled { opacity: 0.45; cursor: not-allowed; }

  .modal-hint {
    font-size: 0.72rem;
    color: var(--text-muted);
    margin: 2px 0 8px 0;
    line-height: 1.4;
  }
  .modal-hint code {
    font-size: 0.68rem;
    background: rgba(124,58,237,0.12);
    padding: 1px 4px;
    border-radius: 3px;
  }
  .override-group {
    background: var(--bg-sidebar);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 8px;
    margin-bottom: 8px;
  }
  .override-group-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
  }
  .override-target-name {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--purple-solid);
    font-family: 'SF Mono', 'Fira Code', monospace;
  }
  .override-rm-group-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 0.95rem;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }
  .override-rm-group-btn:hover { color: #ef4444; }
  .override-row {
    display: flex;
    gap: 4px;
    align-items: center;
    margin-bottom: 4px;
  }
  .override-param {
    width: 100px;
    font-size: 0.7rem;
    padding: 2px 5px;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }
  .override-val {
    flex: 1;
    font-size: 0.7rem;
    padding: 2px 5px;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }
  .override-force {
    display: flex;
    align-items: center;
    gap: 2px;
    font-size: 0.68rem;
    color: var(--text-muted);
    white-space: nowrap;
    cursor: pointer;
  }
  .override-force input[type="checkbox"] {
    accent-color: var(--purple-solid);
  }
  .override-rm-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 0.95rem;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }
  .override-rm-btn:hover { color: #ef4444; }
  .override-add-btn {
    background: var(--bg-card);
    color: var(--purple-solid);
    border: 1px solid var(--purple-soft);
    border-radius: 4px;
    font-family: inherit;
    font-size: 0.68rem;
    cursor: pointer;
    padding: 2px 8px;
    margin-top: 2px;
  }
  .override-add-btn:hover { background: var(--purple-soft); }
  .override-add-group-btn {
    background: var(--bg-sidebar);
    color: var(--text-muted);
    border: 1px dashed var(--border);
    border-radius: 6px;
    font-family: inherit;
    font-size: 0.72rem;
    cursor: pointer;
    padding: 6px 12px;
    width: 100%;
    margin-bottom: 8px;
  }
  .override-add-group-btn:hover { color: var(--text-base); border-color: var(--text-muted); }
</style>

<script>
  import { api } from '../lib/api.js'
  import { teams, loadTeams } from '../lib/stores.js'

  let { provider = null, onclose, oncreated } = $props()

  let isEdit = $derived(!!provider)
  let saving = $state(false)
  let error = $state('')

  let formName = $state('')
  let formProviderType = $state('custom')
  let formBaseURL = $state('')
  let formAPIKey = $state('')
  let formDefaultModel = $state('')
  let formSchemaValidation = $state(true)
  let formHeaders = $state('')
  let formScope = $state('user')
  let formTeam = $state('')


  $effect(() => {
    loadTeams()
    if (provider) {
      formName = provider.name
      formProviderType = provider.provider_type || 'custom'
      formBaseURL = provider.base_url || ''
      formAPIKey = provider.api_key || ''
      formDefaultModel = provider.default_model || ''
      formSchemaValidation = provider.schema_validation !== false
      formHeaders = provider.headers ? Object.entries(provider.headers).map(([k, v]) => `${k}: ${v}`).join('\n') : ''
      formScope = provider.scope || 'user'
      formTeam = provider.team || ''
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

  async function save() {
    if (!formName.trim() || !formBaseURL.trim()) {
      error = 'Name and Base URL are required'
      return
    }
    saving = true
    error = ''
    try {
      const body = {
        name: formName.trim(),
        provider_type: formProviderType,
        base_url: formBaseURL.trim(),
        api_key: formAPIKey.trim(),
        default_model: formDefaultModel.trim(),
        schema_validation: formSchemaValidation,
        headers: parseHeaders(),
        scope: formScope,
        team: formScope === 'team' ? formTeam.trim() : ''
      }
      await api.post('/api/v1/providers', body)
      oncreated()
    } catch (e) {
      error = e.message || 'Failed to save'
    } finally {
      saving = false
    }
  }

  async function update() {
    if (!formName.trim() || !formBaseURL.trim()) {
      error = 'Name and Base URL are required'
      return
    }
    saving = true
    error = ''
    try {
      const body = {
        name: formName.trim(),
        provider_type: formProviderType,
        base_url: formBaseURL.trim(),
        api_key: formAPIKey.trim(),
        default_model: formDefaultModel.trim(),
        schema_validation: formSchemaValidation,
        headers: parseHeaders(),
        scope: formScope,
        team: formScope === 'team' ? formTeam.trim() : ''
      }
      await api.put('/api/v1/providers/' + encodeURIComponent(provider.id), body)
      oncreated()
    } catch (e) {
      error = e.message || 'Failed to update'
    } finally {
      saving = false
    }
  }
</script>

<div class="modal-backdrop" onclick={close} onkeydown={onkeydown}>
  <div class="modal-card" onclick={(e) => e.stopPropagation()} role="dialog" aria-modal="true" tabindex="-1">
    <h3 class="modal-title">{isEdit ? 'Edit Provider' : 'Add Inference Provider'}</h3>

    {#if error}
      <div class="form-error">{error}</div>
    {/if}

    <label class="modal-label">Name</label>
    <input class="sb-input" bind:value={formName} placeholder="my-provider" />

    <label class="modal-label">Provider Type</label>
    <select class="sb-input" bind:value={formProviderType}>
      <option value="openai">OpenAI</option>
      <option value="anthropic">Anthropic</option>
      <option value="openrouter">OpenRouter</option>
      <option value="ollama">Ollama</option>
      <option value="llama-server">llama-server</option>
      <option value="custom">Custom</option>
    </select>

    <label class="modal-label">Base URL</label>
    <input class="sb-input" bind:value={formBaseURL} placeholder="https://api.openai.com/v1" />

    <label class="modal-label">API Key</label>
    <input class="sb-input" bind:value={formAPIKey} placeholder={isEdit ? '(unchanged)' : 'sk-...'} type="password" />

    <label class="modal-label">Default Model</label>
    <input class="sb-input" bind:value={formDefaultModel} placeholder="gpt-4o" />

    <label class="modal-label" style="display:flex;align-items:center;gap:8px;cursor:pointer;">
      <input type="checkbox" bind:checked={formSchemaValidation} style="width:14px;height:14px;" />
      Schema Validation
    </label>

    <label class="modal-label">Headers (one <code>Key: Value</code> per line)</label>
    <textarea class="sb-input sb-textarea" bind:value={formHeaders} placeholder="X-Custom-Header: value" rows="3"></textarea>

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

    <div class="modal-actions">
      <button onclick={close} class="modal-btn modal-btn-cancel">Cancel</button>
      <button onclick={() => isEdit ? update() : save()} class="modal-btn modal-btn-create" disabled={saving}>
        {saving ? 'Saving...' : isEdit ? 'Update' : 'Add Provider'}
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
</style>

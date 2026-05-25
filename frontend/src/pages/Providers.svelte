<script>
  import { api } from '../lib/api.js'
  import ProviderModal from '../components/ProviderModal.svelte'

  let providers = $state([])
  let loading = $state(true)
  let showModal = $state(false)
  let editingProvider = $state(null)
  let expandedProvider = $state(null)

  $effect(() => {
    loadProviders()
  })

  async function loadProviders() {
    loading = true
    try {
      providers = await api.get('/api/v1/providers')
    } catch (e) {
      console.error('Failed to load providers', e)
    } finally {
      loading = false
    }
  }

  function openCreateModal() {
    editingProvider = null
    showModal = true
  }

  function openEditModal(prov) {
    editingProvider = prov
    showModal = true
  }

  function closeModal() {
    showModal = false
    editingProvider = null
  }

  function onProviderCreated() {
    closeModal()
    loadProviders()
  }

  function toggleExpand(name) {
    expandedProvider = expandedProvider === name ? null : name
  }

  async function deleteProvider(name) {
    if (!confirm(`Delete provider "${name}"?`)) return
    try {
      await api.del('/api/v1/providers/' + encodeURIComponent(name))
      if (expandedProvider === name) expandedProvider = null
      loadProviders()
    } catch (e) {
      console.error('Failed to delete provider', e)
      alert('Failed to delete: ' + e.message)
    }
  }

  function scopeBadge(scope) {
    if (scope === 'global') return 'global'
    if (scope === 'team') return 'team'
    return 'personal'
  }

  function typeBadge(t) {
    const labels = { openai: 'OpenAI', anthropic: 'Anthropic', openrouter: 'OpenRouter', ollama: 'Ollama', custom: 'Custom' }
    return labels[t] || t
  }
</script>

<div class="providers-page">
  <div class="page-header">
    <div class="section-header">
      <h3 class="section-title">Inference Providers</h3>
      <button onclick={openCreateModal} class="sb-submit" style="width:auto; padding:8px 18px; font-size:0.82rem;">
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
          <path d="M12 5v14M5 12h14"/>
        </svg>
        Add Provider
      </button>
    </div>
  </div>

  {#if loading}
    <p style="color:var(--text-muted); padding:20px;">Loading...</p>
  {:else if providers.length === 0}
    <div class="empty-state">
      <p class="empty-title">No inference providers configured</p>
      <p class="empty-desc">Add an inference provider to enable LLM access for your agents. Agents reference providers by their <strong>Provider ID</strong>.</p>
    </div>
  {:else}
    <div class="provider-cards">
      {#each providers as prov}
        <div class="provider-card" class:expanded={expandedProvider === prov.name}>
          <div class="provider-card-header" onclick={() => toggleExpand(prov.name)} onkeydown={(e) => { if (e.key === 'Enter') toggleExpand(prov.name) }} role="button" tabindex="0">
            <div class="provider-card-main">
              <span class="provider-name">{prov.name}</span>
              <div class="provider-meta-row">
                <span class="provider-type-badge">{typeBadge(prov.provider_type)}</span>
                <span class="provider-url">{prov.base_url}</span>
              </div>
            </div>
            <div class="provider-card-meta">
              <span class="model-name">{prov.default_model || '—'}</span>
              <span class="scope-badge scope-{scopeBadge(prov.scope)}">{scopeBadge(prov.scope)}</span>
              <span class="expand-arrow">{expandedProvider === prov.name ? '▾' : '▸'}</span>
            </div>
          </div>

          {#if expandedProvider === prov.name}
            <div class="provider-card-body">
              <div class="provider-card-details">
                <div class="detail-row">
                  <span class="detail-label">Provider ID</span>
                  <span class="detail-value"><code>{prov.id}</code></span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">Schema Validation</span>
                  <span class="detail-value">{prov.schema_validation ? 'Enabled' : 'Disabled'}</span>
                </div>
                {#if prov.scope === 'team' && prov.team}
                  <div class="detail-row">
                    <span class="detail-label">Team</span>
                    <span class="detail-value">{prov.team}</span>
                  </div>
                {/if}
                <div class="detail-row">
                  <span class="detail-label">Created By</span>
                  <span class="detail-value">{prov.created_by}</span>
                </div>
              </div>
              <div class="provider-card-actions">
                <button class="action-btn" onclick={() => openEditModal(prov)}>Edit</button>
                <button class="action-btn action-btn-delete" onclick={() => deleteProvider(prov.name)}>Delete</button>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if showModal}
  <ProviderModal provider={editingProvider} onclose={closeModal} oncreated={onProviderCreated} />
{/if}

<style>
  .providers-page {
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

  .provider-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .provider-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
    transition: border-color 0.15s;
  }
  .provider-card.expanded {
    border-color: oklch(59.1% 0.249 292.7 / 0.5);
  }
  .provider-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    cursor: pointer;
    user-select: none;
    transition: background 0.1s;
  }
  .provider-card-header:hover {
    background: rgba(255,255,255,0.02);
  }
  .provider-card-main {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .provider-name {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .provider-meta-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .provider-type-badge {
    font-size: 0.62rem;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: 4px;
    background: oklch(55% 0.2 200 / 0.15);
    color: oklch(55% 0.2 200);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .provider-url {
    font-size: 0.76rem;
    color: var(--text-muted);
    font-family: 'SF Mono', 'Fira Code', monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .provider-card-meta {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }
  .model-name {
    font-size: 0.76rem;
    color: #52525b;
    font-family: 'SF Mono', 'Fira Code', monospace;
    max-width: 160px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
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
  .expand-arrow {
    font-size: 0.85rem;
    color: #52525b;
    width: 16px;
    text-align: center;
  }

  .provider-card-body {
    border-top: 1px solid var(--border);
    padding: 16px;
  }
  .provider-card-details {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 14px;
  }
  .detail-row {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 0.78rem;
  }
  .detail-label {
    width: 140px;
    color: #52525b;
    font-weight: 500;
    flex-shrink: 0;
  }
  .detail-value {
    color: var(--text-muted);
  }
  .detail-value code {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.72rem;
    background: var(--bg-sidebar);
    padding: 1px 6px;
    border-radius: 4px;
    color: var(--text-base);
  }
  .provider-card-actions {
    display: flex;
    gap: 8px;
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
</style>

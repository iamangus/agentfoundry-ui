<script>
  import { api } from '../lib/api.js'

  let keys = $state([])
  let newKeyName = $state('')
  let newKey = $state(null)
  let loading = $state(true)
  let creating = $state(false)

  $effect(() => {
    loadKeys()
  })

  async function loadKeys() {
    try {
      keys = await api.get('/api-keys')
    } catch (e) {
      console.error('Failed to load API keys', e)
    } finally {
      loading = false
    }
  }

  async function createKey() {
    if (!newKeyName.trim()) return
    creating = true
    try {
      const key = await api.post('/api-keys', { name: newKeyName.trim() })
      newKey = key
      newKeyName = ''
      await loadKeys()
    } catch (e) {
      console.error('Failed to create key', e)
      alert('Failed to create key: ' + e.message)
    } finally {
      creating = false
    }
  }

  async function revokeKey(id) {
    if (!confirm('Revoke this API key?')) return
    try {
      await api.del('/api-keys/' + id)
      await loadKeys()
    } catch (e) {
      console.error('Failed to revoke key', e)
    }
  }

  function copyKey(key) {
    navigator.clipboard.writeText(key).then(() => {
      alert('Key copied to clipboard')
    })
  }
</script>

<div class="keys-page">
  <div class="page-header">
    <h2 class="page-title">API Keys</h2>
  </div>

  <div class="create-section">
    <div class="create-form">
      <input
        bind:value={newKeyName}
        class="sb-input"
        placeholder="Key name..."
        onkeydown={(e) => { if (e.key === 'Enter') createKey() }}
      />
      <button onclick={createKey} class="sb-submit" style="width:auto;" disabled={!newKeyName.trim() || creating}>
        {creating ? 'Creating...' : 'Create Key'}
      </button>
    </div>

    {#if newKey && newKey.full_key}
      <div class="key-result">
        <div class="key-result-header">
          <span class="key-result-label">Key created — copy it now, it won't be shown again</span>
          <button class="copy-btn" onclick={() => copyKey(newKey.full_key)}>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="width:14px;height:14px;">
              <rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
            </svg>
            Copy
          </button>
        </div>
        <code class="full-key">{newKey.full_key}</code>
      </div>
    {/if}
  </div>

  {#if loading}
    <p style="color:var(--text-muted); padding:20px;">Loading...</p>
  {:else}
    <div class="key-table-wrap">
      <table class="key-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Prefix</th>
            <th>Created</th>
            <th>Last Used</th>
            <th>Expires</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#if keys.length === 0}
            <tr>
              <td colspan="6" style="text-align:center; padding:32px; color:var(--text-muted);">No API keys</td>
            </tr>
          {:else}
            {#each keys as key}
              <tr>
                <td class="key-name">{key.name}</td>
                <td class="mono">{key.key_prefix}...</td>
                <td>{new Date(key.created_at).toLocaleDateString()}</td>
                <td>{key.last_used_at ? new Date(key.last_used_at).toLocaleDateString() : '—'}</td>
                <td>{key.expires_at ? new Date(key.expires_at).toLocaleDateString() : '—'}</td>
                <td>
                  <button class="revoke-btn" onclick={() => revokeKey(key.id)}>Revoke</button>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .keys-page {
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
  .create-section {
    margin-bottom: 24px;
  }
  .create-form {
    display: flex;
    gap: 10px;
    max-width: 500px;
  }
  .key-result {
    margin-top: 16px;
    padding: 16px;
    background: var(--purple-dim);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.3);
    border-radius: 10px;
    max-width: 600px;
  }
  .key-result-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }
  .key-result-label {
    font-size: 0.8rem;
    color: oklch(70% 0.2 292.0);
    font-weight: 500;
  }
  .copy-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 10px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-muted);
    font-family: inherit;
    font-size: 0.75rem;
    cursor: pointer;
    transition: background 0.12s;
  }
  .copy-btn:hover { background: var(--bg-sidebar); color: var(--text-base); }
  .full-key {
    display: block;
    padding: 10px 14px;
    background: rgba(0,0,0,0.35);
    border-radius: 8px;
    font-size: 0.78rem;
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: var(--text-base);
    word-break: break-all;
    user-select: all;
  }

  .key-table-wrap {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
    max-width: 900px;
  }
  .key-table {
    width: 100%;
    border-collapse: collapse;
  }
  .key-table th {
    text-align: left;
    padding: 10px 16px;
    font-size: 0.72rem;
    font-weight: 600;
    color: #52525b;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--border);
    background: var(--bg-sidebar);
  }
  .key-table td {
    padding: 10px 16px;
    font-size: 0.82rem;
    border-bottom: 1px solid var(--border);
    color: var(--text-muted);
  }
  .key-table tr:last-child td { border-bottom: none; }
  .key-name {
    color: var(--text-base) !important;
    font-weight: 500;
  }
  .mono {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.78rem !important;
  }
  .revoke-btn {
    padding: 3px 10px;
    background: none;
    border: 1px solid var(--red-border);
    color: var(--red-text);
    border-radius: 5px;
    font-family: inherit;
    font-size: 0.75rem;
    cursor: pointer;
    transition: background 0.12s;
  }
  .revoke-btn:hover { background: var(--red-dim); }
</style>

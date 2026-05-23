<script>
  let { def, isNew, onsave, oncancel } = $props()

  let name = $state(def.name || '')
  let description = $state(def.description || '')
  let model = $state(def.model || '')
  let systemPrompt = $state(def.system_prompt || '')
  let toolsText = $state((def.tools || []).join('\n'))
  let maxTurns = $state(def.max_turns || 0)
  let maxConcurrentTools = $state(def.max_concurrent_tools || 0)
  let forceJson = $state(def.force_json || false)
  let scope = $state(def.scope || '')
  let team = $state(def.team || '')
  let soEnabled = $state(!!(def.structured_output))
  let soJSON = $state(def.structured_output ? JSON.stringify(def.structured_output, null, 2) : '')

  function handleSave() {
    let tools = toolsText
      .split('\n')
      .map(t => t.trim())
      .filter(t => t)

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
      <input bind:value={name} class="sb-input" placeholder="Agent name" />
    </div>

    <div class="form-group">
      <label class="form-label">Description</label>
      <input bind:value={description} class="sb-input" placeholder="Brief description" />
    </div>

    <div class="form-group">
      <label class="form-label">Model</label>
      <input bind:value={model} class="sb-input" placeholder="e.g. gpt-4o" />
    </div>

    <div class="form-group">
      <label class="form-label">System Prompt</label>
      <textarea bind:value={systemPrompt} class="sb-input form-textarea" placeholder="System prompt..." rows="6"></textarea>
    </div>

    <div class="form-group">
      <label class="form-label">Tools (one per line)</label>
      <textarea bind:value={toolsText} class="sb-input form-textarea" placeholder="tool-name&#10;another-tool" rows="4"></textarea>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label class="form-label">Max Turns</label>
        <input bind:value={maxTurns} type="number" class="sb-input" placeholder="0 = unlimited" />
      </div>
      <div class="form-group">
        <label class="form-label">Max Concurrent Tools</label>
        <input bind:value={maxConcurrentTools} type="number" class="sb-input" placeholder="0 = unlimited" />
      </div>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label class="form-label">Scope</label>
        <select bind:value={scope} class="sb-input">
          <option value="">personal</option>
          <option value="team">team</option>
          <option value="global">global</option>
        </select>
      </div>
      <div class="form-group">
        <label class="form-label">Team</label>
        <input bind:value={team} class="sb-input" placeholder="Team name" />
      </div>
    </div>

    <div class="form-group">
      <label class="form-check">
        <input bind:checked={forceJson} type="checkbox" />
        Force JSON output
      </label>
    </div>

    <div class="form-group">
      <label class="form-check">
        <input bind:checked={soEnabled} type="checkbox" />
        Structured Output
      </label>
      {#if soEnabled}
        <textarea bind:value={soJSON} class="sb-input form-textarea mono" placeholder='{"name": "...", "schema": {...}, "strict": false}' rows="8" style="margin-top:8px;"></textarea>
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
</style>

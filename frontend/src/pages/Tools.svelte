<script>
  import { api } from '../lib/api.js'

  let servers = $state([])
  let selectedTools = $state([])
  let yamlOutput = $state('')
  let loading = $state(true)

  $effect(() => {
    loadTools()
  })

  async function loadTools() {
    try {
      servers = await api.get('/tools/list')
    } catch (e) {
      console.error('Failed to load tools', e)
    } finally {
      loading = false
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
</script>

<div class="tools-page">
  <div class="page-header">
    <h2 class="page-title">Tools</h2>
    <span class="page-sub">{selectedTools.length} selected</span>
  </div>

  <div class="tools-layout">
    <div class="tool-server-list">
      {#if loading}
        <p style="color:var(--text-muted); padding:20px;">Loading...</p>
      {:else if servers.length === 0}
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
</div>

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
  .server-name {
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
  .tool-name {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .tool-desc {
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
</style>

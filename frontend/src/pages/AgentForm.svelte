<script>
  import { api } from '../lib/api.js'
  import { teams, loadTeams } from '../lib/stores.js'

  let { def, isNew = false, onsave, oncancel, servers = [], providers = [], agents = [] } = $props()

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

  let availableServers = $state(servers)
  let availableAgents = $state((agents || []).filter(a => a.name !== def.name))
  let availableProviders = $state(providers || [])
  let loading = $state(false)
  let providerID = $state(def.provider_id || '')
  let modelCapsPromise = $state(Promise.resolve(null))
  let modelParams = $state(def.model_params ? (typeof def.model_params === 'string' ? JSON.parse(def.model_params) : def.model_params) : {})
let modelParamsExpanded = $state(false)
	let customParamsRaw = $state('')
	let customParamsDirty = $state(false)
	let fetchVersion = 0
  let memoryEnabled = $state(def.memory_enabled || false)
  let memorySearchAgentID = $state(def.memory_search_agent_id || '')
  let memoryIngestAgentID = $state(def.memory_ingest_agent_id || '')
  let enabledTools = $state({})
  let selectedServers = $state([])
  let expandedServer = $state(null)
  let subAgents = $state([])
  let handoffTo = $state(def.handoff_to || '')
  let handoffs = $state(def.handoffs ? [...def.handoffs] : [])
	let preInferenceProcessors = $state((def.pre_inference_processors || []).map((processor) => ({
		...processor,
		configText: JSON.stringify(processor.config || {}, null, 2),
	})))
  let toolOverrides = $state({})
  let initialized = $state(false)

  loadTeams()

  $effect(() => {
    availableServers = servers
    availableProviders = providers || []
    availableAgents = (agents || []).filter(a => a.name !== def.name)
    if (!initialized) {
      initFromDef()
      initialized = true
    }
  })

  $effect(() => {
    if (customParamsDirty) return
    const mp = { ...modelParams }
    modelCapsPromise.then(caps => {
      if (!caps?.supported_parameters) return
      const supported = caps.supported_parameters
      const custom = {}
      for (const k of Object.keys(mp)) {
        if (!supported.includes(k)) custom[k] = mp[k]
      }
      customParamsRaw = Object.keys(custom).length ? JSON.stringify(custom, null, 2) : ''
    })
  })

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
    handoffTo = def.handoff_to || ''
    handoffs = def.handoffs ? [...def.handoffs] : []
    expandedServer = null
    if (def.tool_overrides) {
      try {
        toolOverrides = typeof def.tool_overrides === 'string' ? JSON.parse(def.tool_overrides) : def.tool_overrides
      } catch {}
    }
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

  function addHandoff(agentName) {
    if (handoffs.includes(agentName)) return
    if (agentName === name) return
    handoffs = [...handoffs, agentName]
  }

  function removeHandoff(agentName) {
    handoffs = handoffs.filter(a => a !== agentName)
  }

	function addPreInferenceProcessor() {
		preInferenceProcessors = [...preInferenceProcessors, {
			id: '',
			processor: 'mcp_tool',
			phase: 'run_start',
			on_error: 'warn',
			timeout: 0,
			configText: '{\n  "server": "",\n  "tool": ""\n}',
		}]
	}

	function updatePreInferenceProcessor(index, field, value) {
		preInferenceProcessors = preInferenceProcessors.map((processor, current) =>
			current === index ? { ...processor, [field]: value } : processor,
		)
	}

	function mcpToolConfig(processor) {
		try {
			return JSON.parse(processor.configText || '{}')
		} catch {
			return {}
		}
	}

	function mcpTools(processor) {
		const server = getServer(mcpToolConfig(processor).server)
		return server?.tools || []
	}

	function updateMCPToolConfig(index, field, value) {
		const processor = preInferenceProcessors[index]
		const config = mcpToolConfig(processor)
		config[field] = value
		if (field === 'server') config.tool = ''
		updatePreInferenceProcessor(index, 'configText', JSON.stringify(config, null, 2))
	}

	function removePreInferenceProcessor(index) {
		preInferenceProcessors = preInferenceProcessors.filter((_, current) => current !== index)
	}

	function movePreInferenceProcessor(index, direction) {
		const target = index + direction
		if (target < 0 || target >= preInferenceProcessors.length) return
		const processors = [...preInferenceProcessors]
		const current = processors[index]
		processors[index] = processors[target]
		processors[target] = current
		preInferenceProcessors = processors
	}

  function availableHandoffOptions() {
    return availableAgents.filter(a => !handoffs.includes(a.name) && a.name !== handoffTo)
  }

  $effect(() => {
    const mdl = model
    const pid = providerID
    if (!mdl || !pid) {
      modelCapsPromise = Promise.resolve(null)
      return
    }
    const ver = ++fetchVersion
    modelCapsPromise = (async () => {
      try {
        await new Promise(r => setTimeout(r, 500))
        if (ver !== fetchVersion) return null
        const resp = await api.get('/api/v1/models/capabilities?model=' + encodeURIComponent(mdl) + '&provider_id=' + encodeURIComponent(pid))
        if (ver !== fetchVersion) return null
        return resp
      } catch (e) {
        if (ver !== fetchVersion) return null
        console.error('Model capabilities fetch failed:', e)
        return null
      }
    })()
  })

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

  function getOverrideKey(serverName, toolName) {
    return serverName + '.' + toolName
  }

  function getOverrides(serverName, toolName) {
    const key = getOverrideKey(serverName, toolName)
    return toolOverrides[key] || {}
  }

  function addOverride(serverName, toolName) {
    const key = getOverrideKey(serverName, toolName)
    const ov = { ...toolOverrides }
    if (!ov[key]) ov[key] = {}
    ov[key] = { ...ov[key] }
    ov[key][''] = { value: '', force: false }
    toolOverrides = ov
  }

  function removeOverride(serverName, toolName, param) {
    const key = getOverrideKey(serverName, toolName)
    const ov = { ...toolOverrides }
    if (ov[key]) {
      ov[key] = { ...ov[key] }
      delete ov[key][param]
      if (Object.keys(ov[key]).length === 0) delete ov[key]
    }
    toolOverrides = ov
  }

  function updateOverrideParam(serverName, toolName, oldParam, newParam) {
    const key = getOverrideKey(serverName, toolName)
    const ov = { ...toolOverrides }
    if (ov[key] && ov[key][oldParam] && newParam !== oldParam) {
      ov[key] = { ...ov[key] }
      ov[key][newParam] = ov[key][oldParam]
      delete ov[key][oldParam]
      toolOverrides = ov
    }
  }

  function updateOverrideValue(serverName, toolName, param, val) {
    const key = getOverrideKey(serverName, toolName)
    const ov = { ...toolOverrides }
    if (!ov[key]) ov[key] = {}
    ov[key] = { ...ov[key] }
    ov[key][param] = { ...(ov[key][param] || {}), value: val }
    toolOverrides = ov
  }

  function updateOverrideForce(serverName, toolName, param, force) {
    const key = getOverrideKey(serverName, toolName)
    const ov = { ...toolOverrides }
    if (!ov[key]) ov[key] = {}
    ov[key] = { ...ov[key] }
    ov[key][param] = { ...(ov[key][param] || {}), force }
    toolOverrides = ov
  }

  function overrideParamKeys(serverName, toolName) {
    return Object.keys(getOverrides(serverName, toolName))
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

    if (customParamsRaw.trim()) {
      let custom
      try {
        custom = JSON.parse(customParamsRaw)
      } catch {
        alert('Invalid custom parameters JSON')
        return
      }
      modelParams = { ...modelParams, ...custom }
    }

		const processors = []
		for (const processor of preInferenceProcessors) {
			let processorConfig
			try {
				processorConfig = JSON.parse(processor.configText || '{}')
			} catch {
				alert(`Invalid JSON configuration for pre-inference processor ${processor.id || processor.processor || 'new processor'}`)
				return
			}
			processors.push({
				id: processor.id,
				processor: processor.processor,
				phase: processor.phase || 'run_start',
				config: processorConfig,
				on_error: processor.on_error || 'warn',
				timeout: processor.timeout || 0,
			})
		}

      const mpObj = Object.keys(modelParams).length > 0 ? modelParams : undefined

      onsave({
        originalName: def.name,
        name,
        description,
        model,
        provider_id: providerID,
        system_prompt: systemPrompt,
      model_params: mpObj,
      tools,
      max_turns: maxTurns,
      max_concurrent_tools: maxConcurrentTools,
      force_json: forceJson,
      scope,
      team,
      structured_output: so,
      memory_enabled: memoryEnabled,
      memory_search_agent_id: memorySearchAgentID,
      memory_ingest_agent_id: memoryIngestAgentID,
      tool_overrides: JSON.stringify(toolOverrides),
      handoff_to: handoffTo || '',
       handoffs: handoffs.length > 0 ? handoffs : undefined,
		pre_inference_processors: processors,
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
      <label class="form-label">Inference Provider</label>
      <select value={providerID} onchange={(e) => providerID = e.target.value} class="sb-input">
        <option value="">-- none --</option>
        {#each availableProviders as p}
          <option value={p.id}>{p.name} ({p.provider_type})</option>
        {/each}
      </select>
    </div>

    {#await modelCapsPromise}
      <div class="form-group">
        <div class="params-header">
          <span class="params-spinner"></span>
          <label class="form-label" style="margin-bottom:0;">Model Parameters</label>
        </div>
      </div>
    {:then caps}
      {#if caps && caps.supported_parameters && caps.supported_parameters.length > 0}
        <div class="form-group">
          <div class="params-header" onclick={() => modelParamsExpanded = !modelParamsExpanded} role="button" tabindex="0" onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); modelParamsExpanded = !modelParamsExpanded } }}>
            <span class="params-arrow" class:expanded={modelParamsExpanded}>{modelParamsExpanded ? '▾' : '▸'}</span>
            <label class="form-label" style="margin-bottom:0; cursor:pointer;">Model Parameters</label>
          </div>
          {#if modelParamsExpanded}
            <div class="params-card">
              {#each caps.supported_parameters as param}
                {#if param === 'reasoning'}
                  <div class="param-group">
                    <span class="param-label">Reasoning Effort</span>
                    <select class="sb-input" value={modelParams.reasoning?.effort || 'auto'} onchange={(e) => {
                      const mp = { ...modelParams }
                      if (!mp.reasoning) mp.reasoning = {}
                      mp.reasoning.effort = e.target.value || undefined
                      if (!mp.reasoning.effort && !mp.reasoning.max_tokens) delete mp.reasoning
                      modelParams = mp
                    }}>
                      <option value="auto">Auto</option>
                      <option value="none">None</option>
                      <option value="low">Low</option>
                      <option value="medium">Medium</option>
                      <option value="high">High</option>
                      {#if caps.default_parameters?.reasoning?.effort === 'xhigh'}
                        <option value="xhigh">X-High</option>
                      {/if}
                    </select>
                    <span class="param-label" style="margin-top:8px;">Max Reasoning Tokens</span>
                    <input type="number" class="sb-input" value={modelParams.reasoning?.max_tokens || ''} placeholder="Optional" oninput={(e) => {
                      const mp = { ...modelParams }
                      if (!mp.reasoning) mp.reasoning = {}
                      mp.reasoning.max_tokens = e.target.value ? parseInt(e.target.value) : undefined
                      if (!mp.reasoning.effort && !mp.reasoning.max_tokens) delete mp.reasoning
                      modelParams = mp
                    }} />
                    <label class="form-check" style="margin-top:8px;">
                      <input type="checkbox" checked={modelParams.reasoning?.exclude || false} onchange={(e) => {
                        const mp = { ...modelParams }
                        if (!mp.reasoning) mp.reasoning = {}
                        mp.reasoning.exclude = e.target.checked || undefined
                        if (!mp.reasoning.effort && !mp.reasoning.max_tokens && !mp.reasoning.exclude) delete mp.reasoning
                        modelParams = mp
                      }} />
                      Exclude from reasoning
                    </label>
                  </div>
                {:else if param === 'max_tokens'}
                  <div class="param-group">
                    <span class="param-label">Max Tokens</span>
                    <input type="number" class="sb-input" value={modelParams.max_tokens || ''} placeholder="Optional" oninput={(e) => {
                      const mp = { ...modelParams }
                      mp.max_tokens = e.target.value ? parseInt(e.target.value) : undefined
                      if (!mp.max_tokens) delete mp.max_tokens
                      modelParams = mp
                    }} />
                  </div>
                {:else if param === 'temperature'}
                  <div class="param-group">
                    <span class="param-label">Temperature</span>
                    <input type="number" class="sb-input" value={modelParams.temperature ?? ''} step="0.1" min="0" max="2" placeholder="Optional" oninput={(e) => {
                      const mp = { ...modelParams }
                      mp.temperature = e.target.value ? parseFloat(e.target.value) : undefined
                      if (mp.temperature === undefined) delete mp.temperature
                      modelParams = mp
                    }} />
                  </div>
                {:else if param === 'top_p'}
                  <div class="param-group">
                    <span class="param-label">Top P</span>
                    <input type="number" class="sb-input" value={modelParams.top_p ?? ''} step="0.05" min="0" max="1" placeholder="Optional" oninput={(e) => {
                      const mp = { ...modelParams }
                      mp.top_p = e.target.value ? parseFloat(e.target.value) : undefined
                      if (mp.top_p === undefined) delete mp.top_p
                      modelParams = mp
                    }} />
                  </div>
                {:else}
                  <div class="param-group">
                    <span class="param-label">{param}</span>
                    <input class="sb-input" value={modelParams[param] ?? ''} placeholder="Value" oninput={(e) => {
                      const mp = { ...modelParams }
                      mp[param] = e.target.value || undefined
                      if (mp[param] === undefined) delete mp[param]
                      modelParams = mp
                    }} />
                  </div>
                {/if}
              {/each}
              <div class="param-group" style="margin-top:8px; padding-top:8px; border-top:1px solid var(--border);">
                <span class="param-label">Custom Parameters (JSON)</span>
                <textarea
                  class="sb-input form-textarea"
                  placeholder={"{\"provider\": {\"order\": [\"ProviderName\"]}}"}
                  rows="3"
                  spellcheck="false"
                  value={customParamsRaw}
                  oninput={(e) => {
                    customParamsRaw = e.target.value
                    customParamsDirty = true
                  }}
                ></textarea>
                <span class="param-hint" style="margin-top:4px;font-size:0.73rem;color:var(--text-muted);">Provider-specific parameters merged at save time.</span>
              </div>
            </div>
          {/if}
        </div>
      {/if}
    {/await}

    <div class="form-group">
      <label class="form-label">System Prompt</label>
      <textarea value={systemPrompt} oninput={(e) => systemPrompt = e.target.value} class="sb-input form-textarea" placeholder="System prompt..." rows="6"></textarea>
    </div>

		<div class="form-group">
			<div class="tool-section-header">
				<div>
					<div class="form-label" style="margin-bottom:2px;">Pre-inference processors</div>
					<p class="tool-hint">Run once before the initial inference. Processors contribute context to the system prompt in this order.</p>
				</div>
				<button class="sb-submit" style="width:auto; padding:7px 12px; font-size:0.78rem;" onclick={addPreInferenceProcessor}>Add processor</button>
			</div>
			{#if preInferenceProcessors.length === 0}
				<p class="tool-hint">No pre-inference processors configured.</p>
			{:else}
				<div class="processor-list">
					{#each preInferenceProcessors as processor, index}
						<div class="processor-card">
							<div class="processor-card-header">
								<span class="tool-section-label" style="margin:0;">Processor {index + 1}</span>
								<div class="processor-actions">
									<button class="action-btn" disabled={index === 0} onclick={() => movePreInferenceProcessor(index, -1)}>Up</button>
									<button class="action-btn" disabled={index === preInferenceProcessors.length - 1} onclick={() => movePreInferenceProcessor(index, 1)}>Down</button>
									<button class="action-btn action-btn-delete" onclick={() => removePreInferenceProcessor(index)}>Remove</button>
								</div>
							</div>
							<div class="processor-fields">
								<input class="sb-input" placeholder="ID (optional)" value={processor.id} oninput={(e) => updatePreInferenceProcessor(index, 'id', e.target.value)} />
								<input class="sb-input" placeholder="Processor type" value={processor.processor} oninput={(e) => updatePreInferenceProcessor(index, 'processor', e.target.value)} />
								<select class="sb-input" value={processor.phase || 'run_start'} onchange={(e) => updatePreInferenceProcessor(index, 'phase', e.target.value)}>
									<option value="run_start">run_start</option>
								</select>
								<select class="sb-input" value={processor.on_error || 'warn'} onchange={(e) => updatePreInferenceProcessor(index, 'on_error', e.target.value)}>
									<option value="warn">warn on error</option>
									<option value="skip">skip on error</option>
									<option value="fail">fail on error</option>
								</select>
							</div>
							{#if processor.processor === 'mcp_tool'}
								<div class="processor-fields">
									<select class="sb-input" value={mcpToolConfig(processor).server || ''} onchange={(e) => updateMCPToolConfig(index, 'server', e.target.value)}>
										<option value="">Select MCP server</option>
										{#each availableServers as server}
											<option value={server.name}>{server.name}</option>
										{/each}
									</select>
									<select class="sb-input" value={mcpToolConfig(processor).tool || ''} disabled={!mcpToolConfig(processor).server} onchange={(e) => updateMCPToolConfig(index, 'tool', e.target.value)}>
										<option value="">Select tool</option>
										{#each mcpTools(processor) as tool}
											<option value={tool.name}>{tool.name}</option>
										{/each}
									</select>
								</div>
							{/if}
							<div class="form-label" style="margin-top:10px;">Timeout (seconds, 0 uses default)</div>
							<input class="sb-input processor-timeout" type="number" min="0" value={processor.timeout || 0} oninput={(e) => updatePreInferenceProcessor(index, 'timeout', e.target.valueAsNumber || 0)} />
							<div class="form-label" style="margin-top:10px;">Configuration (JSON)</div>
							<textarea class="sb-input form-textarea mono" rows="4" value={processor.configText} oninput={(e) => updatePreInferenceProcessor(index, 'configText', e.target.value)}></textarea>
						</div>
					{/each}
				</div>
			{/if}
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

                    <div class="tool-override-section">
                      <div class="tool-section-label" style="margin-top:12px; margin-bottom:8px;">Param Overrides</div>
                      {#each tools as tool}
                        {#if isChecked(srvName, tool.name)}
                          <div class="tool-override-tool-group">
                            <span class="tool-override-tool-name">{tool.name}</span>
                            {#each overrideParamKeys(srvName, tool.name) as param}
                              <div class="tool-override-row">
                                <input
                                  class="sb-input override-param-input"
                                  placeholder="param name"
                                  value={param}
                                  onblur={(e) => updateOverrideParam(srvName, tool.name, param, e.target.value.trim())}
                                  oninput={null}
                                />
                                <input
                                  class="sb-input override-val-input"
                                  placeholder='${agentID}'
                                  value={getOverrides(srvName, tool.name)[param]?.value || ''}
                                  oninput={(e) => updateOverrideValue(srvName, tool.name, param, e.target.value)}
                                />
                                <label class="override-force-label">
                                  <input type="checkbox" checked={getOverrides(srvName, tool.name)[param]?.force || false} onchange={(e) => updateOverrideForce(srvName, tool.name, param, e.target.checked)} />
                                  force
                                </label>
                                <button class="override-remove-btn" onclick={() => removeOverride(srvName, tool.name, param)} title="Remove">&times;</button>
                              </div>
                            {/each}
                            <button class="override-add-btn" onclick={() => addOverride(srvName, tool.name)}>+ Add override</button>
                          </div>
                        {/if}
                      {/each}
                    </div>
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

    <div class="form-group">
      <div class="tool-divider"></div>
      <div class="tool-section-label">Handoffs</div>
      <p class="tool-hint" style="margin-bottom:12px;">Deterministic handoff routes the agent to another agent when it would return its final response. LLM-invoked handoffs expose <code>handoff_to_&lt;agent&gt;</code> tools the model can call mid-conversation to transfer control.</p>

      <label class="form-label" style="margin-bottom:4px;">Deterministic handoff target</label>
      <select class="sb-input tool-server-select" value={handoffTo} onchange={(e) => handoffTo = e.target.value}>
        <option value="">None (no deterministic handoff)</option>
        {#each availableAgents as a}
          <option value={a.name} disabled={handoffs.includes(a.name)}>{a.name}{handoffs.includes(a.name) ? ' (in handoffs list)' : ''}</option>
        {/each}
      </select>

      <label class="form-label" style="margin-top:16px; margin-bottom:4px;">LLM-invoked handoffs</label>
      {#if availableHandoffOptions().length > 0}
        <div class="tool-add-row">
          <select class="sb-input tool-server-select" id="handoff-select">
            <option value="">-- Select agent --</option>
            {#each availableHandoffOptions() as a}
              <option value={a.name}>{a.name}</option>
            {/each}
          </select>
          <button class="sb-submit" style="width:auto; padding:9px 16px; font-size:0.82rem;" onclick={() => {
            const sel = document.getElementById('handoff-select')
            if (sel && sel.value) { addHandoff(sel.value); sel.value = '' }
          }}>Add</button>
        </div>
      {/if}

      {#if handoffs.length > 0}
        <div class="sub-agent-list">
          {#each handoffs as agentName}
            <div class="sub-agent-chip">
              <span class="sub-agent-icon">↪</span>
              <div class="sub-agent-info">
                <span class="sub-agent-name">{agentName}</span>
                {#if getAgentDesc(agentName)}
                  <span class="sub-agent-desc">{getAgentDesc(agentName)}</span>
                {/if}
              </div>
              <button class="sub-agent-remove" onclick={() => removeHandoff(agentName)} title="Remove">&times;</button>
            </div>
          {/each}
        </div>
      {:else}
        <p class="tool-hint" style="margin-bottom:16px;">No LLM-invoked handoffs selected.</p>
      {/if}
    </div>

    <div class="form-group">
      <label class="form-check">
        <input checked={memoryEnabled} onchange={(e) => memoryEnabled = e.target.checked} type="checkbox" />
        Enable Memory (Graphiti)
      </label>
      {#if memoryEnabled}
        <div style="margin-top:10px; display:flex; flex-direction:column; gap:10px;">
          <div>
            <label class="form-label" style="margin-bottom:4px;">Memory Search Agent ID</label>
            <input value={memorySearchAgentID} oninput={(e) => memorySearchAgentID = e.target.value} class="sb-input" placeholder="UUID of the memory search agent" />
          </div>
          <div>
            <label class="form-label" style="margin-bottom:4px;">Memory Ingest Agent ID</label>
            <input value={memoryIngestAgentID} oninput={(e) => memoryIngestAgentID = e.target.value} class="sb-input" placeholder="UUID of the memory ingest agent" />
          </div>
        </div>
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

  .tool-override-section {
    margin-top: 8px;
    border-top: 1px solid var(--border);
    padding-top: 4px;
  }
  .tool-override-tool-group {
    margin-bottom: 10px;
    padding: 8px;
    background: var(--bg-sidebar);
    border-radius: 6px;
  }
	.processor-list {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-top: 10px;
	}
	.processor-card {
		background: var(--bg-sidebar);
		border: 1px solid var(--border);
		border-radius: 8px;
		padding: 12px;
	}
	.processor-card-header,
	.processor-actions,
	.processor-fields {
		display: flex;
		align-items: center;
	}
	.processor-card-header {
		justify-content: space-between;
		gap: 12px;
	}
	.processor-actions {
		gap: 6px;
	}
	.processor-fields {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 8px;
		margin-top: 10px;
	}
	.processor-timeout {
		max-width: 180px;
	}
  .tool-override-tool-name {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    font-family: 'SF Mono', 'Fira Code', monospace;
    display: block;
    margin-bottom: 6px;
  }
  .tool-override-row {
    display: flex;
    gap: 6px;
    align-items: center;
    margin-bottom: 4px;
  }
  .override-param-input {
    width: 120px;
    font-size: 0.72rem;
    padding: 3px 6px;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }
  .override-val-input {
    flex: 1;
    font-size: 0.72rem;
    padding: 3px 6px;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }
  .override-force-label {
    display: flex;
    align-items: center;
    gap: 3px;
    font-size: 0.7rem;
    color: var(--text-muted);
    white-space: nowrap;
    cursor: pointer;
  }
  .override-force-label input[type="checkbox"] {
    accent-color: var(--purple-solid);
  }
  .override-remove-btn {
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 0.95rem;
    cursor: pointer;
    padding: 0 4px;
    line-height: 1;
  }
  .override-remove-btn:hover { color: #ef4444; }
  .override-add-btn {
    background: var(--bg-card);
    color: var(--purple-solid);
    border: 1px solid var(--purple-soft);
    border-radius: 5px;
    font-family: inherit;
    font-size: 0.72rem;
    cursor: pointer;
    padding: 3px 10px;
    margin-top: 2px;
  }
  .override-add-btn:hover { background: var(--purple-soft); }

  .params-card {
    background: var(--bg-sidebar);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .params-header {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    user-select: none;
    padding: 2px 0;
  }
  .params-arrow {
    display: inline-block;
    width: 14px;
    text-align: center;
    font-size: 0.8rem;
    color: var(--text-muted);
    transition: transform 0.15s;
  }
  .params-arrow.expanded {
    transform: rotate(90deg);
  }
  .params-spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid var(--border);
    border-top-color: oklch(70% 0.2 292.0);
    border-radius: 50%;
    animation: params-spin 0.7s linear infinite;
  }
  @keyframes params-spin {
    to { transform: rotate(360deg); }
  }
  .param-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .param-label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
  }
</style>

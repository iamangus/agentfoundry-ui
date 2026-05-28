<script>
  import { api } from '../lib/api.js'

  let executions = $state([])
  let loading = $state(true)
  let expandedExecs = $state(new Set())
  let histories = $state({})
  let statusFilter = $state('')
  let expandedEvent = $state(null)
  let selectedSpan = $state(null)
  let selectedSpanExecId = $state(null)

  $effect(() => {
    loadExecutions()
    const interval = setInterval(() => {
      loadExecutions(true)
      refreshExpandedHistories()
    }, 1000)
    return () => clearInterval(interval)
  })

  async function loadExecutions(silent = false) {
    if (!silent) loading = true
    try {
      let url = '/api/v1/executions'
      if (statusFilter) url += '?status=' + statusFilter
      executions = await api.get(url)
    } catch (e) {
      console.error('Failed to load executions', e)
    } finally {
      if (!silent) loading = false
    }
  }

  async function refreshExpandedHistories() {
    const ids = [...expandedExecs]
    if (!ids.length) return
    const results = await Promise.allSettled(
      ids.map(id => api.get(`/api/v1/executions/${id}`))
    )
    results.forEach((r, i) => {
      if (r.status === 'fulfilled') {
        histories[ids[i]] = { ...histories[ids[i]], data: r.value }
      }
    })
    histories = { ...histories }
  }

  async function toggleExpand(execId) {
    if (expandedExecs.has(execId)) {
      expandedExecs.delete(execId)
      expandedExecs = new Set(expandedExecs)
      expandedEvent = null
      selectedSpan = null
      selectedSpanExecId = null
      return
    }
    expandedExecs.add(execId)
    expandedExecs = new Set(expandedExecs)
    expandedEvent = null
    selectedSpan = null
    selectedSpanExecId = null
    if (!histories[execId]?.data) {
      histories[execId] = { ...histories[execId], loading: true }
      histories = { ...histories }
      try {
        const data = await api.get(`/api/v1/executions/${execId}`)
        histories[execId] = { loading: false, data }
        histories = { ...histories }
      } catch (e) {
        console.error('Failed to load execution history', e)
        histories[execId] = { ...histories[execId], loading: false }
        histories = { ...histories }
      }
    }
  }

  function formatTime(t) {
    if (!t) return '\u2014'
    const d = new Date(t)
    return d.toLocaleString()
  }

  function formatDuration(start, end) {
    if (!start) return '\u2014'
    const s = new Date(start).getTime()
    const e = end ? new Date(end).getTime() : Date.now()
    const ms = e - s
    if (ms < 1) return '0ms'
    if (ms < 1000) return Math.round(ms) + 'ms'
    if (ms < 60000) return (ms / 1000).toFixed(1) + 's'
    if (ms < 3600000) return Math.floor(ms / 60000) + 'm ' + Math.round((ms % 60000) / 1000) + 's'
    return Math.floor(ms / 3600000) + 'h ' + Math.floor((ms % 3600000) / 60000) + 'm'
  }

  function statusClass(status) {
    const m = (status || '').toLowerCase()
    if (m.includes('completed') || m === 'completed') return 'status-completed'
    if (m.includes('running') || m === 'running') return 'status-running'
    if (m.includes('failed')) return 'status-failed'
    if (m.includes('cancel')) return 'status-canceled'
    if (m.includes('terminat')) return 'status-failed'
    if (m.includes('timedout') || m.includes('timeout')) return 'status-failed'
    return 'status-running'
  }

  function statusLabel(status) {
    const m = (status || '').toLowerCase()
    if (m.includes('completed') || m === 'completed') return 'Completed'
    if (m.includes('running') || m === 'running') return 'Running'
    if (m.includes('failed')) return 'Failed'
    if (m.includes('cancel')) return 'Canceled'
    if (m.includes('terminat')) return 'Terminated'
    if (m.includes('timedout') || m.includes('timeout')) return 'Timed Out'
    return status || 'Unknown'
  }

  function eventColor() {
    return 'event-workflow'
  }

  function toggleEventDetail(eventId) {
    expandedEvent = expandedEvent === eventId ? null : eventId
  }

  function spanColor(spanType) {
    switch (spanType) {
      case 'activity': return 'span-activity'
      case 'child_workflow': return 'span-child'
      case 'timer': return 'span-timer'
      default: return 'span-default'
    }
  }

  function spanTime(t) {
    if (!t) return ''
    const iso = new Date(t).toISOString()
    return iso.substring(11, 23)
  }

  function eventsForSpan(span) {
    const hx = selectedSpanExecId ? histories[selectedSpanExecId]?.data : null
    if (!hx || !span) return []
    const endId = span.end_event_id || (hx.history.length > 0 ? hx.history[hx.history.length - 1].event_id : span.start_event_id)
    return hx.history.filter(e =>
      e.event_id >= span.start_event_id && e.event_id <= endId
    )
  }

  function timelinePct(t, globalStart, globalEnd) {
    if (!t) return 100
    const ts = new Date(t).getTime()
    const total = globalEnd - globalStart
    if (total <= 0) return 0
    return ((ts - globalStart) / total) * 100
  }

  function selectSpan(span, execId) {
    if (selectedSpan && selectedSpan.id === span.id && selectedSpanExecId === execId) {
      selectedSpan = null
      selectedSpanExecId = null
    } else {
      selectedSpan = span
      selectedSpanExecId = execId
    }
  }

  function spanDisplayName(span, hx) {
    const agentName = hx?.agent_name || 'Agent'
    switch (span.name) {
      case 'ResolveAgentActivity': return `Getting ${agentName} Config`
      case 'BuildToolDefsActivity': return 'Building Tool List'
      case 'LLMChatActivity': return `Running ${agentName}`
      case 'CallToolActivity': return getCallToolLabel(span, hx)
      default: return span.name || span.type
    }
  }

  function getCallToolLabel(span, hx) {
    if (!hx) return 'Running Tool'
    const event = hx.history?.find(e => e.event_id === span.start_event_id)
    const toolName = event?.details?.activityTaskScheduledEventAttributes?.input?.payloads?.[0]?.tool_name
    return toolName ? `Running ${toolName}` : 'Running Tool'
  }
</script>

<div class="executions-page">
  <div class="page-header">
    <h3 class="section-title">Executions</h3>
    <div class="header-actions">
      <select bind:value={statusFilter} class="status-filter" onchange={() => loadExecutions()}>
        <option value="">All Statuses</option>
        <option value="Running">Running</option>
        <option value="Completed">Completed</option>
        <option value="Failed">Failed</option>
        <option value="Canceled">Canceled</option>
        <option value="Terminated">Terminated</option>
        <option value="TimedOut">Timed Out</option>
      </select>
      <button class="sb-submit" onclick={() => loadExecutions()} style="width:auto;padding:8px 18px;font-size:0.82rem;">
        Refresh
      </button>
    </div>
  </div>

  {#if loading}
    <p style="color:var(--text-muted); padding:20px;">Loading executions...</p>
  {:else if executions.length === 0}
    <div class="empty-state">
      <p class="empty-title">No executions found</p>
      <p class="empty-desc">Run an agent from the Chat or Agents tab to see workflow executions appear here. Data is sourced directly from Temporal.</p>
    </div>
  {:else}
    <div class="execution-cards">
      {#each executions as exec (exec.workflow_id)}
        {@const hstate = histories[exec.workflow_id]}
        {@const hx = hstate?.data}
        <div class="execution-card" class:expanded={expandedExecs.has(exec.workflow_id)}>
          <div class="execution-card-header" onclick={() => toggleExpand(exec.workflow_id)} onkeydown={(e) => { if (e.key === 'Enter') toggleExpand(exec.workflow_id) }} role="button" tabindex="0">
            <div class="execution-card-main">
              <div class="execution-title-row">
                <span class="execution-agent-name">{exec.agent_name || 'unknown agent'}</span>
                {#if exec.run_type}
                  <span class="run-type-badge">{exec.run_type}</span>
                {/if}
              </div>
              <span class="execution-wf-id">{exec.workflow_id}</span>
            </div>
            <div class="execution-card-meta">
              <span class="status-badge {statusClass(exec.status)}">{statusLabel(exec.status)}</span>
              <span class="execution-time">{formatTime(exec.start_time)}</span>
              <span class="execution-duration">{formatDuration(exec.start_time, exec.close_time)}</span>
              <span class="expand-arrow">{expandedExecs.has(exec.workflow_id) ? '\u25BE' : '\u25B8'}</span>
            </div>
          </div>

          {#if expandedExecs.has(exec.workflow_id)}
            <div class="execution-card-body">
              {#if hstate?.loading}
                <p style="color:var(--text-muted); padding:16px;">Loading timeline...</p>
              {:else if hx}
                <div class="execution-detail-header">
                  <div class="detail-row">
                    <span class="detail-label">Workflow ID</span>
                    <span class="detail-value"><code>{hx.workflow_id}</code></span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">Run ID</span>
                    <span class="detail-value"><code>{hx.run_id || '\u2014'}</code></span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">Agent</span>
                    <span class="detail-value">{hx.agent_name || '\u2014'}</span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">Status</span>
                    <span class="detail-value"><span class="status-badge {statusClass(hx.status)}">{statusLabel(hx.status)}</span></span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">Started</span>
                    <span class="detail-value">{formatTime(hx.start_time)}</span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">Duration</span>
                    <span class="detail-value">{formatDuration(hx.start_time, hx.close_time)}</span>
                  </div>
                </div>
                {#if hx.spans && hx.spans.length > 0}
                  {@const globalStart = new Date(hx.spans[0].start_time).getTime()}
                  {@const globalEnd = hx.close_time ? new Date(hx.close_time).getTime() : Date.now()}
                  <div class="spans-chart">
                    <div class="spans-time-header">
                      <span class="spans-time-label">{spanTime(hx.spans[0]?.start_time)}</span>
                      <span class="spans-time-label">{spanTime(hx.close_time || new Date().toISOString())}</span>
                    </div>
                     {#each hx.spans as span}
                       {@const sp = timelinePct(span.start_time, globalStart, globalEnd)}
                       {@const ep = timelinePct(span.end_time, globalStart, globalEnd)}
                       {@const bw = Math.max(0.1, ep - sp)}
                       {@const barFits = bw > 20}
                       {@const flip = !barFits && ep > 85}
                       <div class="span-row" class:selected={selectedSpan && selectedSpan.id === span.id && selectedSpanExecId === exec.workflow_id}>
                         <div class="span-track">
                           <div
                             class="span-bar {spanColor(span.type)}"
                             style="left: {sp}%; width: {bw}%"
                             onclick={() => selectSpan(span, exec.workflow_id)}
                             onkeydown={(e) => { if (e.key === 'Enter') selectSpan(span, exec.workflow_id) }}
                             role="button"
                             tabindex="0"
                             title={spanDisplayName(span, hx) + ': ' + formatDuration(span.start_time, span.end_time)}
                           >
                             {#if barFits}
                               <span class="span-bar-label">{spanDisplayName(span, hx)}</span>
                             {/if}
                           </div>
                           {#if !barFits}
                             <span class="span-name-label" class:span-name-label--left={flip} style={flip ? `right: ${100 - sp}%` : `left: ${ep}%`}>{spanDisplayName(span, hx)}</span>
                           {/if}
                         </div>
                         <span class="span-duration">{formatDuration(span.start_time, span.end_time)}</span>
                       </div>
                     {/each}                  </div>
                  {#if selectedSpan && selectedSpanExecId === exec.workflow_id}
                    <div class="spans-event-list">
                      <div class="spans-event-list-header">
                        Events for {spanDisplayName(selectedSpan, hx)}
                        <span class="spans-event-range">({selectedSpan.start_event_id}&ndash;{selectedSpan.end_event_id})</span>
                      </div>
                      {#each eventsForSpan(selectedSpan) as event}
                        <div class="timeline-item">
                          <div class="timeline-marker {eventColor(event.event_type)}"></div>
                          <div class="timeline-content" onclick={() => toggleEventDetail(event.event_id)} onkeydown={(e) => { if (e.key === 'Enter') toggleEventDetail(event.event_id) }} role="button" tabindex="0">
                            <div class="timeline-event-header">
                              <span class="timeline-event-type">{event.event_type}</span>
                              <span class="timeline-event-time">{formatTime(event.event_time)}</span>
                            </div>
                            <span class="timeline-event-summary">{event.summary}</span>
                            {#if expandedEvent === event.event_id}
                              <div class="timeline-event-details">
                                <pre>{JSON.stringify({event_id: event.event_id, event_type: event.event_type, event_time: event.event_time, summary: event.summary, details: event.details}, null, 2)}</pre>
                              </div>
                            {/if}
                          </div>
                        </div>
                      {/each}
                    </div>
                  {/if}
                {:else}
                  <div class="timeline">
                    {#each hx.history as event}
                      <div class="timeline-item">
                        <div class="timeline-marker {eventColor(event.event_type)}"></div>
                        <div class="timeline-content" onclick={() => toggleEventDetail(event.event_id)} onkeydown={(e) => { if (e.key === 'Enter') toggleEventDetail(event.event_id) }} role="button" tabindex="0">
                          <div class="timeline-event-header">
                            <span class="timeline-event-type">{event.event_type}</span>
                            <span class="timeline-event-time">{formatTime(event.event_time)}</span>
                          </div>
                          <span class="timeline-event-summary">{event.summary}</span>
                          {#if expandedEvent === event.event_id}
                            <div class="timeline-event-details">
                              <pre>{JSON.stringify({event_id: event.event_id, event_type: event.event_type, event_time: event.event_time, summary: event.summary, details: event.details}, null, 2)}</pre>
                            </div>
                          {/if}
                        </div>
                      </div>
                    {/each}
                  </div>
                {/if}
              {:else}
                <p style="color:var(--text-muted); padding:16px;">Failed to load execution history.</p>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .executions-page {
    flex: 1;
    overflow-y: auto;
    padding: 24px 32px;
  }

  .page-header {
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
  .header-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .status-filter {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 7px;
    color: var(--text-base);
    padding: 7px 10px;
    font-size: 0.82rem;
    outline: none;
  }
  .status-filter:focus {
    border-color: var(--purple);
  }

  .sb-submit {
    background: var(--purple-dim);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.3);
    border-radius: 7px;
    color: var(--text-base);
    cursor: pointer;
    font-weight: 600;
    transition: background 0.12s;
    display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  }
  .sb-submit:hover { background: var(--purple); }

  .empty-state {
    padding: 40px 24px;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.82rem;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
  }
  .empty-title { font-size: 0.95rem; font-weight: 600; color: var(--text-base); margin: 0 0 8px 0; }
  .empty-desc { margin: 0; max-width: 480px; margin-inline: auto; line-height: 1.5; }

  .execution-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .execution-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
    transition: border-color 0.15s;
  }
  .execution-card.expanded {
    border-color: oklch(59.1% 0.249 292.7 / 0.5);
  }
  .execution-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    cursor: pointer;
    user-select: none;
    transition: background 0.1s;
  }
  .execution-card-header:hover {
    background: rgba(255,255,255,0.02);
  }
  .execution-card-main {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }
  .execution-title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .execution-agent-name {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .execution-wf-id {
    font-size: 0.72rem;
    color: var(--text-muted);
    font-family: monospace;
  }

  .run-type-badge {
    font-size: 0.65rem;
    font-weight: 600;
    padding: 1px 7px;
    border-radius: 4px;
    background: var(--purple-dim);
    color: var(--purple);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.25);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .execution-card-meta {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;
  }

  .status-badge {
    font-size: 0.72rem;
    font-weight: 600;
    padding: 2px 9px;
    border-radius: 5px;
  }
  .status-running { background: rgba(59,130,246,0.15); color: #60a5fa; border: 1px solid rgba(59,130,246,0.3); }
  .status-completed { background: rgba(34,197,94,0.12); color: #4ade80; border: 1px solid rgba(34,197,94,0.25); }
  .status-canceled { background: rgba(251,191,36,0.12); color: #fbbf24; border: 1px solid rgba(251,191,36,0.25); }
  .status-failed { background: rgba(239,68,68,0.12); color: #f87171; border: 1px solid rgba(239,68,68,0.25); }

  .execution-time {
    font-size: 0.72rem;
    color: var(--text-muted);
  }
  .execution-duration {
    font-size: 0.72rem;
    color: var(--text-muted);
    font-family: monospace;
  }
  .expand-arrow {
    font-size: 0.8rem;
    color: var(--text-muted);
    width: 18px;
    text-align: center;
  }

  .execution-card-body {
    border-top: 1px solid var(--border);
    padding: 0;
  }

  .execution-detail-header {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 24px;
    padding: 12px 16px 8px;
    border-bottom: 1px solid var(--border);
  }
  .detail-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.78rem;
  }
  .detail-label { color: var(--text-muted); }
  .detail-value { color: var(--text-base); }
  .detail-value code {
    font-size: 0.7rem;
    background: var(--bg-sidebar);
    padding: 1px 5px;
    border-radius: 3px;
    border: 1px solid var(--border);
  }

  .timeline {
    padding: 16px 16px 16px 32px;
    position: relative;
  }
  .timeline::before {
    content: '';
    position: absolute;
    left: 26px;
    top: 0;
    bottom: 0;
    width: 2px;
    background: var(--border);
  }
  .timeline-item {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    position: relative;
    padding-bottom: 12px;
  }
  .timeline-item:last-child { padding-bottom: 0; }
  .timeline-marker {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
    margin-top: 5px;
    z-index: 1;
  }
  .event-workflow { background: var(--purple); }
  .event-activity { background: #60a5fa; }
  .event-child { background: #4ade80; }
  .event-timer { background: #9ca3af; }
  .event-default { background: #6b7280; }

  .timeline-content {
    flex: 1;
    min-width: 0;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 6px;
    transition: background 0.1s;
  }
  .timeline-content:hover {
    background: rgba(255,255,255,0.03);
  }
  .timeline-event-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }
  .timeline-event-type {
    font-size: 0.78rem;
    font-weight: 600;
    color: var(--text-base);
  }
  .timeline-event-time {
    font-size: 0.65rem;
    color: var(--text-muted);
    font-family: monospace;
    flex-shrink: 0;
  }
  .timeline-event-summary {
    font-size: 0.75rem;
    color: var(--text-muted);
    display: block;
    margin-top: 2px;
  }
  .timeline-event-details {
    margin-top: 8px;
    background: var(--bg-sidebar);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px;
    overflow-x: auto;
  }
  .timeline-event-details pre {
    margin: 0;
    font-size: 0.68rem;
    color: var(--text-muted);
    font-family: monospace;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .spans-chart {
    padding: 16px 16px 20px;
  }
  .spans-time-header {
    display: flex;
    justify-content: space-between;
    padding: 0 64px 6px 8px;
  }
  .spans-time-label {
    font-size: 0.65rem;
    color: var(--text-muted);
    font-family: monospace;
  }
  .span-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 3px 0;
  }
  .span-row.selected {
    background: rgba(255,255,255,0.03);
    border-radius: 4px;
  }
  .span-name-label {
    position: absolute;
    top: 3px;
    font-size: 0.72rem;
    font-weight: 600;
    color: var(--text-base);
    white-space: nowrap;
    pointer-events: none;
    margin-left: 6px;
  }
  .span-name-label--left {
    margin-left: 0;
    margin-right: 6px;
  }
  .span-track {
    flex: 1;
    height: 22px;
    position: relative;
    background: var(--bg-sidebar);
    border-radius: 4px;
    border: 1px solid var(--border);
  }
  .span-bar {
    position: absolute;
    top: 3px;
    height: 14px;
    border-radius: 3px;
    min-width: 3px;
    cursor: pointer;
    display: flex;
    align-items: center;
    overflow: hidden;
    transition: opacity 0.1s;
    opacity: 0.8;
  }
  .span-bar:hover {
    opacity: 1;
  }
  .span-bar-label {
    font-size: 0.72rem;
    font-weight: 600;
    color: #fff;
    padding: 0 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    text-shadow: 0 1px 2px rgba(0,0,0,0.3);
  }
  .span-duration {
    width: 56px;
    flex-shrink: 0;
    font-size: 0.68rem;
    color: var(--text-muted);
    font-family: monospace;
  }
  .span-activity, .span-child, .span-timer, .span-default { background: var(--purple); }

  .spans-event-list {
    margin-top: 8px;
    border-top: 1px solid var(--border);
    padding: 12px 16px 12px 32px;
    position: relative;
  }
  .spans-event-list::before {
    content: '';
    position: absolute;
    left: 26px;
    top: 0;
    bottom: 0;
    width: 2px;
    background: var(--border);
  }
  .spans-event-list-header {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-base);
    padding-bottom: 10px;
  }
  .spans-event-range {
    font-size: 0.68rem;
    font-weight: 400;
    color: var(--text-muted);
    font-family: monospace;
    margin-left: 6px;
  }
</style>

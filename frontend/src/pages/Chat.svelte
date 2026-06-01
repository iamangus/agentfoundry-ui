<script>
  import { api } from '../lib/api.js'
  import { navigate } from '../lib/stores.js'
  import { marked } from 'marked'

  let agents = $state([])
  let sessions = $state([])
  let currentSession = $state(null)
  let messages = $state([])
  let selectedAgent = $state('')
  let newMessage = $state('')
  let loading = $state(true)
  let sidebarOpen = $state(true)
  let messageListEl = $state(null)

  let streamBubbles = $state([])
  let streamingRaw = ''
  let streamParsePending = false
  let streamingStatus = $state('')
  let activeRunId = $state('')
  let eventSource = $state(null)

  $effect(() => {
    loadData()
    return () => {
      eventSource?.close()
    }
  })

  async function loadData() {
    try {
      const [a, s] = await Promise.all([
        api.get('/agents/list'),
        api.get('/api/v1/chat/sessions')
      ])
      agents = a
      sessions = s

      const url = new URL(window.location.href)
      const sessionId = url.searchParams.get('session')
      if (sessionId) {
        const sess = sessions.find(s => s.id === sessionId)
        if (sess) {
          await selectSession(sess)
        }
      }
    } catch (e) {
      console.error('Failed to load chat data', e)
    } finally {
      loading = false
    }
  }

  async function selectSession(session) {
    currentSession = session
    navigate('/chat?session=' + session.id)
    try {
      const full = await api.get('/api/v1/chat/sessions/' + session.id)
      messages = (full.messages || []).map(msg => {
        if (msg.role === 'assistant') {
          return { ...msg, content: marked.parse(msg.content) }
        }
        return msg
      })
      if (full.active_run_id) {
        startStream(full.active_run_id)
      }
    } catch (e) {
      console.error('Failed to load session', e)
    }
  }

  async function createSession() {
    if (!selectedAgent) return
    try {
      const sess = await api.post('/api/v1/chat/sessions', { agent_id: selectedAgent })
      sessions = [sess, ...sessions]
      await selectSession(sess)
      await loadData()
    } catch (e) {
      console.error('Failed to create session', e)
    }
  }

  async function sendMessage() {
    if (!newMessage.trim() || !currentSession) return
    const content = newMessage
    newMessage = ''
    messages = [...messages, { role: 'user', content }]
    requestAnimationFrame(() => scrollDown())

    try {
      const result = await api.post('/api/v1/chat/sessions/' + currentSession.id + '/messages', { message: content })
      startStream(result.run_id)
    } catch (e) {
      console.error('Failed to send message', e)
    }
  }

  function startStream(runId) {
    eventSource?.close()
    activeRunId = runId
    streamingStatus = 'Thinking'
    streamBubbles = []
    streamingRaw = ''
    streamParsePending = false

    const es = new EventSource('/chat/runs/' + runId + '/events')
    eventSource = es

    es.addEventListener('response_start', () => {
      streamingStatus = ''
      streamingRaw = ''
      streamBubbles = [...streamBubbles, '']
    })

    es.addEventListener('token', (e) => {
      streamingStatus = ''
      streamingRaw += e.data
      if (streamBubbles.length === 0) {
        streamBubbles = ['']
      }
      if (streamParsePending === false) {
        streamParsePending = true
        requestAnimationFrame(() => {
          streamBubbles[streamBubbles.length - 1] = marked.parse(streamingRaw)
          streamParsePending = false
        })
      }
      scrollDown()
    })

    es.addEventListener('status', (e) => {
      streamingStatus = e.data
    })

    es.addEventListener('done', (e) => {
      es.close()
      eventSource = null
      finalizeStream(e.data)
    })

    es.addEventListener('error', (e) => {
      if (es.readyState === EventSource.CLOSED) return
      es.close()
      eventSource = null
      if (e.data) {
        finalizeStream(e.data)
      } else {
        streamingStatus = ''
        streamBubbles = []
        streamingRaw = ''
        streamParsePending = false
      }
      activeRunId = ''
    })

    es.onerror = () => {
      es.close()
      eventSource = null
      streamParsePending = false
      activeRunId = ''
    }
  }

  function finalizeStream(mdText) {
    streamingStatus = ''
    if (streamBubbles.length > 0) {
      streamBubbles[streamBubbles.length - 1] = marked.parse(mdText || '')
      for (let i = 0; i < streamBubbles.length; i++) {
        messages = [...messages, { role: 'assistant', content: streamBubbles[i] }]
      }
    }
    streamBubbles = []
    streamingRaw = ''
    streamParsePending = false
    activeRunId = ''
    requestAnimationFrame(() => scrollDown())
  }

  function scrollDown() {
    if (!messageListEl) return
    const nearBottom = messageListEl.scrollHeight - messageListEl.scrollTop - messageListEl.clientHeight < 50
    if (nearBottom) {
      messageListEl.scrollTop = messageListEl.scrollHeight
    }
  }

  function handleKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }
</script>

<aside class="sidebar" class:collapsed={!sidebarOpen}>
  <div class="sidebar-inner">
    <div class="sidebar-section">
      <div class="sb-section-label">New Chat</div>
      <div style="display:flex; gap:6px; margin-top:6px;">
        <select value={selectedAgent} onchange={(e) => selectedAgent = e.target.value} class="sb-input" style="flex:1;">
          {#if agents.length === 0}
            <option disabled>No agents</option>
          {:else}
            <option value="">Select agent...</option>
            {#each agents as a}
              <option value={a.agent_id}>{a.name}</option>
            {/each}
          {/if}
        </select>
        <button onclick={createSession} disabled={!selectedAgent} class="sb-submit" style="width:auto; padding: 6px 12px;" aria-label="Create session">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" style="width:14px;height:14px;">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
        </button>
      </div>
    </div>
    <div class="sidebar-section" style="padding-top:0;">
      <div class="sb-section-label">Sessions</div>
    </div>
    <div class="session-list">
      {#if sessions.length === 0}
        <div class="session-empty">No chats yet</div>
      {:else}
        {#each sessions as s}
          <button
            class="session-row"
            class:active={currentSession?.id === s.id}
            onclick={() => selectSession(s)}
          >
            <span class="session-name">{s.agent_name}</span>
            <span class="session-preview">{s.messages ? s.messages.length : 0} messages</span>
          </button>
        {/each}
      {/if}
    </div>
  </div>
</aside>

<main class="chat">
  {#if currentSession}
    <div class="chat-layout">
      <div class="chat-head">
        <button class="sidebar-toggle-btn" onclick={() => sidebarOpen = !sidebarOpen} aria-label="Toggle sidebar">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 3v18"/>
          </svg>
        </button>
        <div class="chat-head-icon">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09Z" />
          </svg>
        </div>
        <span class="chat-head-name">{currentSession.agent_name}</span>
        <span class="chat-head-badge">{messages.length} messages</span>
      </div>

      <div class="chat-body" bind:this={messageListEl}>
        {#each messages as msg}
          <div class="msg-row" class:msg-right={msg.role === 'user'} class:msg-left={msg.role !== 'user'}>
            <div class="bubble" class:bubble-user={msg.role === 'user'} class:bubble-bot={msg.role !== 'user'}>
              {#if msg.role === 'user'}
                {msg.content}
              {:else}
                {@html msg.content}
              {/if}
            </div>
          </div>
        {/each}

        {#each streamBubbles as content}
          <div class="msg-row msg-left">
            <div class="bubble bubble-bot">
              {@html content || ''}
            </div>
          </div>
        {/each}

        {#if streamingStatus}
          <div class="msg-row msg-left">
            <div class="thinking-bubble">
              <span class="thinking-label">{streamingStatus}</span>
              <div class="thinking-dot"></div>
              <div class="thinking-dot" style="animation-delay:0.2s"></div>
              <div class="thinking-dot" style="animation-delay:0.4s"></div>
            </div>
          </div>
        {/if}

        <div class="scroll-anchor"></div>
      </div>

      <div class="chat-foot">
        <div class="input-wrap">
          <textarea
            value={newMessage}
            onkeydown={handleKeydown}
            rows="1"
            placeholder="Message {currentSession.agent_name}..."
            oninput={(e) => { newMessage = e.target.value; e.target.style.height = 'auto'; e.target.style.height = Math.min(e.target.scrollHeight, 140) + 'px'; }}
          ></textarea>
          <button onclick={sendMessage} class="send-btn" aria-label="Send" disabled={!currentSession}>
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 12 3.269 3.125A59.769 59.769 0 0 1 21.485 12 59.768 59.768 0 0 1 3.27 20.875L5.999 12Zm0 0h7.5" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  {:else}
    <div class="chat-layout">
      <div class="chat-head">
        <button class="sidebar-toggle-btn" onclick={() => sidebarOpen = !sidebarOpen} aria-label="Toggle sidebar">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 3v18"/>
          </svg>
        </button>
      </div>
      <div class="empty-state">
        <div class="empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
          </svg>
        </div>
        <p class="empty-title">No chat selected</p>
        <p class="empty-sub">Choose an agent and start a chat from the sidebar.</p>
      </div>
    </div>
  {/if}
</main>

<style>
  .sidebar {
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    width: 260px;
    min-width: 260px;
    overflow: hidden;
    transition: width 0.2s ease, min-width 0.2s ease;
  }
  .sidebar.collapsed {
    width: 0; min-width: 0; border-right: none;
  }
  .sidebar-inner {
    display: flex; flex-direction: column; height: 100%;
  }
  .sidebar-section {
    padding: 16px 12px 10px;
  }
  .sb-section-label {
    font-size: 0.65rem; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.08em; color: #52525b; margin-bottom: 6px; padding: 0 4px;
  }
  .session-list {
    flex: 1; overflow-y: auto; padding: 4px 12px;
  }
  .session-empty {
    padding: 16px 4px; color: #52525b; font-size: 0.82rem;
  }
  .session-row {
    display: flex; flex-direction: column;
    padding: 8px 10px; border-radius: 8px;
    cursor: pointer; border: none; background: none; color: inherit;
    width: 100%; text-align: left; font-family: inherit;
    transition: background 0.12s, border-color 0.12s;
    border: 1px solid transparent; margin: 2px 0;
  }
  .session-row:hover { background: var(--bg-card); color: var(--text-base); }
  .session-row.active {
    background: var(--purple-dim);
    border-color: oklch(59.1% 0.249 292.7 / 0.25);
    color: var(--text-base);
  }
  .session-name { font-size: 0.82rem; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .session-preview { font-size: 0.75rem; color: var(--text-muted); margin-top: 2px; }

  .chat { display: flex; flex-direction: column; height: 100%; overflow: hidden; flex: 1; }
  .chat-layout { display: flex; flex-direction: column; height: 100%; }

  .chat-head {
    padding: 14px 24px; border-bottom: 1px solid var(--border);
    display: flex; align-items: center; gap: 12px;
    background: var(--bg-base); flex-shrink: 0;
  }
  .chat-head-icon {
    width: 34px; height: 34px; border-radius: 9px;
    background: var(--purple-dim); border: 1px solid oklch(59.1% 0.249 292.7 / 0.4);
    display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  }
  .chat-head-icon svg { width: 17px; height: 17px; color: var(--purple); }
  .chat-head-name { font-size: 0.95rem; font-weight: 600; color: var(--text-base); }
  .chat-head-badge {
    font-size: 0.7rem; font-weight: 500;
    background: var(--purple-dim); color: oklch(70% 0.2 292.0);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.3);
    padding: 2px 8px; border-radius: 20px;
  }

  .chat-body {
    flex: 1; overflow-y: auto; padding: 28px 24px;
    display: flex; flex-direction: column; gap: 16px;
  }

  .chat-foot {
    padding: 16px 24px; border-top: 1px solid var(--border);
    background: var(--bg-base); flex-shrink: 0;
  }
  .input-wrap {
    display: flex; gap: 10px; align-items: flex-end;
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: 12px; padding: 10px 10px 10px 16px;
    transition: border-color 0.15s, box-shadow 0.15s;
  }
  .input-wrap:focus-within {
    border-color: oklch(59.1% 0.249 292.7 / 0.6);
    box-shadow: 0 0 0 3px oklch(59.1% 0.249 292.7 / 0.1);
  }
  .input-wrap textarea {
    flex: 1; resize: none; background: transparent; border: none; outline: none;
    color: var(--text-base); font-family: inherit; font-size: 0.9rem;
    line-height: 1.6; max-height: 140px; overflow-y: auto;
  }
  .input-wrap textarea::placeholder { color: var(--text-muted); }
  .send-btn {
    flex-shrink: 0; width: 36px; height: 36px; border-radius: 8px;
    background: var(--purple-solid); border: none; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    transition: opacity 0.15s, transform 0.1s; color: #fff;
  }
  .send-btn:hover { opacity: 0.85; }
  .send-btn:active { transform: scale(0.93); }
  .send-btn:disabled { opacity: 0.5; pointer-events: none; }
  .send-btn svg { width: 16px; height: 16px; }

  .sidebar-toggle-btn {
    flex-shrink: 0; width: 30px; height: 30px; border-radius: 7px;
    border: 1px solid var(--border); background: var(--bg-card); color: var(--text-muted);
    cursor: pointer; display: flex; align-items: center; justify-content: center;
    transition: background 0.12s, color 0.12s;
  }
  .sidebar-toggle-btn:hover { background: var(--purple-dim); color: var(--text-base); }
  .sidebar-toggle-btn svg { width: 14px; height: 14px; pointer-events: none; }

  .thinking-bubble {
    background: var(--bg-card); border: 1px solid var(--border); border-bottom-left-radius: 4px;
    padding: 14px 18px; border-radius: 14px;
    display: flex; align-items: baseline; gap: 4px;
  }
  .thinking-label { font-size: 0.9rem; color: var(--text-muted); }
  .thinking-dot {
    width: 5px; height: 5px; border-radius: 50%; background: var(--text-muted);
    animation: thinkanim 1.4s infinite both;
  }
  @keyframes thinkanim {
    0%, 80%, 100% { opacity: 0.2; transform: scale(0.85); }
    40%           { opacity: 1;   transform: scale(1.15); }
  }
  .scroll-anchor { overflow-anchor: auto; height: 1px; }
</style>

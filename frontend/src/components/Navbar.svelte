<script>
  import { user, userLoading } from '../lib/stores.js'
  import { navigate } from '../lib/stores.js'

  let { page } = $props()

  const tabs = [
    { name: 'Chat',     path: '/chat',     svg: 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z' },
    { name: 'Agents',   path: '/agents',   svg: 'M12 8a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM4 20c0-4 3.6-7 8-7s8 3 8 7' },
    { name: 'Tools',    path: '/tools',    svg: 'M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z' },
    { name: 'Keys',     path: '/api-keys', svg: 'M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4' },
  ]

  const currentUser = $derived($user)
</script>

<nav class="nav-bar">
  <a href="/" class="nav-brand" onclick={(e) => { e.preventDefault(); navigate('/'); }}>
    <div class="nav-brand-icon">
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09Z" />
      </svg>
    </div>
    <span class="nav-brand-text"><span class="nav-brand-agent">agent</span><span class="nav-brand-file">foundry</span></span>
  </a>

  <div class="nav-tabs">
    {#each tabs as tab}
      <a
        href={tab.path}
        class="nav-tab"
        class:active={page === tab.path}
        onclick={(e) => { e.preventDefault(); navigate(tab.path); }}
      >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d={tab.svg}></path>
        </svg>
        {tab.name}
      </a>
    {/each}
  </div>

  {#if currentUser}
    <div class="nav-spacer"></div>
    <div class="nav-user">
      <span class="nav-user-name">{currentUser.username}</span>
      {#if currentUser.is_admin}<span class="nav-admin-badge">admin</span>{/if}
      {#if currentUser.is_team_admin}<span class="nav-admin-badge">team-admin</span>{/if}
      <a href="/auth/logout" class="nav-logout">Sign out</a>
    </div>
  {/if}
</nav>

<style>
  .nav-bar {
    background: var(--bg-sidebar);
    border-bottom: 1px solid var(--border);
    min-height: 52px;
    display: flex;
    align-items: center;
    padding: 0 20px;
    gap: 16px;
  }
  .nav-brand {
    display: flex; align-items: center; gap: 10px;
    text-decoration: none;
    flex-shrink: 0;
  }
  .nav-brand-icon {
    width: 28px; height: 28px; border-radius: 7px;
    background: var(--purple-dim);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.4);
    display: flex; align-items: center; justify-content: center;
  }
  .nav-brand-icon svg { width: 14px; height: 14px; color: var(--purple); }
  .nav-brand-text {
    font-size: 0.9rem; font-weight: 700;
    letter-spacing: -0.01em;
  }
  .nav-brand-agent { color: var(--text-base); }
  .nav-brand-file { color: var(--purple); }
  .nav-tabs {
    display: flex; align-items: center; gap: 2px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 9px;
    padding: 3px;
  }
  .nav-tab {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 6px 14px;
    border-radius: 7px;
    font-size: 13px;
    font-weight: 500;
    text-decoration: none;
    color: var(--text-muted);
    transition: background 0.12s, color 0.12s;
    border: 1px solid transparent;
  }
  .nav-tab:hover {
    background: var(--bg-sidebar);
    color: var(--text-base);
  }
  .nav-tab.active {
    background: var(--purple-dim);
    border-color: oklch(59.1% 0.249 292.7 / 0.25);
    color: var(--text-base);
  }
  .nav-tab svg {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
  }
  .nav-spacer { flex: 1; }
  .nav-user {
    display: flex; align-items: center; gap: 8px;
    font-size: 0.8rem; color: var(--text-muted);
  }
  .nav-user-name {
    font-weight: 500; color: var(--text-base);
    max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .nav-admin-badge {
    font-size: 0.65rem; font-weight: 600;
    background: var(--purple-dim); color: var(--purple);
    border: 1px solid oklch(59.1% 0.249 292.7 / 0.3);
    padding: 1px 6px; border-radius: 4px;
  }
  .nav-logout {
    font-size: 0.78rem; color: var(--text-muted);
    text-decoration: none; padding: 4px 8px;
    border-radius: 6px; transition: background 0.12s, color 0.12s;
  }
  .nav-logout:hover { background: var(--bg-card); color: var(--text-base); }
</style>

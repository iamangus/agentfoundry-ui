<script>
  import { user, theme, navigate } from '../lib/stores.js'
  let { page } = $props()
  let open = $state(false)
  const tabs = [
    { name: 'Chat', path: '/chat', icon: 'M4 4h16v12H8l-4 4z', group: 'Workspace' },
    { name: 'Agents', path: '/agents', icon: 'M8 9a4 4 0 1 0 8 0 4 4 0 0 0-8 0M4 21c0-7 16-7 16 0' },
    { name: 'Executions', path: '/executions', icon: 'M3 12h4l3-8 4 16 3-8h4' },
    { name: 'Providers', path: '/providers', icon: 'M4 7h16v10H4zM8 3v4m8-4v4M8 17v4m8-4v4', group: 'Configuration' },
    { name: 'Tools', path: '/tools', icon: 'm8 4-5 8 5 8m8-16 5 8-5 8m-2-17-4 18' },
    { name: 'API keys', path: '/api-keys', icon: 'M14 8a5 5 0 1 0-4 8L21 5m-4 4 3 3' },
  ]
  function go(e, path) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
    e.preventDefault(); open = false; navigate(path)
  }
</script>

<aside class="app-navigation" class:menu-open={open}>
  <div class="brand-row">
    <a href="/" class="brand" onclick={(e) => go(e, '/')}><span class="brand-symbol" aria-hidden="true">a<span>f</span></span><span>agent<span class="brand-accent">foundry</span><small>Agent workspace</small></span></a>
    <button class="quiet-button mobile-menu" aria-label="Toggle navigation" aria-expanded={open} aria-controls="primary-navigation" onclick={() => open = !open}><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d={open ? 'M6 6l12 12M18 6 6 18' : 'M4 6h16M4 12h16M4 18h16'}/></svg></button>
  </div>
  <nav id="primary-navigation" aria-label="Main navigation">
    {#each tabs as tab}
      {#if tab.group}<p class="nav-group">{tab.group}</p>{/if}
      <a href={tab.path} class="nav-link" class:active={page === tab.path || (page === '/' && tab.path === '/chat')} aria-current={page === tab.path || (page === '/' && tab.path === '/chat') ? 'page' : undefined} onclick={(e) => go(e, tab.path)}>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={tab.icon}/></svg>{tab.name}
      </a>
    {/each}
  </nav>
  <div class="nav-footer">
    <button class="theme-button" onclick={() => theme.toggle()} aria-label={$theme === 'dark' ? 'Use light theme' : 'Use dark theme'}><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l2 2m10 10 2 2M5 19l2-2M17 7l2-2"/></svg>{$theme === 'dark' ? 'Light appearance' : 'Dark appearance'}</button>
    {#if $user}
      <div class="account-row"><span class="avatar" aria-hidden="true">{($user.username || 'U').slice(0, 1).toUpperCase()}</span><div><strong>{$user.username}</strong><small>{$user.is_admin ? 'Administrator' : $user.is_team_admin ? 'Team administrator' : 'Personal workspace'}</small></div></div>
      <a class="sign-out" href="/auth/logout">Sign out</a>
    {/if}
  </div>
</aside>

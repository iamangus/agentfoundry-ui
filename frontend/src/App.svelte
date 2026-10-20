<script>
  import Navbar from './components/Navbar.svelte'
  import Feedback from './components/Feedback.svelte'
  import Chat from './pages/Chat.svelte'
  import Agents from './pages/Agents.svelte'
  import Tools from './pages/Tools.svelte'
  import ApiKeys from './pages/ApiKeys.svelte'
  import Providers from './pages/Providers.svelte'
  import Executions from './pages/Executions.svelte'
  import { loadUser, canNavigate } from './lib/stores.js'

  let page = $state(window.location.pathname)

  loadUser()

  async function handlePopstate() {
    if (!await canNavigate()) { history.pushState({}, '', page); return }
    page = window.location.pathname
  }
</script>

<svelte:window onpopstate={handlePopstate} onappnavigate={() => page = window.location.pathname} />

<a class="skip-link" href="#main-content">Skip to content</a>
<div class="workspace">
  <Navbar {page} />

  <main id="main-content" class="page-body" tabindex="-1">
    {#key page}
      {#if page === '/chat' || page === '/'}
        <Chat />
      {:else if page === '/agents'}
        <Agents />
      {:else if page === '/providers'}
        <Providers />
      {:else if page === '/tools'}
        <Tools />
      {:else if page === '/api-keys'}
        <ApiKeys />
      {:else if page === '/executions'}
        <Executions />
      {:else}
        <Chat />
      {/if}
    {/key}
  </main>
</div>
<Feedback />

<style>
  .page-body {
    display: flex;
    overflow: hidden;
    min-width: 0;
    min-height: 0;
    height: 100dvh;
    flex: 1;
  }
  @media (max-width: 760px) { .page-body { height: calc(100dvh - 64px); } }
</style>

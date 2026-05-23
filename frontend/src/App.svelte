<script>
  import Navbar from './components/Navbar.svelte'
  import Chat from './pages/Chat.svelte'
  import Agents from './pages/Agents.svelte'
  import Tools from './pages/Tools.svelte'
  import ApiKeys from './pages/ApiKeys.svelte'
  import { loadUser } from './lib/stores.js'

  let page = $state(window.location.pathname)

  loadUser()

  function handlePopstate() {
    page = window.location.pathname
  }
</script>

<svelte:window onpopstate={handlePopstate} />

<div class="dark">
  <Navbar {page} />

  <main class="page-body">
    {#if page === '/chat' || page === '/'}
      <Chat />
    {:else if page === '/agents'}
      <Agents />
    {:else if page === '/tools'}
      <Tools />
    {:else if page === '/api-keys'}
      <ApiKeys />
    {:else}
      <Chat />
    {/if}
  </main>
</div>

<style>
  .page-body {
    display: flex;
    overflow: hidden;
    height: calc(100vh - 52px);
  }
</style>

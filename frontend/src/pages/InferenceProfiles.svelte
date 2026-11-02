<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { user } from '../lib/stores.js'
  import { confirmAction, notify } from '../lib/feedback.js'
  import InferenceProfileModal from '../components/InferenceProfileModal.svelte'
  let profiles = $state([])
  let providers = $state([])
  let loading = $state(true)
  let error = $state('')
  let search = $state('')
  let showModal = $state(false)
  let editing = $state(null)
  const filtered = $derived(profiles.filter(p => `${p.name} ${p.model} ${p.backup_model}`.toLowerCase().includes(search.toLowerCase())))
  onMount(() => { load() })
  async function load() {
    loading = true
    error = ''
    try { [profiles, providers] = await Promise.all([api.get('/api/v1/inference-profiles'), api.get('/api/v1/providers')]) }
    catch (e) { error = e.message }
    finally { loading = false }
  }
  function providerName(id) { return providers.find(p => p.id === id)?.name || id }
  function canEdit(p) { return $user?.is_admin || p.scope === 'user' && p.created_by === ($user?.sub || $user?.subject) || p.scope === 'team' && $user?.is_team_admin && $user?.teams?.includes(p.team) }
  async function remove(p) {
    if (!await confirmAction(`Delete profile "${p.name}"? Agents that used it will not change.`, { title:'Delete profile?', label:'Delete profile' })) return
    try { await api.del('/api/v1/inference-profiles/' + encodeURIComponent(p.id)); await load() }
    catch (e) { notify(e.message) }
  }
</script>

<div class="profiles-page">
  <div class="section-header"><h1 class="section-title">Inference profiles</h1><button class="sb-submit" onclick={() => { editing = null; showModal = true }}>Create profile</button></div>
  <p class="page-description">Save primary and backup routing as reusable templates. Applying a profile copies its settings; later edits never change existing agents.</p>
  <div class="list-toolbar"><input class="sb-input" type="search" aria-label="Search profiles" placeholder="Search profiles or models…" bind:value={search} /></div>
  {#if error}<div class="inline-error" role="alert">{error} <button class="secondary-button" onclick={load}>Retry</button></div>{/if}
  {#if loading}<p class="field-help">Loading…</p>
  {:else}
    {#each filtered as p}
      <section class="editor-section profile">
        <h3>{p.name} <small>{p.scope === 'user' ? 'Personal' : p.scope === 'team' ? `Team: ${p.team}` : 'Global'}</small></h3>
        <p>Primary: <strong>{providerName(p.provider_id)}</strong> / <code>{p.model}</code></p>
        <p class="field-help">Backup: {p.backup_provider_id ? `${providerName(p.backup_provider_id)} / ${p.backup_model}` : 'None'}</p>
        {#if canEdit(p)}<div class="actions"><button class="secondary-button" onclick={() => { editing = p; showModal = true }}>Edit</button><button class="secondary-button" onclick={() => remove(p)}>Delete</button></div>{/if}
      </section>
    {:else}<p class="empty-state">{profiles.length ? 'No profiles match your search.' : 'No inference profiles yet. Create one here or save one from an agent’s model settings.'}</p>{/each}
  {/if}
</div>
{#if showModal}<InferenceProfileModal profile={editing} {providers} onclose={() => showModal = false} onsaved={() => { showModal = false; load() }} />{/if}

<style>
  .profiles-page { flex:1; min-width:0; overflow-y:auto; padding:24px 32px; }
  .section-header, .actions { display:flex; align-items:center; gap:8px; justify-content:space-between; }
  .section-header .sb-submit { width:auto; }
  .profile { margin-top:16px; overflow-wrap:anywhere; }
  .profile small { font-weight:400; font-size:.75rem; color:var(--text-muted); }
  .actions { justify-content:flex-end; }
  @media (max-width:600px) { .profiles-page { padding:16px; } .section-header { flex-wrap:wrap; } }
</style>

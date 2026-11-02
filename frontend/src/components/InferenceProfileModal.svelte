<script>
  import { api } from '../lib/api.js'
  import { dialogFocus } from '../lib/dialog.js'
  import { teams, loadTeams } from '../lib/stores.js'
  import ModelPicker from './ModelPicker.svelte'
  let { profile = null, routing = {}, providers = [], onclose, onsaved } = $props()
  function snapshot() { return { ...(profile || routing) } }
  const initial = snapshot()
  let name = $state(initial.name || '')
  let providerID = $state(initial.provider_id || '')
  let model = $state(initial.model || '')
  let backupProviderID = $state(initial.backup_provider_id || '')
  let backupModel = $state(initial.backup_model || '')
  let scope = $state(initial.scope || 'user')
  let team = $state(initial.team || '')
  let error = $state('')
  let saving = $state(false)
  loadTeams()

  async function save(event) {
    event.preventDefault()
    saving = true
    error = ''
    try {
      const body = { name: name.trim(), provider_id: providerID, model: model.trim(), backup_provider_id: backupProviderID, backup_model: backupProviderID ? backupModel.trim() : '', scope, team: scope === 'team' ? team : '' }
      const saved = profile ? await api.put('/api/v1/inference-profiles/' + encodeURIComponent(profile.id), body) : await api.post('/api/v1/inference-profiles', body)
      onsaved(saved)
    } catch (e) { error = e.message || 'Could not save profile.' }
    finally { saving = false }
  }
</script>

<div class="profile-backdrop">
  <button class="dismiss" aria-label="Close profile form" onclick={() => !saving && onclose()}></button>
  <div class="profile-dialog editor-section" role="dialog" aria-modal="true" aria-label={profile ? 'Edit inference profile' : 'Save inference profile'} tabindex="-1" use:dialogFocus onkeydown={(event) => { if (event.key === 'Escape' && !saving) onclose() }}>
  <form onsubmit={save}>
    <h3>{profile ? 'Edit inference profile' : 'Save inference profile'}</h3>
    <p class="field-help">A reusable routing template. Changes here never update existing agents.</p>
    {#if error}<div class="inline-error" role="alert">{error}</div>{/if}
    <label for="profile-name">Name</label>
    <input id="profile-name" class="sb-input" bind:value={name} required />
    <label for="profile-provider">Primary provider</label>
    <select id="profile-provider" class="sb-input" bind:value={providerID} required>
      <option value="">Choose a provider</option>
      {#each providers as p}<option value={p.id}>{p.name}</option>{/each}
    </select>
    <label for="profile-model">Primary model</label>
    <ModelPicker id="profile-model" bind:value={model} {providerID} placeholder={providers.find(p => p.id === providerID)?.default_model || 'Choose or enter a model'} />
    <label for="profile-backup-provider">Backup provider (optional)</label>
    <select id="profile-backup-provider" class="sb-input" bind:value={backupProviderID}>
      <option value="">No backup</option>
      {#each providers as p}<option value={p.id}>{p.name}</option>{/each}
    </select>
    {#if backupProviderID}
      <label for="profile-backup-model">Backup model</label>
      <ModelPicker id="profile-backup-model" bind:value={backupModel} providerID={backupProviderID} placeholder={providers.find(p => p.id === backupProviderID)?.default_model || 'Choose or enter a model'} />
    {/if}
    <label for="profile-scope">Scope</label>
    <select id="profile-scope" class="sb-input" bind:value={scope}>
      <option value="user">Personal</option><option value="team">Team</option><option value="global">Global</option>
    </select>
    {#if scope === 'team'}
      <label for="profile-team">Team</label>
      <select id="profile-team" class="sb-input" bind:value={team} required><option value="">Choose a team</option>{#each $teams as t}<option value={t}>{t}</option>{/each}</select>
    {/if}
    <div class="actions">
      <button type="button" class="secondary-button" disabled={saving} onclick={onclose}>Cancel</button>
      <button class="sb-submit" disabled={saving}>{saving ? 'Saving…' : 'Save profile'}</button>
    </div>
  </form>
  </div>
</div>

<style>
  .profile-backdrop { position:fixed; inset:0; display:flex; align-items:center; justify-content:center; z-index:100; }
  .dismiss { position:absolute; inset:0; width:100%; height:100%; border:0; background:rgba(0,0,0,.55); }
  .profile-dialog { position:relative; background:var(--bg-card); color:var(--text-base); width:500px; max-width:94vw; max-height:90dvh; overflow-y:auto; padding:24px; border:1px solid var(--border); border-radius:12px; }
  label { display:block; margin:14px 0 6px; color:var(--text-muted); font-size:.82rem; }
  .actions { display:flex; justify-content:flex-end; gap:8px; margin-top:20px; }
  .actions .sb-submit { width:auto; }
</style>

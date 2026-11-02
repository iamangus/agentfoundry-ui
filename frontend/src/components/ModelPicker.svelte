<script>
  import { api } from '../lib/api.js'
  let { id, value = $bindable(''), providerID = '', discover = null, queryKey = '', placeholder = 'Use provider default' } = $props()
  let models = $state([])
  let loading = $state(false)
  let error = $state('')
  let fetched = $state(false)
  let version = 0

  $effect(() => {
    const pid = providerID
    const key = queryKey
    void key
    version++
    models = []
    error = ''
    fetched = false
    loading = false
    if (pid && !discover) fetchModels()
  })

  async function fetchModels() {
    const current = ++version
    loading = true
    error = ''
    try {
      const result = discover ? await discover() : await api.get('/api/v1/providers/' + encodeURIComponent(providerID) + '/models')
      if (current !== version) return
      models = result.data || []
      fetched = true
    } catch (e) {
      if (current === version) error = e.message || 'Could not fetch models.'
    } finally {
      if (current === version) loading = false
    }
  }
</script>

<div class="model-picker">
  <input {id} class="sb-input" bind:value list={id + '-options'} {placeholder} autocomplete="off" />
  <button type="button" class="secondary-button" onclick={fetchModels} disabled={loading || (!providerID && !discover)}>{loading ? 'Fetching…' : fetched ? 'Refresh models' : 'Fetch models'}</button>
</div>
<datalist id={id + '-options'}>
  {#each models as model}<option value={model.id}>{model.name || model.id}</option>{/each}
</datalist>
{#if error}<p class="field-help" role="status">{error} You can enter a model manually.</p>
{:else if fetched}<p class="field-help" role="status">{models.length} models available. Search the list or enter an ID manually.</p>{/if}

<style>
  .model-picker { display:flex; gap:8px; align-items:center; }
  input { min-width:0; flex:1; }
  button { flex-shrink:0; }
  @media (max-width:480px) { .model-picker { flex-wrap:wrap; } input { flex-basis:100%; } }
</style>

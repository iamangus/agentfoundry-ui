<script>
  import { notices, confirmation, dismissNotice, answerConfirmation } from '../lib/feedback.js'
  import { dialogFocus } from '../lib/dialog.js'
</script>

<div class="notifications" role="region" aria-label="Notifications">
  {#each $notices as notice (notice.id)}
    <div class="notice" class:notice-error={notice.tone === 'error'} role={notice.tone === 'error' ? 'alert' : 'status'}>
      <span>{notice.message}</span>
      <button class="quiet-button" onclick={() => dismissNotice(notice.id)} aria-label="Dismiss notification">×</button>
    </div>
  {/each}
</div>

{#if $confirmation}
  <div class="dialog-backdrop">
    <div class="confirm-dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-message" tabindex="-1" use:dialogFocus onkeydown={(e) => { if (e.key === 'Escape') answerConfirmation(false) }}>
      <h2 id="confirm-title">{$confirmation.title}</h2>
      <p id="confirm-message">{$confirmation.message}</p>
      <div class="dialog-actions">
        <button class="secondary-button" onclick={() => answerConfirmation(false)}>Cancel</button>
        <button class="sb-submit" class:destructive={$confirmation.destructive} onclick={() => answerConfirmation(true)}>{$confirmation.label}</button>
      </div>
    </div>
  </div>
{/if}

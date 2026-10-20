import { writable } from 'svelte/store'

export const notices = writable([])
export const confirmation = writable(null)
let sequence = 0

export function notify(message, tone = 'error') {
  const id = ++sequence
  notices.update(items => [...items, { id, message, tone }])
  if (tone !== 'error') setTimeout(() => dismissNotice(id), 5000)
}

export function dismissNotice(id) {
  notices.update(items => items.filter(item => item.id !== id))
}

export function confirmAction(message, { title = 'Confirm action', label = 'Confirm', destructive = true } = {}) {
  return new Promise(resolve => {
    confirmation.update(previous => {
      previous?.resolve(false)
      return { message, title, label, destructive, resolve }
    })
  })
}

export function answerConfirmation(answer) {
  confirmation.update(current => {
    current?.resolve(answer)
    return null
  })
}

import { writable } from 'svelte/store'
import { api } from './api.js'

export const user = writable(null)
export const userLoading = writable(true)

export async function loadUser() {
  try {
    const u = await api.get('/auth/me')
    user.set(u)
  } catch {
    user.set(null)
  } finally {
    userLoading.set(false)
  }
}

function getPath() {
  return window.location.pathname
}

export function navigate(path) {
  history.pushState({}, '', path)
  window.dispatchEvent(new PopStateEvent('popstate'))
}

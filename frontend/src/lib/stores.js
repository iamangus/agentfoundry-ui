import { writable, get } from 'svelte/store'
import { api } from './api.js'

export const user = writable(null)
export const userLoading = writable(true)
export const teams = writable([])

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

export async function loadTeams() {
  try {
    const data = await api.get('/api/v1/teams')
    teams.set(data.teams || [])
  } catch {
    teams.set([])
  }
}

export function navigate(path) {
  history.pushState({}, '', path)
  window.dispatchEvent(new PopStateEvent('popstate'))
}

function createThemeStore() {
  const saved = typeof localStorage !== 'undefined' ? localStorage.getItem('theme') : null
  const initial = saved || 'system'

  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('data-theme', initial)
    applyThemeClass(initial)
  }

  const { subscribe, set } = writable(initial)

  return {
    subscribe,
    set: (value) => {
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem('theme', value)
      }
      document.documentElement.setAttribute('data-theme', value)
      applyThemeClass(value)
      set(value)
    },
    toggle: () => {
      const current = get(theme)
      const next = current === 'system' ? 'dark' : current === 'dark' ? 'light' : 'system'
      theme.set(next)
    },
  }
}

function applyThemeClass(value) {
  const html = document.documentElement
  if (value === 'dark') {
    html.classList.add('dark')
  } else if (value === 'light') {
    html.classList.remove('dark')
  } else {
    const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    html.classList.toggle('dark', isDark)
  }
}

if (typeof window !== 'undefined') {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    const current = typeof localStorage !== 'undefined' ? localStorage.getItem('theme') || 'system' : 'system'
    if (current === 'system') {
      document.documentElement.classList.toggle('dark', e.matches)
    }
  })
}

export const theme = createThemeStore()

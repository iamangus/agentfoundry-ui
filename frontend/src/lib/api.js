async function request(method, path, body) {
  const opts = {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
  }
  if (body) opts.body = JSON.stringify(body)

  const res = await fetch(path, opts)
  if (!res.ok) {
    let msg = res.statusText
    try {
      const text = await res.text()
      msg = text || msg
      try {
        const err = JSON.parse(text)
        msg = err.error || text
      } catch {}
    } catch {} // body already consumed or empty
    throw new Error(msg)
  }
  if (res.status === 204 || res.headers.get('content-length') === '0') return null
  return res.json()
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body),
  put: (path, body) => request('PUT', path, body),
  del: (path) => request('DELETE', path),
}

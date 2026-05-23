async function request(method, path, body) {
  const opts = {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : {},
  }
  if (body) opts.body = JSON.stringify(body)

  const res = await fetch(path, opts)
  if (!res.ok) {
    let msg
    try {
      const err = await res.json()
      msg = err.error || JSON.stringify(err)
    } catch {
      msg = await res.text()
    }
    throw new Error(msg || res.statusText)
  }
  return res.json()
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body),
  put: (path, body) => request('PUT', path, body),
  del: (path) => request('DELETE', path),
}

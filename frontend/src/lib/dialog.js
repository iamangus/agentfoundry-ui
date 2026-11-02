// Focus containment and restoration for existing and shared dialogs.
export function dialogFocus(node) {
  const previous = document.activeElement
  const hiddenSiblings = []
  // aria-modal describes the dialog; inert also removes the background from
  // keyboard, pointer, and screen-reader interaction (including nested dialogs).
  for (let current = node; current?.parentElement && current !== document.body; current = current.parentElement) {
    for (const sibling of current.parentElement.children) {
      if (sibling === current || ['SCRIPT', 'STYLE', 'LINK'].includes(sibling.tagName) || sibling.classList.contains('modal-dismiss')) continue
      hiddenSiblings.push([sibling, sibling.inert])
      sibling.inert = true
    }
  }
  const controls = () => [...node.querySelectorAll('button, a[href], input, select, textarea, [tabindex="0"]')]
    .filter(el => !el.disabled && el.getClientRects().length)
  // Contain keyboard events immediately: waiting only for an animation frame
  // leaves a newly inert background focused, so an early Escape can be lost.
  const focusDialog = () => {
    if (!node.contains(document.activeElement)) (controls()[0] || node).focus()
  }
  focusDialog()
  const frame = requestAnimationFrame(focusDialog)
  function keydown(event) {
    if (event.key !== 'Tab') return
    const items = controls()
    if (!items.length) { event.preventDefault(); node.focus(); return }
    const first = items[0], last = items.at(-1)
    if (event.shiftKey && (document.activeElement === first || !node.contains(document.activeElement))) {
      event.preventDefault(); last.focus()
    } else if (!event.shiftKey && (document.activeElement === last || !node.contains(document.activeElement))) {
      event.preventDefault(); first.focus()
    }
  }
  node.addEventListener('keydown', keydown)
  return { destroy() {
    cancelAnimationFrame(frame)
    node.removeEventListener('keydown', keydown)
    for (const [element, wasInert] of hiddenSiblings) element.inert = wasInert
    if (previous?.isConnected) previous.focus()
  } }
}

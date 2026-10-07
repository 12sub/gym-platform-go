// Let HTMX show validation errors: by default it ignores 4xx responses.
document.addEventListener('htmx:beforeSwap', (e) => {
  if (e.detail.xhr.status === 422) {
    e.detail.shouldSwap = true;
    e.detail.isError = false;
  }
});

// The server asks us to close a dialog with: HX-Trigger: {"close-dialog":"some-dialog-id"}
document.addEventListener('close-dialog', (e) => {
  const id = e.detail && (e.detail.value || e.detail);
  if (id && window.tui && window.tui.dialog) window.tui.dialog.close(id);
});

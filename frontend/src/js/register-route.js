(function (root) {
  'use strict';
  if (!root || !root.location || typeof URLSearchParams === 'undefined') return;
  const params = new URLSearchParams(root.location.search || '');
  if (params.get('register') !== '1') return;

  let observer = null;
  let timer = null;
  function finish() {
    if (observer) observer.disconnect();
    if (timer) clearTimeout(timer);
    params.delete('register');
    const query = params.toString();
    const next = root.location.pathname + (query ? '?' + query : '') + (root.location.hash || '');
    if (root.history && typeof root.history.replaceState === 'function') {
      root.history.replaceState({}, '', next);
    }
  }

  function activateRegistration() {
    const auth = document.querySelector('[data-screen-label="Auth"]');
    if (!auth) return false;
    const button = Array.from(auth.querySelectorAll('button')).find(candidate =>
      /register|sign\s*up|create\s+account/i.test(candidate.textContent || '')
    );
    if (!button) return false;
    button.click();
    finish();
    return true;
  }

  function start() {
    if (activateRegistration()) return;
    observer = new MutationObserver(() => activateRegistration());
    observer.observe(document.documentElement, { childList: true, subtree: true });
    timer = setTimeout(() => observer && observer.disconnect(), 10000);
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start, { once: true });
  else start();
})(typeof window !== 'undefined' ? window : null);

const ASSET_ERROR_PATTERNS = [
  'Failed to fetch dynamically imported module',
  'Importing a module script failed',
  'Unable to preload CSS',
  'error loading dynamically imported module',
];

export function isAssetLoadError(error: unknown): boolean {
  const message = error instanceof Error ? error.message : String(error ?? '');
  return ASSET_ERROR_PATTERNS.some((pattern) => message.includes(pattern));
}

export function showStartupRecovery(): void {
  const root = document.querySelector<HTMLElement>('#app');
  if (!root || root.dataset.recoveryShown === 'true') return;
  root.dataset.recoveryShown = 'true';
  root.replaceChildren();

  const panel = document.createElement('main');
  panel.className = 'startup-recovery';
  panel.innerHTML = `
    <svg class="startup-recovery__mark" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"><path d="M9 15l6-6M9.5 7.5l2-2a5 5 0 017 7l-2 2M14.5 16.5l-2 2a5 5 0 01-7-7l2-2" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
    <h1>页面资源暂时不可用</h1>
    <p>请刷新页面重试。</p>
  `;
  const button = document.createElement('button');
  button.type = 'button';
  button.textContent = '重新加载';
  button.addEventListener('click', () => window.location.reload());
  panel.append(button);
  root.append(panel);
}

export function installStartupRecovery(): void {
  window.addEventListener('vite:preloadError', (event) => {
    event.preventDefault();
    showStartupRecovery();
  });
}

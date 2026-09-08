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
    <div class="startup-recovery__mark">G</div>
    <h1>页面资源暂时不可用</h1>
    <p>服务可能正在启动，或部署后浏览器仍缓存着旧版本。请稍后刷新页面。</p>
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

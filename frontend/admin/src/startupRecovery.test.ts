// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { installStartupRecovery, isAssetLoadError, showStartupRecovery } from './startupRecovery';

describe('startup resource recovery', () => {
  beforeEach(() => {
    document.body.innerHTML = '<div id="app"></div>';
  });

  it('recognizes dynamic module and stylesheet preload failures', () => {
    expect(isAssetLoadError(new Error('Unable to preload CSS for /assets/view.css'))).toBe(true);
    expect(isAssetLoadError(new Error('Failed to fetch dynamically imported module'))).toBe(true);
    expect(isAssetLoadError(new Error('request failed'))).toBe(false);
  });

  it('renders one safe recovery panel', () => {
    showStartupRecovery();
    showStartupRecovery();
    expect(document.querySelectorAll('.startup-recovery')).toHaveLength(1);
    expect(document.body.textContent).toContain('重新加载');
  });

  it('handles the Vite preload error event', () => {
    installStartupRecovery();
    const event = new Event('vite:preloadError', { cancelable: true });
    const prevent = vi.spyOn(event, 'preventDefault');
    window.dispatchEvent(event);
    expect(prevent).toHaveBeenCalled();
    expect(document.querySelector('.startup-recovery')).not.toBeNull();
  });
});

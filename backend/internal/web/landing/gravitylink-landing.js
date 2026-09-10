(() => {
  const qr = document.querySelector('[data-liveqr-image]');
  if (qr) {
    const failed = () => { qr.hidden = true; const message = document.querySelector('[data-liveqr-error]'); if (message) message.hidden = false; };
    qr.addEventListener('error', failed);
    if (qr.complete && qr.naturalWidth === 0) failed();
  }

  // 客服码：微信号一键复制
  const copyBtn = document.querySelector('[data-copy-wx]');
  if (copyBtn) {
    copyBtn.addEventListener('click', async () => {
      const text = copyBtn.getAttribute('data-copy-wx') || '';
      try {
        await navigator.clipboard.writeText(text);
      } catch {
        const ta = document.createElement('textarea');
        ta.value = text;
        ta.style.position = 'fixed';
        ta.style.opacity = '0';
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
      }
      copyBtn.textContent = '已复制 ✓';
      copyBtn.classList.add('done');
      window.setTimeout(() => { copyBtn.textContent = '复制'; copyBtn.classList.remove('done'); }, 2000);
    });
  }

  // 卡密提取页
  const issueBtn = document.querySelector('[data-kami-issue]');
  if (issueBtn) {
    const resultEl = document.querySelector('[data-kami-result]');
    const errorEl = document.querySelector('[data-kami-error]');
    const copyEl = document.querySelector('[data-kami-copy]');
    const pwdWrap = document.querySelector('[data-kami-password-wrap]');
    const pwdInput = document.getElementById('kami-password');
    const originalText = issueBtn.textContent;
    let lastContent = '';

    issueBtn.addEventListener('click', async () => {
      if (issueBtn.disabled) return;
      issueBtn.disabled = true;
      issueBtn.textContent = '领取中…';
      if (errorEl) { errorEl.hidden = true; errorEl.textContent = ''; }
      if (resultEl) resultEl.hidden = true;
      if (copyEl) copyEl.hidden = true;
      try {
        const projectId = issueBtn.getAttribute('data-project-id');
        const password = pwdInput ? pwdInput.value.trim() : '';
        const res = await fetch(`/api/v1/kami/${projectId}/issue`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ password }),
        });
        const body = await res.json();
        if (!res.ok || body.code !== 0) {
          const msg = (body && body.message) || '领取失败，请稍后重试';
          if (errorEl) { errorEl.textContent = msg; errorEl.hidden = false; }
          if (pwdWrap && (res.status === 400 || res.status === 403)) pwdWrap.hidden = false;
          return;
        }
        lastContent = (body.data && body.data.content) || '';
        if (resultEl) {
          resultEl.textContent = lastContent;
          resultEl.hidden = false;
        }
        if (copyEl) copyEl.hidden = false;
      } catch {
        if (errorEl) { errorEl.textContent = '网络异常，请稍后重试'; errorEl.hidden = false; }
      } finally {
        issueBtn.disabled = false;
        issueBtn.textContent = originalText;
      }
    });

    if (copyEl) {
      copyEl.addEventListener('click', async () => {
        if (!lastContent) return;
        try { await navigator.clipboard.writeText(lastContent); } catch { /* ignore */ }
        copyEl.textContent = '已复制 ✓';
        window.setTimeout(() => { copyEl.textContent = '复制卡密'; }, 2000);
      });
    }
  }

  const countdown = document.querySelector("[data-countdown]");
  if (!countdown) {
    return;
  }

  const target = countdown.getAttribute("data-target");
  let seconds = Number(countdown.getAttribute("data-countdown"));
  if (!target || !Number.isFinite(seconds) || seconds <= 0) {
    return;
  }

  const tick = () => {
    countdown.textContent = `${seconds} 秒后自动跳转`;
    seconds -= 1;
    if (seconds < 0) {
      window.location.href = target;
      return;
    }
    window.setTimeout(tick, 1000);
  };

  tick();
})();

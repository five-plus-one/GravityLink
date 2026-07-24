(() => {
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

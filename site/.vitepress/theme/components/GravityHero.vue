<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

const canvasRef = ref<HTMLCanvasElement | null>(null)
let raf = 0
let cleanup: (() => void) | null = null

type Node = {
  label: string
  angle: number
  radius: number
  speed: number
  size: number
  light: string
  dark: string
}

const NODES: Node[] = [
  { label: '短链', angle: 0.35, radius: 0.26, speed: 0.16, size: 4.5, light: '#2787f5', dark: '#62a7ff' },
  { label: '渠道码', angle: 1.7, radius: 0.34, speed: -0.11, size: 3.8, light: '#1877e5', dark: '#8ec2ff' },
  { label: '群活码', angle: 3.0, radius: 0.4, speed: 0.08, size: 3.5, light: '#0f5fbf', dark: '#2787f5' },
  { label: '落地页', angle: 4.5, radius: 0.3, speed: -0.13, size: 4.0, light: '#3d8ef0', dark: '#a8d0ff' },
  { label: '统计', angle: 5.6, radius: 0.38, speed: 0.1, size: 3.2, light: '#1c6ed4', dark: '#5b9fff' },
]

function isDark() {
  return document.documentElement.classList.contains('dark')
}

function drawField(ctx: CanvasRenderingContext2D, w: number, h: number, t: number, mx: number, my: number, reduced: boolean, dark: boolean) {
  // match page background so mask fade blends cleanly
  if (dark) {
    const bg = ctx.createLinearGradient(0, 0, 0, h)
    bg.addColorStop(0, '#0b1220')
    bg.addColorStop(0.45, '#070b14')
    bg.addColorStop(1, '#0b1220')
    ctx.fillStyle = bg
    ctx.fillRect(0, 0, w, h)
  } else {
    const bg = ctx.createLinearGradient(0, 0, 0, h)
    bg.addColorStop(0, '#ffffff')
    bg.addColorStop(0.5, '#eef4fc')
    bg.addColorStop(1, '#ffffff')
    ctx.fillStyle = bg
    ctx.fillRect(0, 0, w, h)
  }

  const cx = w * 0.5 + (reduced ? 0 : (mx - 0.5) * 22)
  const cy = h * 0.5 + (reduced ? 0 : (my - 0.5) * 14)
  const scale = Math.min(w, h * 1.65)

  if (dark) {
    for (let i = 0; i < 90; i++) {
      const x = ((i * 137) % w) + ((i % 3) * 2)
      const y = (i * 61) % h
      const a = 0.12 + ((i % 9) / 28)
      const r = i % 11 === 0 ? 1.4 : 0.9
      ctx.fillStyle = `rgba(232, 238, 247, ${a})`
      ctx.beginPath()
      ctx.arc(x, y, r, 0, Math.PI * 2)
      ctx.fill()
    }
  } else {
    for (let i = 0; i < 40; i++) {
      const x = ((i * 97) % w)
      const y = (i * 53) % h
      ctx.fillStyle = 'rgba(39, 135, 245, 0.08)'
      ctx.beginPath()
      ctx.arc(x, y, 1.2, 0, Math.PI * 2)
      ctx.fill()
    }
  }

  const nebula = ctx.createRadialGradient(cx, cy, scale * 0.05, cx, cy, scale * 0.55)
  if (dark) {
    nebula.addColorStop(0, 'rgba(39, 135, 245, 0.22)')
    nebula.addColorStop(0.45, 'rgba(39, 135, 245, 0.06)')
    nebula.addColorStop(1, 'rgba(7, 11, 20, 0)')
  } else {
    nebula.addColorStop(0, 'rgba(39, 135, 245, 0.16)')
    nebula.addColorStop(0.45, 'rgba(39, 135, 245, 0.04)')
    nebula.addColorStop(1, 'rgba(246, 249, 253, 0)')
  }
  ctx.fillStyle = nebula
  ctx.fillRect(0, 0, w, h)

  ctx.lineWidth = 1.25
  for (let i = 0; i < 5; i++) {
    const r = 0.2 + i * 0.08
    ctx.strokeStyle = dark
      ? `rgba(98, 167, 255, ${0.22 - i * 0.02})`
      : `rgba(39, 135, 245, ${0.2 - i * 0.02})`
    ctx.beginPath()
    ctx.ellipse(cx, cy, scale * r, scale * r * 0.4, -0.2, 0, Math.PI * 2)
    ctx.stroke()
  }

  const well = ctx.createRadialGradient(cx, cy, 2, cx, cy, scale * 0.16)
  if (dark) {
    well.addColorStop(0, 'rgba(165, 210, 255, 0.95)')
    well.addColorStop(0.18, 'rgba(39, 135, 245, 0.9)')
    well.addColorStop(0.55, 'rgba(10, 22, 40, 0.75)')
    well.addColorStop(1, 'rgba(7, 11, 20, 0)')
  } else {
    well.addColorStop(0, 'rgba(39, 135, 245, 0.55)')
    well.addColorStop(0.25, 'rgba(98, 167, 255, 0.28)')
    well.addColorStop(1, 'rgba(246, 249, 253, 0)')
  }
  ctx.fillStyle = well
  ctx.beginPath()
  ctx.arc(cx, cy, scale * 0.16, 0, Math.PI * 2)
  ctx.fill()

  ctx.fillStyle = dark ? '#e8eef7' : '#ffffff'
  ctx.beginPath()
  ctx.arc(cx, cy, 6, 0, Math.PI * 2)
  ctx.fill()

  ctx.font = '600 13px "Segoe UI", "PingFang SC", sans-serif'
  ctx.textAlign = 'center'
  ctx.fillStyle = dark ? 'rgba(232, 238, 247, 0.7)' : 'rgba(20, 35, 58, 0.72)'
  ctx.fillText('GravityLink', cx, cy + 28)

  for (const n of NODES) {
    const a = reduced ? n.angle : n.angle + t * n.speed
    const rx = scale * n.radius
    const ry = scale * n.radius * 0.4
    const x = cx + Math.cos(a) * rx
    const y = cy + Math.sin(a) * ry - Math.sin(a * 0.45) * 6
    const hue = dark ? n.dark : n.light

    ctx.strokeStyle = dark ? 'rgba(39, 135, 245, 0.28)' : 'rgba(39, 135, 245, 0.2)'
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.moveTo(cx, cy)
    ctx.lineTo(x, y)
    ctx.stroke()

    const glow = ctx.createRadialGradient(x, y, 0, x, y, n.size * 4)
    glow.addColorStop(0, dark ? hue : 'rgba(39, 135, 245, 0.35)')
    glow.addColorStop(1, dark ? 'rgba(7, 11, 20, 0)' : 'rgba(246, 249, 253, 0)')
    ctx.fillStyle = glow
    ctx.beginPath()
    ctx.arc(x, y, n.size * 4, 0, Math.PI * 2)
    ctx.fill()

    ctx.fillStyle = hue
    ctx.beginPath()
    ctx.arc(x, y, n.size, 0, Math.PI * 2)
    ctx.fill()

    const lx = Math.min(Math.max(x, 36), w - 36)
    const ly = Math.max(y - n.size - 8, 16)
    ctx.fillStyle = dark ? 'rgba(232, 238, 247, 0.88)' : 'rgba(20, 35, 58, 0.8)'
    ctx.font = '600 12px "Segoe UI", "PingFang SC", sans-serif'
    ctx.textAlign = 'center'
    ctx.fillText(n.label, lx, ly)
  }
}

onMounted(() => {
  const canvas = canvasRef.value
  if (!canvas) return
  const parent = canvas.parentElement
  if (!parent) return

  const ctx = canvas.getContext('2d')
  if (!ctx) return

  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  let mx = 0.5
  let my = 0.5
  const start = performance.now()

  const resize = () => {
    const dpr = Math.min(window.devicePixelRatio || 1, 2)
    const w = parent.clientWidth
    const h = Math.max(320, Math.min(parent.clientHeight || 420, 480))
    canvas.width = Math.floor(w * dpr)
    canvas.height = Math.floor(h * dpr)
    canvas.style.width = `${w}px`
    canvas.style.height = `${h}px`
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  }

  const onMove = (e: PointerEvent) => {
    const rect = canvas.getBoundingClientRect()
    mx = (e.clientX - rect.left) / Math.max(rect.width, 1)
    my = (e.clientY - rect.top) / Math.max(rect.height, 1)
  }

  const frame = (now: number) => {
    const t = (now - start) / 1000
    drawField(ctx, canvas.clientWidth, canvas.clientHeight, t, mx, my, reduced, isDark())
    if (!reduced) raf = requestAnimationFrame(frame)
  }

  resize()
  frame(performance.now())

  const mo = new MutationObserver(() => {
    // theme class flip → next frame uses new palette
    if (reduced) frame(performance.now())
  })
  mo.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })

  window.addEventListener('resize', resize)
  if (!reduced) canvas.addEventListener('pointermove', onMove)

  cleanup = () => {
    cancelAnimationFrame(raf)
    mo.disconnect()
    window.removeEventListener('resize', resize)
    canvas.removeEventListener('pointermove', onMove)
  }
})

onUnmounted(() => {
  cleanup?.()
})
</script>

<template>
  <div class="gl-field" aria-hidden="true">
    <canvas ref="canvasRef" />
  </div>
</template>

<style scoped>
.gl-field {
  width: 100%;
  height: 400px;
  margin: 0;
  position: relative;
  overflow: hidden;
  background: transparent;
}

canvas {
  display: block;
  width: 100%;
  height: 100%;
}

@media (max-width: 640px) {
  .gl-field {
    height: 280px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .gl-field {
    height: 320px;
  }
}
</style>

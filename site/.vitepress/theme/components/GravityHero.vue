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
  hue: string
}

const NODES: Node[] = [
  { label: '短链', angle: 0.35, radius: 0.26, speed: 0.16, size: 4.5, hue: '#62a7ff' },
  { label: '渠道码', angle: 1.7, radius: 0.34, speed: -0.11, size: 3.8, hue: '#8ec2ff' },
  { label: '群活码', angle: 3.0, radius: 0.4, speed: 0.08, size: 3.5, hue: '#2787f5' },
  { label: '落地页', angle: 4.5, radius: 0.3, speed: -0.13, size: 4.0, hue: '#a8d0ff' },
  { label: '统计', angle: 5.6, radius: 0.38, speed: 0.1, size: 3.2, hue: '#5b9fff' },
]

function drawField(ctx: CanvasRenderingContext2D, w: number, h: number, t: number, mx: number, my: number, reduced: boolean) {
  // void background so field reads on light theme too
  const bg = ctx.createLinearGradient(0, 0, 0, h)
  bg.addColorStop(0, '#0a1222')
  bg.addColorStop(0.55, '#070b14')
  bg.addColorStop(1, '#0b1528')
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, w, h)

  const cx = w * 0.5 + (reduced ? 0 : (mx - 0.5) * 22)
  const cy = h * 0.5 + (reduced ? 0 : (my - 0.5) * 14)
  const scale = Math.min(w, h * 1.65)

  // stars
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

  // soft nebula
  const nebula = ctx.createRadialGradient(cx, cy, scale * 0.05, cx, cy, scale * 0.55)
  nebula.addColorStop(0, 'rgba(39, 135, 245, 0.22)')
  nebula.addColorStop(0.45, 'rgba(39, 135, 245, 0.06)')
  nebula.addColorStop(1, 'rgba(7, 11, 20, 0)')
  ctx.fillStyle = nebula
  ctx.fillRect(0, 0, w, h)

  // orbital rings
  ctx.lineWidth = 1.25
  for (let i = 0; i < 5; i++) {
    const r = 0.2 + i * 0.08
    ctx.strokeStyle = `rgba(98, 167, 255, ${0.22 - i * 0.02})`
    ctx.beginPath()
    ctx.ellipse(cx, cy, scale * r, scale * r * 0.4, -0.2, 0, Math.PI * 2)
    ctx.stroke()
  }

  // gravity well
  const well = ctx.createRadialGradient(cx, cy, 2, cx, cy, scale * 0.16)
  well.addColorStop(0, 'rgba(165, 210, 255, 0.95)')
  well.addColorStop(0.18, 'rgba(39, 135, 245, 0.9)')
  well.addColorStop(0.55, 'rgba(10, 22, 40, 0.75)')
  well.addColorStop(1, 'rgba(7, 11, 20, 0)')
  ctx.fillStyle = well
  ctx.beginPath()
  ctx.arc(cx, cy, scale * 0.16, 0, Math.PI * 2)
  ctx.fill()

  ctx.fillStyle = '#e8eef7'
  ctx.beginPath()
  ctx.arc(cx, cy, 6, 0, Math.PI * 2)
  ctx.fill()

  ctx.font = '600 13px "Segoe UI", "PingFang SC", sans-serif'
  ctx.textAlign = 'center'
  ctx.fillStyle = 'rgba(232, 238, 247, 0.7)'
  ctx.fillText('GravityLink', cx, cy + 28)

  // nodes
  for (const n of NODES) {
    const a = reduced ? n.angle : n.angle + t * n.speed
    const rx = scale * n.radius
    const ry = scale * n.radius * 0.4
    const x = cx + Math.cos(a) * rx
    const y = cy + Math.sin(a) * ry - Math.sin(a * 0.45) * 6

    ctx.strokeStyle = 'rgba(39, 135, 245, 0.28)'
    ctx.lineWidth = 1
    ctx.beginPath()
    ctx.moveTo(cx, cy)
    ctx.lineTo(x, y)
    ctx.stroke()

    // glow
    const glow = ctx.createRadialGradient(x, y, 0, x, y, n.size * 4)
    glow.addColorStop(0, n.hue)
    glow.addColorStop(1, 'rgba(7, 11, 20, 0)')
    ctx.fillStyle = glow
    ctx.beginPath()
    ctx.arc(x, y, n.size * 4, 0, Math.PI * 2)
    ctx.fill()

    ctx.fillStyle = n.hue
    ctx.beginPath()
    ctx.arc(x, y, n.size, 0, Math.PI * 2)
    ctx.fill()

    // clamp labels inside canvas
    const lx = Math.min(Math.max(x, 36), w - 36)
    const ly = Math.max(y - n.size - 8, 16)
    ctx.fillStyle = 'rgba(232, 238, 247, 0.88)'
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
    drawField(ctx, canvas.clientWidth, canvas.clientHeight, t, mx, my, reduced)
    if (!reduced) raf = requestAnimationFrame(frame)
  }

  resize()
  frame(performance.now())

  window.addEventListener('resize', resize)
  if (!reduced) canvas.addEventListener('pointermove', onMove)

  cleanup = () => {
    cancelAnimationFrame(raf)
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
  height: 420px;
  margin: 0;
  position: relative;
  overflow: hidden;
  background: #070b14;
}

canvas {
  display: block;
  width: 100%;
  height: 100%;
}

@media (max-width: 640px) {
  .gl-field {
    height: 300px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .gl-field {
    height: 340px;
  }
}
</style>

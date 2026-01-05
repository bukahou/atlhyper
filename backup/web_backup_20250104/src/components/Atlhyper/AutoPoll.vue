<template>
  <!-- 渲染一个几乎不可见的占位即可 -->
  <span style="display: none" />
</template>

<script>
export default {
  name: 'AutoPoll',
  props: {
    // 轮询间隔(ms)
    interval: { type: Number, default: 5000 },
    // 初次挂载是否立刻触发一次
    immediate: { type: Boolean, default: true },
    // 页面不可见(切到其他tab)时是否暂停
    visibleOnly: { type: Boolean, default: true },
    // 是否启用（可用于条件开关）
    enabled: { type: Boolean, default: true },
    // 直接传入要执行的任务函数（推荐）
    task: { type: Function, default: null }
  },
  data() {
    return {
      timer: null,
      isFetching: false
    }
  },
  mounted() {
    document.addEventListener('visibilitychange', this.onVisibility)
    this.start()
  },
  beforeDestroy() {
    document.removeEventListener('visibilitychange', this.onVisibility)
    this.stop()
  },
  activated() {
    this.start()
  },
  deactivated() {
    this.stop()
  },
  methods: {
    onVisibility() {
      if (!this.visibleOnly) return
      if (document.visibilityState === 'visible') this.start()
      else this.stop()
    },
    start() {
      if (!this.enabled || this.timer) return
      if (!this.visibleOnly || document.visibilityState === 'visible') {
        if (this.immediate) this.safeTick()
        this.timer = setInterval(this.safeTick, this.interval)
      }
    },
    stop() {
      if (this.timer) {
        clearInterval(this.timer)
        this.timer = null
      }
    },
    async safeTick() {
      if (this.isFetching) return
      this.isFetching = true
      try {
        if (this.task) {
          await this.task() // 直接调用传入的方法
        } else {
          this.$emit('tick') // 或者用事件让父组件处理
        }
      } catch (e) {
        // 可选：这里不弹 Toast，避免频繁打扰
        // console.warn("[AutoPoll] tick failed:", e);
      } finally {
        this.isFetching = false
      }
    }
  }
}
</script>

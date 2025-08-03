<template>
  <div class="card-stat-wrapper">
    <div class="card-stat" :class="iconBg">
      <div class="left-icon">
        <slot name="icon" />
      </div>
      <div class="right-text">
        <div class="number" :class="numberColor">{{ number }}</div>
        <div class="title">{{ title }}</div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'CardStat',
  props: {
    iconBg: {
      type: String,
      required: true
    },
    number: {
      type: Number,
      required: true
    },
    numberColor: {
      type: String,
      required: true
    },
    title: {
      type: String,
      required: true
    }
  }
}
</script>

<style scoped>
.card-stat-wrapper {
  display: inline-block;
  width: 100%;
  max-width: 315px;
  min-width: 200px;
  box-sizing: border-box;
  margin: 8px;
  vertical-align: top;
}

.card-stat {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  border-radius: 12px;
  padding: 16px 20px;
  color: white;
  transition: all 0.3s ease;
  width: 100%;
  height: 100%;
  box-sizing: border-box;
}

.left-icon {
  width: 60px;
  height: 60px;
  border-radius: 12px;
  background: linear-gradient(
    135deg,
    rgba(255, 255, 255, 0.15),
    rgba(255, 255, 255, 0.05)
  );
  box-shadow: 0 6px 12px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.3s ease;
}

/* ✅ 通用图标样式：支持 svg、el-icon 等 */
.left-icon svg,
.left-icon i {
  font-size: 28px;
  width: 28px;
  height: 28px;
  color: #ffffff;
  transition: transform 0.3s ease;
}

.left-icon:hover svg,
.left-icon:hover i {
  transform: scale(1.1);
}

.right-text {
  margin-left: 80px;
  flex: 1;
  overflow: hidden;
}
.right-text .number {
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
}
.right-text .title {
  font-size: 14px;
  opacity: 0.85;
  margin-top: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bg1 {
  background-color: #409eff;
}
.bg2 {
  background-color: #67c23a;
}
.bg3 {
  background-color: #e6a23c;
}
.bg4 {
  background-color: #f56c6c;
}

.color1,
.color2,
.color3,
.color4 {
  color: #ffffff;
}

@media (max-width: 768px) {
  .card-stat-wrapper {
    max-width: 100%;
  }
}
</style>

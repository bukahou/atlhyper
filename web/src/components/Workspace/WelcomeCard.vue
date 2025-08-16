<template>
  <div class="welcome-card">
    <!-- 左侧头像和文字 -->
    <div class="left">
      <img class="avatar" :src="avatarUrl" alt="avatar">
      <div class="text">
        <div class="greeting">
          {{ greeting }}，<b>{{ username }}</b>，{{ message }}
        </div>
        <div class="weather">{{ weather }}</div>
      </div>
    </div>

    <!-- 右侧统计信息 -->
    <div class="right">
      <div v-for="item in stats" :key="item.label" class="stat">
        <div class="label">{{ item.label }}</div>
        <div class="value">{{ item.value }}</div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from 'axios'
import { mapState } from 'vuex'

export default {
  name: 'WelcomeCard',
  data() {
    return {
      weather: '天气加载中...',
      greeting: '您好',
      message: ''
    }
  },
  computed: {
    ...mapState({
      username: (state) => state.user.displayName
    })
  },
  props: {
    avatarUrl: {
      type: String,
      default: require('@/assets/img/avatar.png')
    },
    stats: {
      type: Array,
      default: () => [
        { label: '待办', value: '2/10' },
        { label: '项目', value: 8 },
        { label: '团队', value: 300 }
      ]
    }
  },
  created() {
    this.setGreeting()
    this.fetchWeather('Tokyo')
  },
  methods: {
    setGreeting() {
      const hour = new Date().getHours()
      if (hour >= 5 && hour < 12) {
        this.greeting = '早安'
        this.message = '开始您一天的创造吧！'
      } else if (hour >= 12 && hour < 18) {
        this.greeting = '下午好'
        this.message = '下午也要继续加油哦！'
      } else if (hour >= 18 && hour < 23) {
        this.greeting = '晚上好'
        this.message = '辛苦啦，晚上也要注意休息～'
      } else {
        this.greeting = '深夜好'
        this.message = '已是深夜，请注意休息～'
      }
    },
    async fetchWeather(cityName) {
      try {
        const geoRes = await axios.get(
          `https://geocoding-api.open-meteo.com/v1/search?name=${cityName}&count=1&language=ja&format=json`
        )
        const { latitude, longitude } = geoRes.data.results[0]
        const weatherRes = await axios.get(
          `https://api.open-meteo.com/v1/forecast?latitude=${latitude}&longitude=${longitude}&current_weather=true`
        )
        const { temperature, weathercode } = weatherRes.data.current_weather
        const weatherMap = {
          0: '晴朗',
          1: '基本晴',
          2: '多云',
          3: '阴天',
          45: '有雾',
          61: '小雨',
          71: '小雪'
        }
        const weatherText = weatherMap[weathercode] || '未知天气'
        this.weather = `${weatherText}，${temperature}℃`
      } catch (err) {
        console.error('❌ 天气获取失败', err)
        this.weather = '天气信息获取失败'
      }
    }
  }
}
</script>

<style scoped>
.welcome-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: rgba(8, 124, 219, 0.7); /* 原 #2767dd 转为 rgba 并加透明度 */
  border-radius: 12px;
  color: #fff;
  box-shadow: 0 4px 12px rgba(20, 32, 197, 0.25);
  min-height: 140px;
  width: 100%;
  padding: 30px 40px;
  box-sizing: border-box;
}

.left {
  display: flex;
  align-items: center;
  flex: 1;
}

.avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  margin-right: 20px;
}

.text .greeting {
  font-size: 20px;
  font-weight: 600;
}

.text .weather {
  margin-top: 8px;
  font-size: 16px;
  color: #ccc;
}

.right {
  display: flex;
  gap: 50px;
  flex-shrink: 0;
}

.stat .label {
  font-size: 16px;
  color: #bbb;
}

.stat .value {
  font-size: 22px;
  font-weight: bold;
  margin-top: 6px;
}
</style>

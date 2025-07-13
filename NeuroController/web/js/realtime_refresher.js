// ✅ 非模块版本：定义全局类 window.RealtimeRefresher，可在普通 <script> 中使用

window.RealtimeRefresher = class {
  constructor({ fetchFunc, interval = 30000, onChange = null, hashFunc = null }) {
    this.fetchFunc = fetchFunc;    // 必填：返回 Promise<any> 的异步获取函数
    this.interval = interval;      // 轮询间隔，默认 30 秒
    this.onChange = onChange;      // 回调函数：数据变动时触发
    this.hashFunc = hashFunc || this.defaultHashFunc;

    this.lastHash = "";
    this.timer = null;
  }

  // 默认 hash 算法（截取前 500 字节防止长对象浪费性能）
  defaultHashFunc(data) {
    return JSON.stringify(data).slice(0, 500);
  }

  async run() {
    try {
      const data = await this.fetchFunc();
      const newHash = this.hashFunc(data);

      if (newHash === this.lastHash) {
        console.log("✅ 无变化，跳过刷新");
        return;
      }

      this.lastHash = newHash;

      if (this.onChange) {
        this.onChange(data);
      }
    } catch (err) {
      console.error("❌ 定时刷新失败:", err);
    }
  }

  start() {
    this.run(); // 启动时立即执行一次
    this.timer = setInterval(() => this.run(), this.interval);
  }

  stop() {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
  }
};

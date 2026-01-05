<template>
  <el-card class="todos-card full-height-card" shadow="never">
    <div class="card-header">
      <div class="title">全体代办（摘要）</div>
      <el-tag size="mini" type="info">{{ total }} 条</el-tag>
    </div>

    <div class="content">
      <el-row v-loading="loading" :gutter="8" class="stats">
        <el-col :span="8">
          <div class="stat">
            <div class="stat__label">完成率</div>
            <div class="stat__value">{{ doneRate }}%</div>
          </div>
        </el-col>
        <el-col :span="16">
          <div class="stat">
            <div class="stat__label">优先级分布</div>
            <div class="chips">
              <el-tag size="mini" type="danger">P1 {{ byPrio.p1 }}</el-tag>
              <el-tag size="mini" type="warning">P2 {{ byPrio.p2 }}</el-tag>
              <el-tag size="mini" type="info">P3 {{ byPrio.p3 }}</el-tag>
            </div>
          </div>
        </el-col>
      </el-row>

      <div class="sub-title">最近更新</div>
      <el-timeline v-loading="loading" class="timeline">
        <el-timeline-item
          v-for="(it, i) in pageItems"
          :key="i"
          :timestamp="(it.updated_at || it.created_at || '').slice(0, 16)"
          placement="top"
        >
          <span class="mono">{{ it.username }}</span>
          <span class="sep">·</span>
          <span class="title-txt">{{ it.title }}</span>
          <el-tag
            size="mini"
            :type="it.is_done ? 'success' : 'info'"
            class="ml8"
          >
            {{ it.is_done ? "完成" : "待办" }}
          </el-tag>
          <el-tag size="mini" :type="prioType(it.priority)" class="ml8">
            P{{ it.priority || 2 }}
          </el-tag>
        </el-timeline-item>

        <div v-if="!pageItems.length && !loading" class="empty">暂无数据</div>
      </el-timeline>

      <div v-if="total > pageSize" class="pager">
        <el-pagination
          layout="prev, pager, next"
          background
          :current-page.sync="page"
          :page-size="pageSize"
          :total="total"
          @current-change="onPage"
        />
      </div>
    </div>
  </el-card>
</template>

<script>
import { listUserTodos } from '@/api/user'

export default {
  name: 'TodosAllCard',
  data() {
    return {
      loading: false,
      todos: [],
      total: 0,
      byPrio: { p1: 0, p2: 0, p3: 0 },
      page: 1,
      pageSize: 10
    }
  },
  computed: {
    doneRate() {
      if (!this.total) return 0
      const done = this.todos.filter((x) => x.is_done === 1).length
      return Math.round((done / this.total) * 100)
    },
    pageItems() {
      const start = (this.page - 1) * this.pageSize
      return (this.todos || []).slice(start, start + this.pageSize)
    }
  },
  created() {
    this.fetchAll()
  },
  methods: {
    async fetchAll() {
      this.loading = true
      try {
        const res = await listUserTodos()
        if (res.code === 20000 && res.data) {
          const arr = res.data.items || []
          // 统一按更新时间/创建时间倒序
          this.todos = arr.sort((a, b) =>
            (b.updated_at || b.created_at || '').localeCompare(
              a.updated_at || a.created_at || ''
            )
          )
          this.total = this.todos.length
          this.byPrio = {
            p1: this.todos.filter((x) => x.priority === 1).length,
            p2: this.todos.filter((x) => x.priority === 2).length,
            p3: this.todos.filter((x) => x.priority === 3).length
          }
          this.page = 1
        } else {
          this.$message.error(res.message || '加载失败')
        }
      } catch (e) {
        this.$message.error(e.message || '请求异常')
      } finally {
        this.loading = false
      }
    },
    prioType(val) {
      return val === 1 ? 'danger' : val === 2 ? 'warning' : 'info'
    },
    onPage(p) {
      this.page = p
    }
  }
}
</script>

<style scoped>
.full-height-card {
  display: flex;
  flex-direction: column;
}
.full-height-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.title {
  font-weight: 600;
}

.content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.stats {
  margin-bottom: 8px;
  flex: none;
}
.stat {
  background: #f6f8fa;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 8px 10px;
  height: 100%;
}
.stat__label {
  font-size: 12px;
  color: #909399;
}
.stat__value {
  font-size: 18px;
  font-weight: 600;
}
.chips :deep(.el-tag) {
  margin-right: 6px;
}

.sub-title {
  margin: 4px 0 4px;
  color: #606266;
  font-size: 12px;
  flex: none;
}

.timeline {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 6px 0;
}

.mono {
  font-family: ui-monospace, Menlo, Consolas, monospace;
}
.sep {
  margin: 0 6px;
  color: #c0c4cc;
}
.title-txt {
  font-weight: 500;
}
.empty {
  color: #a8abb2;
  text-align: center;
  padding: 12px 0;
}

.pager {
  flex: none;
  padding-top: 6px;
  display: flex;
  justify-content: flex-end;
}

.ml8 {
  margin-left: 8px;
}
</style>

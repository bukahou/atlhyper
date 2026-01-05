<template>
  <el-dialog
    :visible="visible"
    width="60%"
    title="Pod 详情"
    @update:visible="(val) => emits('update:visible', val)"
    @close="emits('close')"
  >
    <el-tabs v-model="activeTab">
      <el-tab-pane label="状态概览" name="status">
        <div class="tab-content">
          <p><strong>状态：</strong>{{ pod.phase }}</p>
          <p><strong>启动时间：</strong>{{ pod.startTime }}</p>
          <p><strong>重启次数：</strong>{{ pod.restartCount }}</p>
          <p><strong>QoS 类别：</strong>{{ pod.qosClass || "N/A" }}</p>
          <p><strong>当前 CPU 使用：</strong>{{ metrics.cpu || "未知" }}</p>
          <p><strong>当前内存使用：</strong>{{ metrics.memory || "未知" }}</p>
        </div>
      </el-tab-pane>

      <el-tab-pane label="基本信息" name="basic">
        <div class="tab-content">
          <p><strong>名称：</strong>{{ pod.name }}</p>
          <p><strong>命名空间：</strong>{{ pod.namespace }}</p>
          <p><strong>IP：</strong>{{ pod.podIP }}</p>
          <p><strong>所属节点：</strong>{{ pod.node }}</p>
        </div>
      </el-tab-pane>

      <el-tab-pane label="容器信息" name="container">
        <div v-if="pod.containers" class="tab-content">
          <div
            v-for="(container, idx) in pod.containers"
            :key="idx"
            class="container-box"
          >
            <p><strong>容器名称：</strong>{{ container.name }}</p>
            <p><strong>镜像：</strong>{{ container.image }}</p>
            <p><strong>端口：</strong>{{ container.port || "N/A" }}</p>
            <p><strong>CPU 限制：</strong>{{ container.cpuLimit || "N/A" }}</p>
            <p>
              <strong>内存限制：</strong>{{ container.memoryLimit || "N/A" }}
            </p>
            <hr>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="Pod 条件" name="conditions">
        <el-table :data="pod.conditions || []" border style="width: 100%">
          <el-table-column prop="type" label="类型" width="180" />
          <el-table-column prop="status" label="状态" width="100" />
          <el-table-column prop="lastTransitionTime" label="变更时间" />
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="服务信息" name="service">
        <div class="tab-content">
          <p><strong>类型：</strong>{{ pod.service?.type || "N/A" }}</p>
          <p>
            <strong>Cluster IP：</strong>{{ pod.service?.clusterIP || "N/A" }}
          </p>
          <p><strong>服务端口：</strong>{{ pod.service?.port || "N/A" }}</p>
          <p>
            <strong>容器端口：</strong>{{ pod.service?.targetPort || "N/A" }}
          </p>
          <p><strong>端口名称：</strong>{{ pod.service?.portName || "N/A" }}</p>
        </div>
      </el-tab-pane>

      <el-tab-pane label="事件" name="events">
        <div class="tab-content">
          <p v-if="events.length === 0">暂无事件</p>
          <el-table v-else :data="events" border>
            <el-table-column prop="type" label="类型" width="100" />
            <el-table-column prop="reason" label="原因" width="160" />
            <el-table-column prop="message" label="消息" />
            <el-table-column prop="time" label="时间" width="180" />
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="Pod 内日志" name="logs">
        <el-scrollbar class="log-box">
          <pre>{{ logs || "暂无日志数据" }}</pre>
        </el-scrollbar>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'

// Props & Emits
const props = defineProps({
  visible: Boolean,
  pod: Object
})
const emits = defineEmits(['close', 'update:visible'])

// 本地状态
const activeTab = ref('status')
const logs = ref('')
const events = ref([])
const metrics = ref({ cpu: '', memory: '' })

// 监听 pod 变化，刷新详情数据
watch(
  () => props.pod,
  (pod) => {
    if (!pod) return
    activeTab.value = 'status'

    logs.value = `2025-07-28T17:06:38 ✅ 返回清理后的事件，共 0 条\n...`

    events.value = [
      {
        type: 'Normal',
        reason: 'Started',
        message: 'Started container neuroagent',
        time: '2025-07-20T16:35:35Z'
      },
      {
        type: 'Normal',
        reason: 'Created',
        message: 'Created container neuroagent',
        time: '2025-07-20T16:35:34Z'
      }
    ]

    metrics.value = { cpu: '0.00 core', memory: '16.6 Mi' }
  },
  { immediate: true }
)
</script>

<style scoped>
.tab-content {
  padding: 10px 5px;
  line-height: 1.8;
  font-size: 14px;
}

.container-box {
  margin-bottom: 12px;
}

.log-box {
  max-height: 300px;
  overflow-y: auto;
  background: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 13px;
  white-space: pre-wrap;
}
</style>

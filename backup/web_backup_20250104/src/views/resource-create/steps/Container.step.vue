<template>
  <div class="step-container">
    <el-form label-width="120px">
      <!-- 基本信息 -->
      <el-divider content-position="left">基本信息</el-divider>
      <BasicFields v-model="local.basic" />

      <!-- 端口 -->
      <el-divider content-position="left">端口</el-divider>
      <PortsTable v-model="local.ports" />

      <!-- 环境变量 -->
      <el-divider content-position="left">环境变量</el-divider>
      <EnvTable v-model="local.env" />

      <!-- envFrom -->
      <el-divider content-position="left">envFrom</el-divider>
      <EnvFromTable v-model="local.envFrom" />

      <!-- 卷挂载 -->
      <el-divider content-position="left">卷挂载</el-divider>
      <VolumeMountsTable v-model="local.volumeMounts" />

      <!-- 资源（仍留在父组件，非 6 子模块之一） -->
      <el-divider content-position="left">资源（Resources）</el-divider>
      <el-form-item label="Requests">
        <div class="row">
          <el-input
            v-model="local.resources.requests.cpu"
            placeholder="cpu 如 100m"
            class="mr8"
          />
          <el-input
            v-model="local.resources.requests.memory"
            placeholder="memory 如 128Mi"
          />
        </div>
      </el-form-item>
      <el-form-item label="Limits">
        <div class="row">
          <el-input
            v-model="local.resources.limits.cpu"
            placeholder="cpu 如 500m"
            class="mr8"
          />
          <el-input
            v-model="local.resources.limits.memory"
            placeholder="memory 如 512Mi"
          />
        </div>
      </el-form-item>

      <!-- 健康检查（第 6 个子模块） -->
      <el-divider content-position="left">健康检查（Probes）</el-divider>
      <ProbesEditor v-model="local.probes" :ports="local.ports" />
    </el-form>
  </div>
</template>

<script>
import store from '../stores/createForm.store'

// 子组件
import BasicFields from './Container/BasicFields.vue'
import PortsTable from './Container/PortsTable.vue'
import EnvTable from './Container/EnvTable.vue'
import EnvFromTable from './Container/EnvFromTable.vue'
import VolumeMountsTable from './Container/VolumeMountsTable.vue'
import ProbesEditor from './Container/ProbesEditor.vue'

const trim = (x) => String(x || '').trim()
const clone = (x) => JSON.parse(JSON.stringify(x || {}))

export default {
  name: 'ContainerStep',
  components: {
    BasicFields,
    PortsTable,
    EnvTable,
    EnvFromTable,
    VolumeMountsTable,
    ProbesEditor
  },
  data() {
    const s = store.form.container || {}
    return {
      local: {
        basic: {
          name: s.name || '',
          image: s.image || '',
          pullPolicy: s.pullPolicy || ''
        },
        ports: (s.ports || []).map((p) => ({ ...p })),
        env: (s.env || []).map((e) => ({ ...e })),
        envFrom: (s.envFrom || []).map((x) => ({ ...x })),
        volumeMounts: (s.volumeMounts || []).map((m) => ({ ...m })),
        resources: {
          requests: { ...(s.resources?.requests || {}) },
          limits: { ...(s.resources?.limits || {}) }
        },
        // 子组件内部会做默认与启停控制；这里只透传已有值
        probes: clone(s.probes) || { readiness: null, liveness: null }
      }
    }
  },
  watch: {
    local: {
      deep: true,
      handler(v) {
        // ports：只保留有 containerPort 的；强转 Number
        const ports = (v.ports || [])
          .map((p) => ({
            name: trim(p.name),
            containerPort: Number(p.containerPort)
          }))
          .filter((p) => Number.isFinite(p.containerPort))

        // env：只保留有 name；同名后者覆盖前者
        const envMap = new Map();
        (v.env || []).forEach((e) => {
          const n = trim(e.name)
          if (!n) return
          envMap.set(n, { name: n, value: String(e.value ?? '') })
        })
        const env = Array.from(envMap.values())

        // envFrom：规范化 type；去重（type/name 组合）
        const seen = new Set()
        const envFrom = (v.envFrom || [])
          .map((x) => ({
            type: x.type === 'secretRef' ? 'secretRef' : 'configMapRef',
            name: trim(x.name)
          }))
          .filter((x) => x.name)
          .filter((x) => {
            const k = `${x.type}/${x.name}`
            if (seen.has(k)) return false
            seen.add(k)
            return true
          })

        // volumeMounts：只保留 name+mountPath；去重
        const vmSeen = new Set()
        const volumeMounts = (v.volumeMounts || [])
          .map((m) => ({
            name: trim(m.name),
            mountPath: trim(m.mountPath),
            subPath: trim(m.subPath),
            readOnly: !!m.readOnly
          }))
          .filter((m) => m.name && m.mountPath)
          .filter((m) => {
            const k = `${m.name}|${m.mountPath}`
            if (vmSeen.has(k)) return false
            vmSeen.add(k)
            return true
          })

        // 资源：保持原样（builder 决定是否输出）
        const resources = {
          requests: { ...(v.resources?.requests || {}) },
          limits: { ...(v.resources?.limits || {}) }
        }

        // probes：子组件已做“启用且完整才返回对象，否则 null”
        const probes = clone(v.probes)

        // 基本字段
        const name = trim(v.basic?.name)
        const image = trim(v.basic?.image)
        const pullPolicy = trim(v.basic?.pullPolicy)

        store.form.container = {
          name,
          image,
          pullPolicy,
          ports,
          env,
          envFrom,
          volumeMounts,
          resources,
          probes
        }
      }
    }
  }
}
</script>

<style scoped>
.row {
  display: flex;
  align-items: center;
}
.mr8 {
  margin-right: 8px;
}
</style>

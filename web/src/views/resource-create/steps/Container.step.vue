<template>
  <div class="step-container">
    <el-form label-width="120px">
      <el-form-item label="容器名">
        <el-input v-model="local.name" placeholder="不填默认 container" />
      </el-form-item>

      <el-form-item label="镜像" required>
        <el-input
          v-model="local.image"
          placeholder="如 nginx:1.25 或 ghcr.io/org/img:tag"
        />
      </el-form-item>

      <el-form-item label="拉取策略">
        <el-select v-model="local.pullPolicy" placeholder="可选">
          <el-option label="IfNotPresent" value="IfNotPresent" />
          <el-option label="Always" value="Always" />
          <el-option label="Never" value="Never" />
        </el-select>
      </el-form-item>

      <!-- 端口 -->
      <el-form-item label="端口">
        <el-table :data="local.ports" border size="mini" style="width: 100%">
          <el-table-column label="名称" width="160">
            <template slot-scope="{ row }">
              <el-input v-model="row.name" placeholder="可选" />
            </template>
          </el-table-column>
          <el-table-column label="容器端口" width="160">
            <template slot-scope="{ row }">
              <el-input v-model.number="row.containerPort" placeholder="80" />
            </template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                size="mini"
                type="text"
                @click="local.ports.splice($index, 1)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="local.ports.push({ name: '', containerPort: null })"
          >+ 添加端口</el-button>
        </div>
      </el-form-item>

      <!-- 环境变量 -->
      <el-form-item label="环境变量">
        <el-table :data="local.env" border size="mini" style="width: 100%">
          <el-table-column label="名称" width="200">
            <template slot-scope="{ row }">
              <el-input v-model="row.name" placeholder="ENV_NAME" />
            </template>
          </el-table-column>
          <el-table-column label="值">
            <template slot-scope="{ row }">
              <el-input v-model="row.value" placeholder="ENV_VALUE" />
            </template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                size="mini"
                type="text"
                @click="local.env.splice($index, 1)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="local.env.push({ name: '', value: '' })"
          >+ 添加变量</el-button>
        </div>
      </el-form-item>

      <!-- ✅ envFrom（批量引入 ConfigMap/Secret） -->
      <el-form-item label="envFrom">
        <el-table :data="local.envFrom" border size="mini" style="width: 100%">
          <el-table-column label="类型" width="160">
            <template slot-scope="{ row }">
              <el-select v-model="row.type" placeholder="选择">
                <el-option label="ConfigMap" value="configMapRef" />
                <el-option label="Secret" value="secretRef" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="名称">
            <template slot-scope="{ row }">
              <el-input v-model="row.name" placeholder="如 common-config" />
            </template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                size="mini"
                type="text"
                @click="local.envFrom.splice($index, 1)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="local.envFrom.push({ type: 'configMapRef', name: '' })"
          >+ 添加来源</el-button>
        </div>
        <div class="hint">将渲染为 <code>containers[].envFrom</code></div>
      </el-form-item>

      <!-- ✅ 卷挂载（容器内） -->
      <el-form-item label="卷挂载">
        <el-table
          :data="local.volumeMounts"
          border
          size="mini"
          style="width: 100%"
        >
          <el-table-column label="卷名" width="220">
            <template slot-scope="{ row }">
              <el-input
                v-model="row.name"
                placeholder="如 user-avatar-storage"
              />
            </template>
          </el-table-column>
          <el-table-column label="挂载路径">
            <template slot-scope="{ row }">
              <el-input
                v-model="row.mountPath"
                placeholder="/app/img/UserAvatar"
              />
            </template>
          </el-table-column>
          <el-table-column label="subPath" width="200">
            <template slot-scope="{ row }">
              <el-input v-model="row.subPath" placeholder="可选" />
            </template>
          </el-table-column>
          <el-table-column label="只读" width="100">
            <template slot-scope="{ row }">
              <el-switch v-model="row.readOnly" />
            </template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                size="mini"
                type="text"
                @click="local.volumeMounts.splice($index, 1)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="
              local.volumeMounts.push({
                name: '',
                mountPath: '',
                readOnly: false,
                subPath: '',
              })
            "
          >+ 添加挂载</el-button>
        </div>
        <div class="hint">
          卷实体在「存储」步骤配置（Pod 级
          <code>spec.volumes</code>），这里是容器内的挂载点（<code>volumeMounts</code>）。
        </div>
      </el-form-item>

      <!-- 资源 -->
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
    </el-form>
  </div>
</template>

<script>
import store from '../stores/createForm.store'

export default {
  name: 'ContainerStep',
  data() {
    const s = store.form.container
    return {
      local: {
        name: s.name || '',
        image: s.image || '',
        pullPolicy: s.pullPolicy || '',
        ports: (s.ports && s.ports.map((p) => ({ ...p }))) || [],
        env: (s.env && s.env.map((e) => ({ ...e }))) || [],
        envFrom: (s.envFrom && s.envFrom.map((x) => ({ ...x }))) || [],
        volumeMounts:
          (s.volumeMounts && s.volumeMounts.map((m) => ({ ...m }))) || [],
        resources: {
          requests: { ...(s.resources?.requests || {}) },
          limits: { ...(s.resources?.limits || {}) }
        }
      }
    }
  },
  watch: {
    local: {
      deep: true,
      handler(v) {
        // ---- 统一清洗 & 去重 ----
        const trimStr = (x) => String(x || '').trim()

        // ports：只保留有 containerPort 的；强转 Number
        const ports = (v.ports || [])
          .map((p) => ({
            name: trimStr(p.name),
            containerPort: Number(p.containerPort)
          }))
          .filter((p) => Number.isFinite(p.containerPort))

        // env：只保留有 name 的；同名去重（后者覆盖前者）
        const envMap = new Map();
        (v.env || []).forEach((e) => {
          const name = trimStr(e.name)
          if (!name) return
          envMap.set(name, { name, value: String(e.value ?? '') })
        })
        const env = Array.from(envMap.values())

        // envFrom：只保留有 name；规范化 type；去重（type/name 组合）
        const envFromSeen = new Set()
        const envFrom = (v.envFrom || [])
          .map((x) => ({
            type: x.type === 'secretRef' ? 'secretRef' : 'configMapRef',
            name: trimStr(x.name)
          }))
          .filter((x) => x.name)
          .filter((x) => {
            const k = `${x.type}/${x.name}`
            if (envFromSeen.has(k)) return false
            envFromSeen.add(k)
            return true
          })

        // volumeMounts：只保留 name+mountPath；去重（按 name+mountPath）
        const vmSeen = new Set()
        const volumeMounts = (v.volumeMounts || [])
          .map((m) => ({
            name: trimStr(m.name),
            mountPath: trimStr(m.mountPath),
            subPath: trimStr(m.subPath),
            readOnly: !!m.readOnly
          }))
          .filter((m) => m.name && m.mountPath)
          .filter((m) => {
            const k = `${m.name}|${m.mountPath}`
            if (vmSeen.has(k)) return false
            vmSeen.add(k)
            return true
          })

        // 资源：保持原样（让 builder 决定是否输出）
        const resources = {
          requests: { ...(v.resources?.requests || {}) },
          limits: { ...(v.resources?.limits || {}) }
        }

        // 顶部基本字段
        const name = trimStr(v.name)
        const image = trimStr(v.image)
        const pullPolicy = trimStr(v.pullPolicy)

        store.form.container = {
          name,
          image,
          pullPolicy,
          ports,
          env,
          envFrom,
          volumeMounts,
          resources
        }
      }
    }
  }
}
</script>

<style scoped>
.row {
  display: flex;
}
.mr8 {
  margin-right: 8px;
}
.mt8 {
  margin-top: 8px;
}
.hint {
  color: #909399;
  font-size: 12px;
  margin-top: 6px;
}
</style>

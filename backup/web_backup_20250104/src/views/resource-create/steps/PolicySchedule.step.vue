<template>
  <div class="step-policy">
    <el-form label-width="140px">
      <!-- HPA -->
      <el-form-item label="启用 HPA">
        <el-switch v-model="hpa.enabled" />
      </el-form-item>
      <template v-if="hpa.enabled">
        <el-form-item label="最小副本">
          <el-input-number v-model="hpa.minReplicas" :min="1" />
        </el-form-item>
        <el-form-item label="最大副本">
          <el-input-number v-model="hpa.maxReplicas" :min="1" />
        </el-form-item>
        <el-form-item label="目标 CPU 利用率(%)">
          <el-input-number v-model="cpuTarget" :min="1" :max="100" />
        </el-form-item>
      </template>

      <el-divider />

      <!-- PDB -->
      <el-form-item label="启用 PDB">
        <el-switch v-model="pdb.enabled" />
      </el-form-item>
      <template v-if="pdb.enabled">
        <el-form-item label="模式">
          <el-select v-model="pdb.mode">
            <el-option label="minAvailable" value="minAvailable" />
            <el-option label="maxUnavailable" value="maxUnavailable" />
          </el-select>
        </el-form-item>
        <el-form-item label="值（如 1 或 10%）">
          <el-input v-model="pdb.value" />
        </el-form-item>
      </template>

      <el-divider />

      <!-- 调度：nodeSelector -->
      <el-form-item label="nodeSelector">
        <el-table
          :data="nodeSelectorRows"
          border
          size="mini"
          style="width: 100%"
        >
          <el-table-column label="Key" width="220">
            <template
              slot-scope="{ row }"
            ><el-input
              v-model="row.key"
            /></template>
          </el-table-column>
          <el-table-column label="Value">
            <template
              slot-scope="{ row }"
            ><el-input
              v-model="row.value"
            /></template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                type="text"
                size="mini"
                @click="nodeSelectorRows.splice($index, 1)"
              >删</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="nodeSelectorRows.push({ key: '', value: '' })"
          >+ 添加</el-button>
        </div>
      </el-form-item>

      <!-- ✅ 调度：imagePullSecrets -->
      <el-form-item label="imagePullSecrets">
        <el-table :data="ips" border size="mini" style="width: 100%">
          <el-table-column label="Name">
            <template
              slot-scope="{ row }"
            ><el-input
              v-model="row.name"
              placeholder="如 dockerhub-regcred"
            /></template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                type="text"
                size="mini"
                @click="ips.splice($index, 1)"
              >删</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="ips.push({ name: '' })"
          >+ 添加</el-button>
        </div>
      </el-form-item>

      <!-- ✅ 调度：Anti-Affinity（软约束 preferred） -->
      <el-form-item label="Anti-Affinity(软)">
        <div class="row">
          <el-input
            v-model="paa.labelApp"
            placeholder="matchLabels.app，例如 user"
            class="mr8"
          />
          <el-input
            v-model="paa.topologyKey"
            placeholder="kubernetes.io/hostname"
            class="mr8"
          />
          <el-input-number v-model.number="paa.weight" :min="1" :max="100" />
        </div>
        <div class="hint">
          生成 preferredDuringSchedulingIgnoredDuringExecution 规则；留空 app
          则不生成。
        </div>
      </el-form-item>
    </el-form>
  </div>
</template>

<script>
import store from '../stores/createForm.store'
export default {
  name: 'PolicyScheduleStep',
  data() {
    const h = store.form.hpa || {}
    const p = store.form.pdb || {}
    const s = store.form.sched || {}

    // 取 Anti-Affinity 的 preferredDuringSchedulingIgnoredDuringExecution 第一条
    const preferred =
      s?.affinity?.podAntiAffinity
        ?.preferredDuringSchedulingIgnoredDuringExecution || []
    const first = preferred[0]

    return {
      // HPA
      hpa: {
        enabled: !!h.enabled,
        minReplicas: Number(h.minReplicas) || 1,
        maxReplicas: Number(h.maxReplicas) || 1,
        metrics: Array.isArray(h.metrics) ? [...h.metrics] : []
      },
      cpuTarget:
        Array.isArray(h.metrics) && h.metrics[0]?.target
          ? Number(h.metrics[0].target)
          : null,

      // PDB
      pdb: { enabled: !!p.enabled, mode: p.mode || '', value: p.value || '' },

      // nodeSelector
      nodeSelectorRows: Object.keys(s.nodeSelector || {}).map((k) => ({
        key: k,
        value: String(s.nodeSelector[k])
      })),

      // imagePullSecrets
      ips: (s.imagePullSecrets || []).map((x) =>
        typeof x === 'string' ? { name: x } : { name: x?.name || '' }
      ),

      // Anti-Affinity(软) 编辑模型
      paa: {
        labelApp: first?.podAffinityTerm?.labelSelector?.matchLabels?.app || '',
        topologyKey:
          first?.podAffinityTerm?.topologyKey || 'kubernetes.io/hostname',
        weight: Number(first?.weight) || 100
      }
    }
  },
  watch: {
    // HPA：数值收敛 + metrics 同步
    hpa: {
      deep: true,
      handler(v) {
        const enabled = !!v.enabled
        let min = Number(v.minReplicas) || 1
        min = Math.max(1, Math.trunc(min))
        let max = Number(v.maxReplicas) || min
        max = Math.max(min, Math.trunc(max))

        const metrics =
          enabled && this.cpuTarget
            ? [{ type: 'cpuUtilization', target: Number(this.cpuTarget) }]
            : []

        store.form.hpa = {
          enabled,
          minReplicas: min,
          maxReplicas: max,
          metrics
        }
      }
    },
    cpuTarget() {
      const h = this.hpa
      const metrics =
        h.enabled && this.cpuTarget
          ? [{ type: 'cpuUtilization', target: Number(this.cpuTarget) }]
          : []
      store.form.hpa = {
        enabled: h.enabled,
        minReplicas: Math.max(1, Math.trunc(Number(h.minReplicas) || 1)),
        maxReplicas: Math.max(
          Math.max(1, Math.trunc(Number(h.minReplicas) || 1)),
          Math.trunc(Number(h.maxReplicas) || 1)
        ),
        metrics
      }
    },

    // PDB 同步
    pdb: {
      deep: true,
      handler(v) {
        store.form.pdb = { ...v }
      }
    },

    // nodeSelector 同步
    nodeSelectorRows: {
      deep: true,
      handler(rows) {
        const nodeSelector = {}
        rows.forEach(({ key, value }) => {
          const k = (key || '').trim()
          if (k) nodeSelector[k] = value || ''
        })
        store.form.sched = { ...(store.form.sched || {}), nodeSelector }
      }
    },

    // imagePullSecrets 同步
    ips: {
      deep: true,
      handler(list) {
        const cleaned = (list || [])
          .map((x) => ({ name: String(x?.name || '').trim() }))
          .filter((x) => x.name)
        store.form.sched = {
          ...(store.form.sched || {}),
          imagePullSecrets: cleaned
        }
      }
    },

    // Anti-Affinity(软) -> 正确的 k8s 字段名
    paa: {
      deep: true,
      handler(v) {
        const app = String(v.labelApp || '').trim()
        const prev = store.form.sched || {}

        if (!app) {
          // 不填 app 就清空 affinity，避免输出脏字段
          store.form.sched = { ...prev, affinity: null }
          return
        }

        const weight = Math.max(
          1,
          Math.min(100, Math.trunc(Number(v.weight) || 100))
        )
        const topologyKey = v.topologyKey || 'kubernetes.io/hostname'
        const affinity = {
          podAntiAffinity: {
            // ⬇️ 关键：使用标准字段名
            preferredDuringSchedulingIgnoredDuringExecution: [
              {
                weight,
                podAffinityTerm: {
                  labelSelector: { matchLabels: { app }},
                  topologyKey
                }
              }
            ]
          }
        }
        store.form.sched = { ...prev, affinity }
      }
    }
  }
}
</script>

<style scoped>
.mt8 {
  margin-top: 8px;
}
.row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.mr8 {
  margin-right: 8px;
}
.hint {
  color: #909399;
  font-size: 12px;
  margin-top: 4px;
}
</style>

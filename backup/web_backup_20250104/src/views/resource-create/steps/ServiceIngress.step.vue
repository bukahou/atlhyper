<template>
  <div class="step-svc-ing">
    <el-form label-width="120px">
      <!-- Service -->
      <el-form-item label="创建 Service">
        <el-switch v-model="svc.enabled" />
      </el-form-item>

      <template v-if="svc.enabled">
        <el-form-item label="名称">
          <el-input v-model="svc.name" :placeholder="`默认 ${basicName}`" />
        </el-form-item>

        <el-form-item label="命名空间">
          <el-input
            v-model="svc.namespace"
            :placeholder="`默认 ${basicNs || 'default'}`"
          />
        </el-form-item>

        <el-form-item label="类型">
          <el-select v-model="svc.type" placeholder="ClusterIP">
            <el-option label="ClusterIP" value="ClusterIP" />
            <el-option label="NodePort" value="NodePort" />
            <el-option label="LoadBalancer" value="LoadBalancer" />
          </el-select>
        </el-form-item>

        <el-form-item label="端口映射">
          <div class="toolbar">
            <el-button
              size="mini"
              @click="importFromContainerPorts"
            >从容器端口导入</el-button>
          </div>

          <el-table :data="svc.ports" border size="mini" style="width: 100%">
            <el-table-column label="名称" width="160">
              <template slot-scope="{ row }">
                <el-input v-model="row.name" placeholder="可选" />
              </template>
            </el-table-column>

            <el-table-column label="Service端口" width="140">
              <template slot-scope="{ row }">
                <el-input v-model.number="row.port" placeholder="80" />
              </template>
            </el-table-column>

            <el-table-column label="目标端口" width="140">
              <template slot-scope="{ row }">
                <el-input
                  v-model.number="row.targetPort"
                  placeholder="默认等于 Service 端口"
                />
              </template>
            </el-table-column>

            <el-table-column label="协议" width="120">
              <template slot-scope="{ row }">
                <el-select v-model="row.protocol">
                  <el-option label="TCP" value="TCP" />
                  <el-option label="UDP" value="UDP" />
                </el-select>
              </template>
            </el-table-column>

            <el-table-column
              v-if="svc.type === 'NodePort'"
              label="NodePort"
              width="140"
            >
              <template slot-scope="{ row }">
                <el-input v-model.number="row.nodePort" placeholder="可选" />
              </template>
            </el-table-column>

            <el-table-column width="90" label="操作">
              <template slot-scope="{ $index }">
                <el-button
                  size="mini"
                  type="text"
                  @click="svc.ports.splice($index, 1)"
                >删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="mt8">
            <el-button
              size="mini"
              @click="
                svc.ports.push({
                  name: '',
                  port: null,
                  targetPort: null,
                  protocol: 'TCP',
                })
              "
            >+ 添加端口</el-button>
          </div>
        </el-form-item>
      </template>

      <el-divider />

      <!-- Ingress -->
      <el-form-item label="创建 Ingress">
        <el-switch v-model="ing.enabled" />
      </el-form-item>

      <template v-if="ing.enabled">
        <el-form-item label="名称">
          <el-input v-model="ing.name" :placeholder="`默认 ${basicName}-ing`" />
        </el-form-item>

        <el-form-item label="命名空间">
          <el-input
            v-model="ing.namespace"
            :placeholder="`默认 ${basicNs || 'default'}`"
          />
        </el-form-item>

        <el-form-item label="IngressClass">
          <el-input
            v-model="ing.ingressClassName"
            placeholder="可选，如 nginx"
          />
        </el-form-item>

        <el-form-item label="规则">
          <div class="toolbar">
            <el-button
              size="mini"
              @click="addBasicRule"
            >添加一个基础规则</el-button>
          </div>

          <el-table :data="ing.rules" border size="mini" style="width: 100%">
            <el-table-column label="Host" width="220">
              <template slot-scope="{ row }">
                <el-input v-model="row.host" placeholder="example.com" />
              </template>
            </el-table-column>

            <el-table-column label="PathType" width="140">
              <template slot-scope="{ row }">
                <el-select v-model="row.pathType">
                  <el-option label="Prefix" value="Prefix" />
                  <el-option label="Exact" value="Exact" />
                </el-select>
              </template>
            </el-table-column>

            <el-table-column label="Paths">
              <template slot-scope="{ row }">
                <el-table
                  :data="row.paths"
                  border
                  size="mini"
                  style="width: 100%"
                >
                  <el-table-column label="Path" width="200">
                    <template slot-scope="{ row: p }">
                      <el-input v-model="p.path" placeholder="/" />
                    </template>
                  </el-table-column>

                  <el-table-column label="Service 名" width="200">
                    <template slot-scope="{ row: p }">
                      <el-input
                        v-model="p.serviceName"
                        :placeholder="svcNameDefault"
                      />
                    </template>
                  </el-table-column>

                  <el-table-column label="Service 端口" width="160">
                    <template slot-scope="{ row: p }">
                      <el-input
                        v-model.number="p.servicePort"
                        placeholder="80"
                      />
                    </template>
                  </el-table-column>

                  <el-table-column width="90" label="操作">
                    <template slot-scope="{ $index }">
                      <el-button
                        type="text"
                        size="mini"
                        @click="row.paths.splice($index, 1)"
                      >删除</el-button>
                    </template>
                  </el-table-column>
                </el-table>

                <div class="mt8">
                  <el-button
                    size="mini"
                    @click="
                      row.paths.push({
                        path: '/',
                        serviceName: '',
                        servicePort: firstSvcPort || 80,
                      })
                    "
                  >+ 添加 Path</el-button>
                </div>
              </template>
            </el-table-column>

            <el-table-column width="90" label="操作">
              <template slot-scope="{ $index }">
                <el-button
                  size="mini"
                  type="text"
                  @click="ing.rules.splice($index, 1)"
                >删除规则</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="mt8">
            <el-button size="mini" @click="addBasicRule">+ 添加规则</el-button>
          </div>
        </el-form-item>

        <el-form-item label="TLS">
          <el-switch v-model="ing.tlsEnabled" />
        </el-form-item>

        <template v-if="ing.tlsEnabled">
          <el-form-item label="Secret 名">
            <el-input v-model="ing.tlsSecretName" placeholder="如 my-tls" />
          </el-form-item>
          <el-form-item label="TLS Host">
            <el-input v-model="ing.tlsHost" placeholder="可选：example.com" />
          </el-form-item>
        </template>
      </template>
    </el-form>
  </div>
</template>

<script>
import store from '../stores/createForm.store'

// 简单 debounce
function debounce(fn, delay = 120) {
  let t = null
  return function(...args) {
    clearTimeout(t)
    t = setTimeout(() => fn.apply(this, args), delay)
  }
}

export default {
  name: 'ServiceIngressStep',
  data() {
    return {
      svc: JSON.parse(
        JSON.stringify(
          store.form.svc || {
            enabled: false,
            name: '',
            namespace: '',
            type: 'ClusterIP',
            ports: []
          }
        )
      ),
      ing: JSON.parse(
        JSON.stringify(
          store.form.ing || {
            enabled: false,
            name: '',
            namespace: '',
            ingressClassName: '',
            rules: [],
            tlsEnabled: false,
            tlsSecretName: '',
            tlsHost: ''
          }
        )
      )
    }
  },
  computed: {
    basicName() {
      return store.form?.basic?.name || ''
    },
    basicNs() {
      return store.form?.basic?.namespace || ''
    },
    svcNameDefault() {
      return (this.svc.name || this.basicName || '').trim() || 'svc-name'
    },
    firstSvcPort() {
      const p = (this.svc.ports || []).find((x) => Number(x?.port) > 0)
      return p ? Number(p.port) : 80
    },
    containerPorts() {
      return (store.form?.container?.ports || []).filter(
        (p) => Number(p?.containerPort) > 0
      )
    }
  },
  watch: {
    // 用去抖版本，键入时更顺滑
    svc: {
      deep: true,
      handler() {
        this._flushToStoreDebounced()
      }
    },
    ing: {
      deep: true,
      handler() {
        this._flushToStoreDebounced()
      }
    },
    // 关闭 TLS 时清空字段，避免 YAML 留下无效值
    'ing.tlsEnabled'(v) {
      if (!v) {
        this.ing.tlsSecretName = ''
        this.ing.tlsHost = ''
        this._flushToStoreDebounced()
      }
    }
  },
  created() {
    // 包装一个去抖版写回
    this._flushToStoreDebounced = debounce(this._flushToStore, 120)
  },
  methods: {
    importFromContainerPorts() {
      // 从容器端口生成 service 端口映射（去重按端口号+协议）
      const existed = new Set(
        (this.svc.ports || []).map(
          (p) => `${Number(p.port)}|${p.protocol || 'TCP'}`
        )
      );
      (this.containerPorts || []).forEach((cp) => {
        const port = Number(cp.containerPort)
        const key = `${port}|TCP`
        if (!port || existed.has(key)) return
        this.svc.ports.push({
          name: cp.name || `p-${port}`,
          port,
          targetPort: port,
          protocol: 'TCP'
        })
        existed.add(key)
      })
      this.$message.success('已从容器端口导入')
    },
    addBasicRule() {
      this.ing.rules.push({
        host: '',
        pathType: 'Prefix',
        paths: [
          {
            path: '/',
            serviceName: this.svcNameDefault,
            servicePort: this.firstSvcPort || 80
          }
        ]
      })
    },
    _dedupeSvcPorts(ports = []) {
      const map = new Map() // key: port|protocol
      ports.forEach((p) => {
        const key = `${Number(p.port)}|${p.protocol || 'TCP'}`
        if (!map.has(key)) map.set(key, p)
      })
      return Array.from(map.values())
    },
    _validateNodePort(rows = []) {
      // 典型范围 30000-32767（集群可自定义），这里若超出仅提示
      if (this.svc.type !== 'NodePort') return
      rows.forEach((p) => {
        if (p.nodePort && (p.nodePort < 30000 || p.nodePort > 32767)) {
          this.$message.warning(
            `NodePort ${p.nodePort} 看起来不在 30000-32767 常见范围内（如集群自定义可忽略）`
          )
        }
      })
    },
    // 规范化写回
    _flushToStore() {
      // --- 清洗 Service ---
      let svcPorts = (this.svc.ports || [])
        .map((p) => ({
          name: (p.name || '').trim(),
          port: Number(p.port) || null,
          targetPort: Number(p.targetPort) || null,
          protocol: p.protocol === 'UDP' ? 'UDP' : 'TCP',
          nodePort:
            this.svc.type === 'NodePort'
              ? Number(p.nodePort) || null
              : undefined
        }))
        .filter((p) => p.port)
        .map((p) => ({ ...p, targetPort: p.targetPort || p.port }))

      // 去重（端口+协议）
      svcPorts = this._dedupeSvcPorts(svcPorts)
      // NodePort 简单校验提示
      this._validateNodePort(svcPorts)

      const svc = {
        ...this.svc,
        name: (this.svc.name || '').trim(),
        namespace: (this.svc.namespace || '').trim(),
        type: this.svc.type || 'ClusterIP',
        ports: svcPorts
      }

      // --- 清洗 Ingress ---
      const rules = (this.ing.rules || [])
        .map((r) => {
          const pathType = r.pathType === 'Exact' ? 'Exact' : 'Prefix'
          const paths = (r.paths || [])
            .map((p) => ({
              path: (p.path || '/').trim() || '/',
              serviceName: (p.serviceName || this.svcNameDefault).trim(),
              servicePort: Number(p.servicePort) || this.firstSvcPort || 80
            }))
            .filter((p) => p.servicePort > 0)
          return {
            host: (r.host || '').trim(),
            pathType,
            paths
          }
        })
        .filter((r) => (r.paths || []).length > 0)

      const ing = {
        ...this.ing,
        name: (this.ing.name || '').trim(),
        namespace: (this.ing.namespace || '').trim(),
        ingressClassName: (this.ing.ingressClassName || '').trim(),
        rules,
        tlsEnabled: !!this.ing.tlsEnabled,
        tlsSecretName: (this.ing.tlsSecretName || '').trim(),
        tlsHost: (this.ing.tlsHost || '').trim()
      }

      // 写回 store
      store.form.svc = JSON.parse(JSON.stringify(svc))
      store.form.ing = JSON.parse(JSON.stringify(ing))
    }
  }
}
</script>

<style scoped>
.mt8 {
  margin-top: 8px;
}
.toolbar {
  margin-bottom: 6px;
}
</style>

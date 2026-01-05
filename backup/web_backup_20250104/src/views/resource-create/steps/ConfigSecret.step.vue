<template>
  <div class="step-config-secret">
    <el-tabs v-model="tab">
      <!-- ConfigMap（键值对编辑） -->
      <el-tab-pane label="ConfigMap" name="cm">
        <el-table :data="configRows" border size="mini" style="width: 100%">
          <el-table-column label="Key" width="260">
            <template slot-scope="{ row }">
              <el-input v-model="row.key" />
            </template>
          </el-table-column>
          <el-table-column label="Value">
            <template slot-scope="{ row }">
              <el-input v-model="row.value" type="textarea" :rows="1" />
            </template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                type="text"
                size="mini"
                @click="configRows.splice($index, 1)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="configRows.push({ key: '', value: '' })"
          >+ 添加项</el-button>
        </div>
      </el-tab-pane>

      <!-- Secret（键值对编辑） -->
      <el-tab-pane label="Secret" name="secret">
        <el-form label-width="90px" class="mb8">
          <el-form-item label="Type">
            <el-input v-model="secret.type" placeholder="如 Opaque（可选）" />
          </el-form-item>
        </el-form>

        <el-table :data="secretRows" border size="mini" style="width: 100%">
          <el-table-column label="Key" width="260">
            <template slot-scope="{ row }">
              <el-input v-model="row.key" />
            </template>
          </el-table-column>
          <el-table-column label="Value（原文）">
            <template slot-scope="{ row }">
              <el-input v-model="row.value" type="textarea" :rows="1" />
            </template>
          </el-table-column>
          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                type="text"
                size="mini"
                @click="secretRows.splice($index, 1)"
              >删除</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="mt8">
          <el-button
            size="mini"
            @click="secretRows.push({ key: '', value: '' })"
          >+ 添加项</el-button>
        </div>
        <div class="tip">（生成 YAML 时由 builder 决定是否 base64 编码）</div>
      </el-tab-pane>
    </el-tabs>

    <!-- 新增：从 ConfigMap/Secret 批量引入环境变量 -->
    <el-divider>envFrom（批量引入环境变量）</el-divider>
    <el-table :data="envFrom" border size="mini" style="width: 100%">
      <el-table-column label="类型" width="160">
        <template slot-scope="{ row }">
          <el-select v-model="row.type" placeholder="选择类型">
            <el-option label="ConfigMap" value="configMapRef" />
            <el-option label="Secret" value="secretRef" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="名称">
        <template slot-scope="{ row }">
          <el-input
            v-model="row.name"
            placeholder="如：common-config / dockerhub-cred"
          />
        </template>
      </el-table-column>
      <el-table-column width="90" label="操作">
        <template slot-scope="{ $index }">
          <el-button
            type="text"
            size="mini"
            @click="envFrom.splice($index, 1)"
          >删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="mt8">
      <el-button
        size="mini"
        @click="envFrom.push({ type: 'configMapRef', name: '' })"
      >+ 添加来源</el-button>
    </div>
    <div class="tip">
      将渲染为 <code>containers[].envFrom</code>，例如
      <code>- configMapRef: { name: common-config }</code>。
    </div>
  </div>
</template>

<script>
import store from '../stores/createForm.store'
export default {
  name: 'ConfigSecretStep',
  data() {
    // 将对象形式的 configmap/secret 转成可编辑的行数组
    const cm = store.form.configmap || {}
    const sec = store.form.secret || { type: 'Opaque', data: {}}

    return {
      tab: 'cm',

      // ConfigMap
      configRows: Object.keys(cm).map((k) => ({
        key: k,
        value: String(cm[k])
      })),

      // Secret
      secret: { type: sec.type || 'Opaque' },
      secretRows: Object.keys(sec.data || {}).map((k) => ({
        key: k,
        value: String(sec.data[k])
      })),

      // ✅ envFrom：与 container.envFrom 同步
      envFrom: (store.form.container.envFrom || []).map((x) => ({ ...x }))
    }
  },
  watch: {
    // ConfigMap -> 对象
    configRows: {
      deep: true,
      handler(rows) {
        const obj = {}
        rows.forEach(({ key, value }) => {
          const k = (key || '').trim()
          if (k) obj[k] = value || ''
        })
        store.form.configmap = obj
      }
    },

    // Secret type
    secret: {
      deep: true,
      handler(v) {
        store.form.secret = {
          ...(store.form.secret || {}),
          type: v.type || 'Opaque',
          data: store.form.secret?.data || {}
        }
      }
    },

    // Secret key/val -> 对象
    secretRows: {
      deep: true,
      handler(rows) {
        const data = {}
        rows.forEach(({ key, value }) => {
          const k = (key || '').trim()
          if (k) data[k] = value || ''
        })
        store.form.secret = { type: this.secret.type || 'Opaque', data }
      }
    },

    // ✅ envFrom（数组）-> 写回 container.envFrom
    envFrom: {
      deep: true,
      handler(list) {
        const cleaned = (list || [])
          .map((x) => ({
            type: x.type === 'secretRef' ? 'secretRef' : 'configMapRef',
            name: (x.name || '').trim()
          }))
          .filter((x) => x.name) // 过滤空名
        store.form.container.envFrom = cleaned
      }
    }
  }
}
</script>

<style scoped>
.mt8 {
  margin-top: 8px;
}
.mb8 {
  margin-bottom: 8px;
}
.tip {
  color: #909399;
  font-size: 12px;
  margin-top: 6px;
}
</style>

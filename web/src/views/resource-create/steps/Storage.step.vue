<template>
  <div class="step-storage">
    <el-form label-width="120px">
      <!-- PVC 快速声明 -->
      <el-form-item label="使用 PVC">
        <el-switch v-model="usePVC" />
      </el-form-item>

      <template v-if="usePVC">
        <el-form-item label="PVC 名">
          <el-input v-model="pvc.name" placeholder="如 data-pvc" />
        </el-form-item>
        <el-form-item label="容量">
          <el-input v-model="pvc.size" placeholder="如 5Gi" />
        </el-form-item>
        <el-form-item label="AccessModes">
          <el-select v-model="pvc.accessModes" multiple placeholder="选择">
            <el-option label="ReadWriteOnce" value="ReadWriteOnce" />
            <el-option label="ReadOnlyMany" value="ReadOnlyMany" />
            <el-option label="ReadWriteMany" value="ReadWriteMany" />
          </el-select>
        </el-form-item>
        <el-form-item label="StorageClass">
          <el-input v-model="pvc.storageClassName" placeholder="可选" />
        </el-form-item>
      </template>

      <el-divider />

      <!-- ✅ Volumes -->
      <el-form-item label="Volumes（Pod）">
        <div class="toolbar">
          <el-button size="mini" @click="addHostPath">+ hostPath</el-button>
          <el-button size="mini" @click="addEmptyDir">+ emptyDir</el-button>
          <el-button size="mini" @click="addPVCRef">+ pvc 引用</el-button>
        </div>

        <el-table :data="vols" border size="mini" style="width: 100%">
          <el-table-column label="名称" width="220">
            <template slot-scope="{ row }">
              <el-input
                v-model="row.name"
                placeholder="如 user-avatar-storage"
              />
            </template>
          </el-table-column>

          <el-table-column label="类型" width="160">
            <template slot-scope="{ row }">
              <el-select v-model="row.type">
                <el-option label="hostPath" value="hostPath" />
                <el-option label="emptyDir" value="emptyDir" />
                <el-option label="pvc" value="pvc" />
              </el-select>
            </template>
          </el-table-column>

          <el-table-column label="参数">
            <template slot-scope="{ row }">
              <!-- hostPath -->
              <div v-if="row.type === 'hostPath'" class="row">
                <el-input
                  v-model="row.hostPath.path"
                  placeholder="路径，如 /data"
                  class="mr8"
                />
                <el-input
                  v-model="row.hostPath.type"
                  placeholder="Directory/File…（可选）"
                />
              </div>

              <!-- emptyDir -->
              <div v-else-if="row.type === 'emptyDir'" class="row">
                <el-input
                  v-model="row.emptyDir.medium"
                  placeholder="medium，如 Memory（可选）"
                  class="mr8"
                />
                <el-input
                  v-model="row.emptyDir.sizeLimit"
                  placeholder="sizeLimit，如 1Gi（可选）"
                />
              </div>

              <!-- pvc -->
              <div v-else-if="row.type === 'pvc'" class="row">
                <el-input
                  v-model="row.pvc.claimName"
                  placeholder="PVC 名（claimName）"
                />
              </div>
            </template>
          </el-table-column>

          <el-table-column width="90" label="操作">
            <template slot-scope="{ $index }">
              <el-button
                type="text"
                size="mini"
                @click="vols.splice($index, 1)"
              >删</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="hint">
          注：这里只定义 <code>volumes</code>。容器内
          <code>volumeMounts</code> 请在「容器配置」步骤里维护。
        </div>
      </el-form-item>
    </el-form>
  </div>
</template>

<script>
import store from '../stores/createForm.store'

export default {
  name: 'StorageStep',
  data() {
    const p = store.form.pvc || {}
    return {
      usePVC: !!(p.name || p.size || (p.accessModes && p.accessModes.length)),
      pvc: {
        name: p.name || '',
        size: p.size || '',
        accessModes: Array.isArray(p.accessModes) ? [...p.accessModes] : [],
        storageClassName: p.storageClassName || ''
      },
      vols: (store.form.volumes || []).map((v) =>
        JSON.parse(JSON.stringify(v))
      )
    }
  },
  watch: {
    usePVC(val) {
      store.form.pvc = val ? { ...this.pvc } : {}
    },
    pvc: {
      deep: true,
      handler(v) {
        if (this.usePVC) store.form.pvc = JSON.parse(JSON.stringify(v))
      }
    },
    vols: {
      deep: true,
      handler() {
        this.flushVolumes()
      }
    }
  },
  methods: {
    addHostPath() {
      this.vols.push({
        name: '',
        type: 'hostPath',
        hostPath: { path: '', type: '' }
      })
    },
    addEmptyDir() {
      this.vols.push({
        name: '',
        type: 'emptyDir',
        emptyDir: { medium: '', sizeLimit: '' }
      })
    },
    addPVCRef() {
      this.vols.push({
        name: '',
        type: 'pvc',
        pvc: { claimName: this.pvc.name || '' }
      })
    },
    flushVolumes() {
      const cleaned = this.vols
        .map((v) => {
          if (!v.name) return null
          const out = { name: v.name, type: v.type }
          if (v.type === 'hostPath' && v.hostPath?.path) {
            out.hostPath = { ...v.hostPath }
          } else if (v.type === 'emptyDir') {
            out.emptyDir = { ...v.emptyDir }
          } else if (v.type === 'pvc' && v.pvc?.claimName) {
            out.pvc = { claimName: v.pvc.claimName }
          }
          return out
        })
        .filter(Boolean)
      store.form.volumes = JSON.parse(JSON.stringify(cleaned))
    }
  }
}
</script>

<style scoped>
.row {
  display: flex;
  gap: 8px;
  align-items: center;
}
.mr8 {
  margin-right: 8px;
}
.mt8 {
  margin-top: 8px;
}
.toolbar {
  margin-bottom: 8px;
  display: flex;
  gap: 8px;
}
.hint {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
}
</style>

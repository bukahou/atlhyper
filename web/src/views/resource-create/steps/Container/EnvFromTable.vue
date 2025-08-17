<template>
  <el-form-item label="envFrom">
    <el-table :data="list" border size="mini" style="width: 100%">
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
          <el-button size="mini" type="text" @click="list.splice($index, 1)"
            >删除</el-button
          >
        </template>
      </el-table-column>
    </el-table>
    <div class="mt8">
      <el-button
        size="mini"
        @click="list.push({ type: 'configMapRef', name: '' })"
        >+ 添加来源</el-button
      >
    </div>
    <div class="hint">将渲染为 <code>containers[].envFrom</code></div>
  </el-form-item>
</template>

<script>
export default {
  name: "EnvFromTable",
  props: { value: { type: Array, default: () => [] } },
  data() {
    return { list: this.value.map((x) => ({ ...x })) };
  },
  watch: {
    list: {
      deep: true,
      handler(v) {
        this.$emit(
          "input",
          v.map((x) => ({ ...x }))
        );
      },
    },
    value: {
      deep: true,
      handler(v) {
        this.list = v.map((x) => ({ ...x }));
      },
    },
  },
};
</script>

<style scoped>
.mt8 {
  margin-top: 8px;
}
.hint {
  color: #909399;
  font-size: 12px;
  margin-top: 6px;
}
</style>

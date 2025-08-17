<template>
  <el-form-item label="卷挂载">
    <el-table :data="list" border size="mini" style="width: 100%">
      <el-table-column label="卷名" width="220">
        <template slot-scope="{ row }">
          <el-input v-model="row.name" placeholder="如 user-avatar-storage" />
        </template>
      </el-table-column>
      <el-table-column label="挂载路径">
        <template slot-scope="{ row }">
          <el-input v-model="row.mountPath" placeholder="/app/img/UserAvatar" />
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
          <el-button size="mini" type="text" @click="list.splice($index, 1)"
            >删除</el-button
          >
        </template>
      </el-table-column>
    </el-table>
    <div class="mt8">
      <el-button
        size="mini"
        @click="
          list.push({ name: '', mountPath: '', readOnly: false, subPath: '' })
        "
        >+ 添加挂载</el-button
      >
    </div>
    <div class="hint">
      卷实体在「存储」步骤配置（Pod 级
      <code>spec.volumes</code
      >），这里是容器内的挂载点（<code>volumeMounts</code>）。
    </div>
  </el-form-item>
</template>

<script>
export default {
  name: "VolumeMountsTable",
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

<template>
  <el-form-item label="端口">
    <el-table :data="list" border size="mini" style="width: 100%">
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
          <el-button size="mini" type="text" @click="list.splice($index, 1)"
            >删除</el-button
          >
        </template>
      </el-table-column>
    </el-table>
    <div class="mt8">
      <el-button
        size="mini"
        @click="list.push({ name: '', containerPort: null })"
        >+ 添加端口</el-button
      >
    </div>
  </el-form-item>
</template>

<script>
export default {
  name: "PortsTable",
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
</style>

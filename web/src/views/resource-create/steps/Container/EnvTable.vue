<template>
  <el-form-item label="环境变量">
    <el-table :data="list" border size="mini" style="width: 100%">
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
          <el-button size="mini" type="text" @click="list.splice($index, 1)"
            >删除</el-button
          >
        </template>
      </el-table-column>
    </el-table>
    <div class="mt8">
      <el-button size="mini" @click="list.push({ name: '', value: '' })"
        >+ 添加变量</el-button
      >
    </div>
  </el-form-item>
</template>

<script>
export default {
  name: "EnvTable",
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

<template>
  <div class="page-container">
    <!-- ✅ 顶部统计卡片区域 -->
    <div class="card-row">
      <CardStat
        icon-bg="bg1"
        :number="totalUsers"
        number-color="color1"
        title="总用户"
      >
        <template #icon>
          <i class="fas fa-users" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg2"
        :number="adminUsers"
        number-color="color1"
        title="管理员"
      >
        <template #icon>
          <i class="fas fa-user-shield" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg3"
        :number="normalUsers"
        number-color="color1"
        title="一般用户"
      >
        <template #icon>
          <i class="fas fa-user" />
        </template>
      </CardStat>

      <CardStat
        icon-bg="bg4"
        :number="failedOperations"
        number-color="color1"
        title="操作失败次数"
      >
        <template #icon>
          <i class="fas fa-exclamation-triangle" />
        </template>
      </CardStat>
    </div>

    <!-- ✅ 用户表格组件 -->
    <UserTable :users="userList" @view-user="handleViewUser" />
  </div>
</template>

<script>
import CardStat from "@/components/Atlhyper/CardStat.vue";
import UserTable from "@/components/Atlhyper/UserTable.vue";
import { listUsers } from "@/api/user"; // ✅ 导入 API

export default {
  name: "UserView",
  components: {
    CardStat,
    UserTable,
  },
  data() {
    return {
      totalUsers: 0,
      adminUsers: 0,
      normalUsers: 0,
      failedOperations: 0,
      userList: [],
    };
  },
  created() {
    this.fetchUsers();
  },
  methods: {
    async fetchUsers() {
      try {
        const res = await listUsers();
        if (res.code === 20000 && Array.isArray(res.data)) {
          const rawUsers = res.data;

          // ✅ 格式化数据（role 字段翻译）
          const mappedUsers = rawUsers.map((u) => ({
            username: u.Username,
            displayName: u.DisplayName,
            email: u.Email,
            createdAt: u.CreatedAt,
            role: this.translateRole(u.Role), // 转换角色字段
          }));

          // ✅ 设置表格数据
          this.userList = mappedUsers;

          // ✅ 统计卡片数据
          this.totalUsers = rawUsers.length;
          this.adminUsers = rawUsers.filter(
            (u) => u.Role === 2 || u.Role === 3
          ).length;
          this.normalUsers = rawUsers.filter((u) => u.Role === 1).length;
          // ⚠️ 操作失败次数暂留为 0，除非你有相关统计
          this.failedOperations = 0;
        } else {
          this.$message.error("获取用户失败：" + res.message);
        }
      } catch (err) {
        console.error(err);
        this.$message.error("用户请求异常");
      }
    },
    translateRole(roleNum) {
      switch (roleNum) {
        case 1:
          return "普通用户";
        case 2:
          return "管理员";
        case 3:
          return "超级管理员";
        default:
          return "未知";
      }
    },
    handleViewUser(user) {
      console.log("查看用户：", user);
    },
  },
};
</script>

<style scoped>
.page-container {
  padding: 35px 32px;
}
.card-row {
  display: flex;
  flex-wrap: wrap;
  gap: 65px; /* 你想要的间距值 */
  margin-bottom: 24px;
}
</style>

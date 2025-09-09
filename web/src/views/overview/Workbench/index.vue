<template>
  <div class="workspace-page">
    <WelcomeCard username="AtlHyper" :todo="'2/10'" :project="8" :team="300" />

    <el-row :gutter="16" class="mt16 two-col-stretch">
      <el-col class="left-col" :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
        <SlackConfigCard />
        <div class="mt16">
          <TodosMyCard :username="loginUsername" />
        </div>
      </el-col>

      <el-col class="right-col" :xs="24" :sm="24" :md="12" :lg="12" :xl="12">
        <TodosAllCard />
      </el-col>
    </el-row>
  </div>
</template>

<script>
import WelcomeCard from '@/components/Workspace/WelcomeCard.vue'
import SlackConfigCard from './components/SlackConfigCard.vue'
import TodosMyCard from './components/TodosMyCard.vue'
import TodosAllCard from './components/TodosAllCard.vue'

export default {
  components: { WelcomeCard, SlackConfigCard, TodosMyCard, TodosAllCard },
  data() {
    return { loginUsername: 'admin' }
  }
}
</script>

<style scoped>
.workspace-page {
  padding: 35px 32px;
}
.mt16 {
  margin-top: 16px;
}

.two-col-stretch {
  align-items: stretch;
}
.two-col-stretch .el-col {
  display: flex;
  flex-direction: column;
}

.right-col > * {
  flex: 1;
  display: flex;
  flex-direction: column;
}
</style>

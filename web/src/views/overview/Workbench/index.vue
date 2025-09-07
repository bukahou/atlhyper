<template>
  <div class="workspace-page">
    <WelcomeCard username="AtlHyper" :todo="'2/10'" :project="8" :team="300" />
    <SlackConfigCard />
  </div>
</template>

<script>
import WelcomeCard from '@/components/Workspace/WelcomeCard.vue'
import SlackConfigCard from './components/SlackConfigCard.vue' // 同级 components 目录

export default {
  components: { WelcomeCard, SlackConfigCard }
}
</script>

<style scoped>
.workspace-page {
  padding: 35px 32px;
}
</style>

<template>
  <div class="workspace-page">
    <WelcomeCard username="AtlHyper" :todo="'2/10'" :project="8" :team="300" />
  </div>
</template>

<script>
import WelcomeCard from '@/components/Workspace/WelcomeCard.vue'

export default {
  components: {
    WelcomeCard
  }
}
</script>

<style scoped>
.workspace-page {
  padding: 35px 32px;
}
</style>

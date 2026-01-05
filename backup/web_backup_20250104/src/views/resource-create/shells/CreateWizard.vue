<template>
  <div class="rc-wizard">
    <!-- 步骤条 -->
    <el-steps :active="active" finish-status="success" class="mb16">
      <el-step v-for="(s, i) in steps" :key="i" :title="s" />
    </el-steps>

    <div class="toolbar">
      <el-button type="primary" icon="el-icon-view" @click="refreshYaml">
        刷新 YAML
      </el-button>
    </div>

    <el-row :gutter="16">
      <!-- 左：步骤表单 -->
      <el-col :sm="24" :md="12">
        <el-card shadow="never">
          <keep-alive>
            <component :is="currentComp" class="step-body" />
          </keep-alive>
        </el-card>

        <div class="actions">
          <el-button :disabled="active === 0" @click="prev">上一步</el-button>
          <el-button type="primary" @click="nextOrGen">
            {{ active < steps.length - 1 ? "下一步" : "生成 YAML（最终）" }}
          </el-button>
        </div>
      </el-col>

      <!-- 右：YAML 预览 -->
      <el-col :sm="24" :md="12">
        <YamlDock :yaml="store.yaml" :results="store.results" />
      </el-col>
    </el-row>
  </div>
</template>

<script>
import store from '../stores/createForm.store'

// 步骤表单（去掉“配置与密钥”，在容器步骤里用 envFrom 即可）
import BasicInfo from '../steps/BasicInfo.step.vue'
import ContainerStep from '../steps/Container.step.vue'
import ServiceIngress from '../steps/ServiceIngress.step.vue'
import StorageStep from '../steps/Storage.step.vue'
import PolicySchedule from '../steps/PolicySchedule.step.vue'
import LabelsAnnotations from '../steps/LabelsAnnotations.step.vue'

// 右侧预览
import YamlDock from '../components/YamlDock.vue'

// ✅ 新结构的生成器入口（目录下的 index.js）
import { generateYamlStrict } from '../services/builders/deployment'
// / 等价写法：import { generateYamlStrict } from "../services/builders/deployment/index.js";

export default {
  name: 'CreateWizard',
  components: {
    BasicInfo,
    ContainerStep,
    ServiceIngress,
    StorageStep,
    PolicySchedule,
    LabelsAnnotations,
    YamlDock
  },
  data() {
    return {
      store,
      steps: [
        '基础信息',
        '容器配置',
        '服务与入口',
        '存储',
        '策略与调度',
        '标签与注解'
      ],
      active: 0
    }
  },
  computed: {
    currentComp() {
      return [
        'BasicInfo',
        'ContainerStep',
        'ServiceIngress',
        'StorageStep',
        'PolicySchedule',
        'LabelsAnnotations'
      ][this.active]
    }
  },
  methods: {
    prev() {
      if (this.active > 0) this.active--
    },
    nextOrGen() {
      if (this.active < this.steps.length - 1) {
        this.active++
      } else {
        // 最终生成 YAML（走新的 builders/deployment/index.js）
        this.store.yaml = generateYamlStrict(this.store.form)
        if (this.store.yaml) {
          this.$message.success('已生成最终 YAML')
        } else {
          this.$message.info('请至少填写：名称 与 容器镜像')
        }
      }
    },
    refreshYaml() {
      this.store.yaml = generateYamlStrict(this.store.form)
      if (this.store.yaml) {
        this.$message.success('YAML 已刷新')
      } else {
        this.$message.info('请输入名称和镜像后再刷新')
      }
    }
  }
}
</script>

<style scoped>
.rc-wizard {
  padding: 16px;
}
.mb16 {
  margin-bottom: 16px;
}
.toolbar {
  margin-bottom: 12px;
}
.actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
}
.step-body {
  padding: 4px;
}
</style>

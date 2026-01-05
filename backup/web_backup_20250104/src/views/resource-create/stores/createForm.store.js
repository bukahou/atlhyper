// resource-create/stores/createForm.store.js
import Vue from 'vue'

/**
 * 单一真源（全局可观察状态）
 * - workload: 当前创建的资源类型（先固定为 "Deployment"，以后可由路由传入）
 * - form: 各步骤收集到的字段（通用化，后续可扩展到 StatefulSet/Job 等）
 * - yaml/results/summaryCards/testing/applying: 右侧预览 & 提交相关状态
 *
 * 使用方式（在任意步骤/壳里）：
 *   import store from "../stores/createForm.store"
 *   store.form.basic.name = "my-app"
 *   store.yaml = "...yaml text..."
 */
const state = Vue.observable({
  workload: 'Deployment',

  form: {
    // 基础信息
    basic: { name: '', namespace: '' },

    // ✅ 副本数（Deployment 常用；以后也可被 HPA 的 min/max 参考）
    replicas: 1,

    // 容器与镜像
    container: {
      name: '',
      image: '',
      pullPolicy: '',
      ports: [], // [{ name, containerPort }]
      env: [], // [{ name, value }]
      // ✅ 从 ConfigMap/Secret 批量引入环境变量
      //    例：[{ type:'configMapRef'|'secretRef', name:'common-config' }]
      envFrom: [],
      // ✅ 卷挂载（容器级别）
      //    例：[{ name:'vol-name', mountPath:'/path', subPath:'', readOnly:false }]
      volumeMounts: [],
      resources: {
        // { requests: {cpu, memory}, limits: {cpu, memory} }
        requests: {},
        limits: {}
      }
    },

    // Service & Ingress（可选）
    svc: {
      enabled: false,
      name: '',
      namespace: '',
      type: 'ClusterIP',
      ports: [] // [{ name, port, targetPort, protocol, nodePort }]
    },
    ing: {
      enabled: false,
      name: '',
      namespace: '',
      ingressClassName: '',
      rules: [], // [{ host, pathType, paths:[{path, serviceName, servicePort}] }]
      tlsEnabled: false,
      tlsSecretName: '',
      tlsHost: ''
    },

    // 配置与密钥（可选）
    configmap: {}, // { key: value }
    secret: {
      // { type, data: {key: rawValue} } 由 builder 决定是否 base64
      type: 'Opaque',
      data: {}
    },

    // 存储（可选）
    pvc: {
      // { name, size, accessModes:[], storageClassName }
    },
    vct: {}, // 预留：volumeClaimTemplates（StatefulSet 用）

    // ✅ Pod 级 volumes（hostPath/emptyDir/pvc…）
    //    例：[
    //      { name:'user-avatar-storage', type:'hostPath', hostPath:{ path:'/xxx', type:'Directory' } },
    //      { name:'cache', type:'emptyDir', emptyDir:{ medium:'Memory', sizeLimit:'1Gi' } },
    //      { name:'data', type:'pvc', pvc:{ claimName:'data-pvc' } }
    //    ]
    volumes: [],

    // 策略与调度（可选）
    hpa: {
      enabled: false,
      minReplicas: null,
      maxReplicas: null,
      metrics: [] // [{ type:'cpuUtilization'|'memoryUtilization', target:number }]
    },
    pdb: { enabled: false, mode: '', value: '' },
    sched: {
      nodeSelector: {}, // { key: value }
      // ✅ 镜像拉取凭据
      //    例：[{ name:'dockerhub-regcred' }]
      imagePullSecrets: [],
      // ✅ 亲和/反亲和（完整结构对象，交由 builder 递归渲染为 YAML）
      //    例：
      //    {
      //      podAntiAffinity: {
      //        preferred: [
      //          {
      //            weight: 100,
      //            podAffinityTerm: {
      //              labelSelector: { matchLabels: { app: 'user' } },
      //              topologyKey: 'kubernetes.io/hostname'
      //            }
      //          }
      //        ]
      //      }
      //    }
      affinity: null
      // 预留：tolerations 等
    },
    np: {}, // NetworkPolicy 预留

    // 标签与注解
    labels: [], // [{ key, value }]
    annotations: [], // [{ key, value }]
    scope: [] // 自定义扩展位
  },

  // 右侧 YAML 预览与操作反馈
  yaml: '', // 预览文本
  results: [], // [{ type:'success'|'error'|'info', message }]
  summaryCards: [], // 审核页概要卡片
  testing: false, // Dry-run 状态
  applying: false // Apply 状态
})

export default state

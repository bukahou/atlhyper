// js/api_config.js
// const API_BASE_URL = "";

const ENV = "dev"; // dev / prod

const API_BASE_URL = ENV === "dev" ? "http://localhost:8081" : "";

const API_ENDPOINTS = {
  cluster: {
    overview: `${API_BASE_URL}/uiapi/cluster/overview`,
  },
  deployment: {
    listAll: `${API_BASE_URL}/uiapi/deployment/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/deployment/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/deployment/get/${ns}/${name}`,
    listUnavailable: `${API_BASE_URL}/uiapi/deployment/list/unavailable`,
    listProgressing: `${API_BASE_URL}/uiapi/deployment/list/progressing`,
    scale: `${API_BASE_URL}/uiapi/deployment/scale`,
  },
  event: {
    listAll: `${API_BASE_URL}/uiapi/event/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/event/list/by-namespace/${ns}`,
    listByObject: (ns, kind, name) =>
      `${API_BASE_URL}/uiapi/event/list/by-object/${ns}/${kind}/${name}`,
    summaryByType: `${API_BASE_URL}/uiapi/event/summary/type`,
    listRecent: (days) =>
      `${API_BASE_URL}/uiapi/event/list/recent?days=${days}`,
  },
  ingress: {
    listAll: `${API_BASE_URL}/uiapi/ingress/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/ingress/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/ingress/get/${ns}/${name}`,
    listReady: `${API_BASE_URL}/uiapi/ingress/list/ready`,
  },
  namespace: {
    list: `${API_BASE_URL}/uiapi/namespace/list`,
    get: (name) => `${API_BASE_URL}/uiapi/namespace/get/${name}`,
    listActive: `${API_BASE_URL}/uiapi/namespace/list/active`,
    listTerminating: `${API_BASE_URL}/uiapi/namespace/list/terminating`,
    summaryStatus: `${API_BASE_URL}/uiapi/namespace/summary/status`,
  },
  node: {
    list: `${API_BASE_URL}/uiapi/node/list`,
    metrics: `${API_BASE_URL}/uiapi/node/metrics`,
    overview: `${API_BASE_URL}/uiapi/node/overview`,
    getByName: (name) => `${API_BASE_URL}/uiapi/node/get/${name}`,
  },
  pod: {
    listAll: `${API_BASE_URL}/uiapi/pod/list`,
    listByNamespace: (ns) => `${API_BASE_URL}/uiapi/pod/list/${ns}`,
    summary: `${API_BASE_URL}/uiapi/pod/summary`,
    usage: `${API_BASE_URL}/uiapi/pod/usage`,
    // ✅ 新增：获取简略 Pod 列表（PodInfo 结构）
    listBrief: `${API_BASE_URL}/uiapi/pod/list/brief`,
    describe: (ns, name) => `${API_BASE_URL}/uiapi/pod/describe/${ns}/${name}`,
    restart: (namespace, name) =>
      `${API_BASE_URL}/uiapi/pod/restart/${namespace}/${name}`, // ✅ 新增重启接口
    logs: (namespace, name) =>
      `${API_BASE_URL}/uiapi/pod/logs/${namespace}/${name}`,
  },
  configmap: {
    listAll: `${API_BASE_URL}/uiapi/configmap/list`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/configmap/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/configmap/get/${ns}/${name}`,
    // ✅ 告警系统配置
    getAlertSettings: `${API_BASE_URL}/uiapi/configmap/alert/get`, // 获取配置
    updateSlack: `${API_BASE_URL}/uiapi/configmap/alert/slack`, // 更新 Slack
    updateWebhook: `${API_BASE_URL}/uiapi/configmap/alert/webhook`, // 更新 Webhook 开关
    updateMail: `${API_BASE_URL}/uiapi/configmap/alert/mail`, // 更新 Mail（含多人）
  },
  service: {
    listAll: `${API_BASE_URL}/uiapi/service/list/all`,
    listByNamespace: (ns) =>
      `${API_BASE_URL}/uiapi/service/list/by-namespace/${ns}`,
    get: (ns, name) => `${API_BASE_URL}/uiapi/service/get/${ns}/${name}`,
    listExternal: `${API_BASE_URL}/uiapi/service/list/external`,
    listHeadless: `${API_BASE_URL}/uiapi/service/list/headless`,
  },
};

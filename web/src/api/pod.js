import request from '@/utils/request'

export function getPodOverview(clusterId) {
  return request({
    url: '/uiapi/pod/overview',
    method: 'post',
    data: { ClusterID: clusterId }
  })
}

export function getPodDetail(clusterId, namespace, podName) {
  return request({
    url: '/uiapi/pod/detail',
    method: 'post',
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
      PodName: podName
    }
  })
}

export function getPodLogs(clusterId, namespace, Pod, tailLines) {
  return request({
    url: '/uiapi/ops/pod/logs',
    method: 'post',
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
      Pod: Pod,
      TailLines: tailLines
    }
  })
}

export function getPodRestart(clusterId, namespace, pod) {
  return request({
    url: '/uiapi/ops/pod/restart',
    method: 'post',
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
      Pod: pod
    }
  })
}

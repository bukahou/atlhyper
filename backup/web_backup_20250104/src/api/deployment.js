import request from '@/utils/request'

export function getDeploymentOverview(clusterId) {
  return request({
    url: '/uiapi/deployment/overview',
    method: 'post',
    data: { ClusterID: clusterId }
  })
}

export function getDeploymentDetail(clusterId, namespace, name) {
  return request({
    url: '/uiapi/deployment/detail',
    method: 'post',
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
      Name: name
    }
  })
}

export function getDeploymentupdateImage(
  clusterId,
  namespace,
  kind,
  name,
  newImage,
  oldImage
) {
  return request({
    url: '/uiapi/ops/workload/updateImage',
    method: 'post',
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
      Kind: kind,
      Name: name,
      NewImage: newImage,
      OldImage: oldImage
    }
  })
}

export function getDeploymentScale(clusterId, namespace, kind, name, replicas) {
  return request({
    url: '/uiapi/ops/workload/scale',
    method: 'post',
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
      Kind: kind,
      Name: name,
      Replicas: replicas
    }
  })
}

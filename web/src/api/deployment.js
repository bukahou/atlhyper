import request from '@/utils/request'

export function getAllDeployments() {
  return request({
    url: '/uiapi/deployment/list/all',
    method: 'get'
  })
}

// src/api/deployment.js
export function updateDeployment({ namespace, name, replicas, image }) {
  return request({
    url: '/uiapi/deployment-ops/scale',
    method: 'post',
    data: {
      namespace,
      name,
      replicas,
      image
    }
  })
}

export function getDeploymentByName(namespace, name) {
  return request({
    url: `/uiapi/deployment/get/${namespace}/${name}`,
    method: 'get'
  })
}

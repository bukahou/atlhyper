import { prune } from '../core/prune'
import { toYaml } from '../core/toYaml'
import { kvArrayToObject, selectorFrom } from './util'

import { buildContainers } from './parts/containers'
import { buildPodSpec } from './parts/podSpec'
import { buildService } from './parts/service'
import { buildIngress } from './parts/ingress'
import { buildHpa } from './parts/hpa'
import { buildPdb } from './parts/pdb'

/**
 * 统一入口：把表单 -> 多文档 YAML
 */
export function generateYamlStrict(form) {
  const name = (form?.basic?.name || '').trim()
  const namespace = (form?.basic?.namespace || '').trim()
  const image = (form?.container?.image || '').trim()
  if (!name || !image) return ''

  const labels = kvArrayToObject(form?.labels || [])
  const annotations = kvArrayToObject(form?.annotations || [])
  const selector = selectorFrom(labels, name)

  // Deployment 对象
  const deployment = {
    apiVersion: 'apps/v1',
    kind: 'Deployment',
    metadata: {
      name,
      ...(namespace ? { namespace } : {}),
      ...(Object.keys(annotations).length ? { annotations } : {}),
      ...(Object.keys(labels).length ? { labels } : {})
    },
    spec: {
      ...(Number(form?.replicas) > 0
        ? { replicas: Number(form.replicas) }
        : {}),
      selector: { matchLabels: selector },
      template: {
        metadata: {
          labels: selector // 与 selector 对齐
        },
        spec: (() => {
          const pod = buildPodSpec(form)
          pod.containers = buildContainers(form)
          return pod
        })()
      }
    }
  }

  // 其他可选文档
  const ctx = { name, selector, labels, annotations }
  const docs = [
    deployment,
    buildService(form, ctx),
    buildIngress(form, ctx),
    buildHpa(form, ctx),
    buildPdb(form, ctx)
  ]
    .map(prune)
    .filter(Boolean)
    .map(toYaml)

  return docs.join('---\n')
}

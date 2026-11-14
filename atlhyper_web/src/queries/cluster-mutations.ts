/**
 * 集群写操作的描述（重启 Pod、封锁 / 解封节点、Deployment 扩缩容与换镜像）
 *
 * 每个工厂返回一份 mutationOptions，组件用 useMutation 执行。
 *
 * 写操作只是把指令放进队列，由 Agent 稍后执行。所以刷新分两次：
 * - 指令下发成功后立即让该类资源的列表与详情失效（这里的 onSuccess）
 * - 约 2 秒后指令执行完，再失效一次（组件里用 useDelayedInvalidate，范围取 clusterKeys.resourceScope）
 */
import { mutationOptions, type QueryClient } from "@tanstack/react-query";
import {
  cordonNode,
  restartPod,
  scaleDeployment,
  uncordonNode,
  updateDeploymentImage,
} from "@/datasource/cluster";
import { clusterKeys, type ClusterResourceKind, type NamespacedTarget } from "./keys";

/** 让某类资源的列表与详情重新取数。不等它们取完：指令已经下发，按钮不必为了刷新继续转圈 */
function invalidateResource(client: QueryClient, kind: ClusterResourceKind): void {
  for (const queryKey of clusterKeys.resourceScope(kind)) {
    void client.invalidateQueries({ queryKey });
  }
}

/** 重启 Pod（删除后由控制器重建） */
export function restartPodMutation(clusterId: string) {
  return mutationOptions({
    mutationFn: (pod: NamespacedTarget) =>
      restartPod({ ClusterID: clusterId, Namespace: pod.namespace, Pod: pod.name }),
    onSuccess: (_command, _pod, _onMutateResult, context) => invalidateResource(context.client, "pod"),
  });
}

export interface NodeSchedulingChange {
  nodeName: string;
  /** 节点现在是否可调度：可调度就封锁，已封锁就解封 */
  currentlySchedulable: boolean;
}

/** 切换节点的可调度状态（cordon / uncordon） */
export function nodeSchedulingMutation(clusterId: string) {
  return mutationOptions({
    mutationFn: ({ nodeName, currentlySchedulable }: NodeSchedulingChange) =>
      currentlySchedulable
        ? cordonNode({ ClusterID: clusterId, Node: nodeName })
        : uncordonNode({ ClusterID: clusterId, Node: nodeName }),
    onSuccess: (_command, _change, _onMutateResult, context) => invalidateResource(context.client, "node"),
  });
}

export interface DeploymentScaleChange extends NamespacedTarget {
  replicas: number;
}

/** Deployment 扩缩容 */
export function scaleDeploymentMutation(clusterId: string) {
  return mutationOptions({
    mutationFn: ({ namespace, name, replicas }: DeploymentScaleChange) =>
      scaleDeployment({ ClusterID: clusterId, Namespace: namespace, Name: name, Kind: "Deployment", Replicas: replicas }),
    onSuccess: (_command, _change, _onMutateResult, context) => invalidateResource(context.client, "deployment"),
  });
}

export interface DeploymentImageChange extends NamespacedTarget {
  containerName: string;
  oldImage: string;
  newImage: string;
}

/** Deployment 更换某个容器的镜像 */
export function updateDeploymentImageMutation(clusterId: string) {
  return mutationOptions({
    mutationFn: ({ namespace, name, containerName, oldImage, newImage }: DeploymentImageChange) =>
      updateDeploymentImage({
        ClusterID: clusterId,
        Namespace: namespace,
        Name: name,
        Kind: "Deployment",
        ContainerName: containerName,
        NewImage: newImage,
        OldImage: oldImage,
      }),
    onSuccess: (_command, _change, _onMutateResult, context) => invalidateResource(context.client, "deployment"),
  });
}

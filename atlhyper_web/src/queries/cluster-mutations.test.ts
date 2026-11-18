import { QueryClient, type MutationOptions } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升
const datasource = vi.hoisted(() => ({
  restartPod: vi.fn(),
  cordonNode: vi.fn(),
  uncordonNode: vi.fn(),
  scaleDeployment: vi.fn(),
  updateDeploymentImage: vi.fn(),
}));
vi.mock("@/datasource/cluster", () => datasource);

import {
  nodeSchedulingMutation,
  restartPodMutation,
  scaleDeploymentMutation,
  updateDeploymentImageMutation,
} from "./cluster-mutations";
import { clusterKeys } from "./keys";

const podList = clusterKeys.resourceList("pod", "c1", "api");
const podDetail = clusterKeys.resourceDetail("pod", "c1", "api", { namespace: "ns1", name: "web-1" });
const nodeList = clusterKeys.resourceList("node", "c1", "api");
const nodeDetail = clusterKeys.resourceDetail("node", "c1", "api", { name: "node-1" });
const deploymentList = clusterKeys.resourceList("deployment", "c1", "api");
const deploymentDetail = clusterKeys.resourceDetail("deployment", "c1", "api", { namespace: "ns1", name: "web" });
const allKeys = [podList, podDetail, nodeList, nodeDetail, deploymentList, deploymentDetail];

/** 已有缓存的 QueryClient：用来观察写操作成功后哪些查询被标记为失效 */
function clientWithCachedQueries(): QueryClient {
  const client = new QueryClient();
  for (const key of allKeys) client.setQueryData(key, { cached: true });
  return client;
}

function invalidatedKeys(client: QueryClient): (readonly unknown[])[] {
  return allKeys.filter((key) => client.getQueryState(key)?.isInvalidated);
}

function runMutation<TResult, TVariables>(
  client: QueryClient,
  options: MutationOptions<TResult, Error, TVariables>,
  variables: TVariables,
) {
  return client.getMutationCache().build(client, options).execute(variables);
}

beforeEach(() => {
  for (const writer of Object.values(datasource)) {
    writer.mockReset();
    writer.mockResolvedValue({ data: { commandId: "cmd-1" } });
  }
});

describe("restartPodMutation", () => {
  it("向数据源下发重启指令，成功后让 Pod 的列表与详情失效", async () => {
    const client = clientWithCachedQueries();

    await runMutation(client, restartPodMutation("c1"), { namespace: "ns1", name: "web-1" });

    expect(datasource.restartPod).toHaveBeenCalledWith({ ClusterID: "c1", Namespace: "ns1", Pod: "web-1" });
    expect(invalidatedKeys(client)).toEqual([podList, podDetail]);
  });

  it("指令下发失败时不动缓存", async () => {
    datasource.restartPod.mockRejectedValue(new Error("forbidden"));
    const client = clientWithCachedQueries();

    await expect(
      runMutation(client, { ...restartPodMutation("c1"), retry: false }, { namespace: "ns1", name: "web-1" }),
    ).rejects.toThrow("forbidden");

    expect(invalidatedKeys(client)).toEqual([]);
  });
});

describe("nodeSchedulingMutation", () => {
  it("可调度的节点下发封锁，不可调度的下发解封；成功后让 Node 的列表与详情失效", async () => {
    const client = clientWithCachedQueries();

    await runMutation(client, nodeSchedulingMutation("c1"), { nodeName: "node-1", currentlySchedulable: true });
    expect(datasource.cordonNode).toHaveBeenCalledWith({ ClusterID: "c1", Node: "node-1" });
    expect(datasource.uncordonNode).not.toHaveBeenCalled();
    expect(invalidatedKeys(client)).toEqual([nodeList, nodeDetail]);

    await runMutation(client, nodeSchedulingMutation("c1"), { nodeName: "node-1", currentlySchedulable: false });
    expect(datasource.uncordonNode).toHaveBeenCalledWith({ ClusterID: "c1", Node: "node-1" });
  });
});

describe("scaleDeploymentMutation / updateDeploymentImageMutation", () => {
  it("扩缩容：下发副本数，成功后让 Deployment 的列表与详情失效", async () => {
    const client = clientWithCachedQueries();

    await runMutation(client, scaleDeploymentMutation("c1"), { namespace: "ns1", name: "web", replicas: 3 });

    expect(datasource.scaleDeployment).toHaveBeenCalledWith({
      ClusterID: "c1",
      Namespace: "ns1",
      Name: "web",
      Kind: "Deployment",
      Replicas: 3,
    });
    expect(invalidatedKeys(client)).toEqual([deploymentList, deploymentDetail]);
  });

  it("更新镜像：下发容器与新旧镜像，成功后让 Deployment 的列表与详情失效", async () => {
    const client = clientWithCachedQueries();

    await runMutation(client, updateDeploymentImageMutation("c1"), {
      namespace: "ns1",
      name: "web",
      containerName: "app",
      oldImage: "app:1",
      newImage: "app:2",
    });

    expect(datasource.updateDeploymentImage).toHaveBeenCalledWith({
      ClusterID: "c1",
      Namespace: "ns1",
      Name: "web",
      Kind: "Deployment",
      ContainerName: "app",
      NewImage: "app:2",
      OldImage: "app:1",
    });
    expect(invalidatedKeys(client)).toEqual([deploymentList, deploymentDetail]);
  });
});

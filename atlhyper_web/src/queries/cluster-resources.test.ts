import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

// 集群资源的列表与详情工厂形状一致，用表驱动一次覆盖全部 16 类资源。
// vi.mock 会被提升到文件顶部，替身必须用 vi.hoisted 一起提升。
const datasource = vi.hoisted(() => {
  const readerNames = [
    "getPodOverview", "getPodDetail",
    "getNodeOverview", "getNodeDetail",
    "getDeploymentOverview", "getDeploymentDetail",
    "getServiceOverview", "getServiceDetail",
    "getNamespaceOverview", "getNamespaceDetail",
    "getIngressOverview", "getIngressDetail",
    "getStatefulSetList", "getStatefulSetDetail",
    "getDaemonSetList", "getDaemonSetDetail",
    "getJobList", "getJobDetail",
    "getCronJobList", "getCronJobDetail",
    "getPVList", "getPVDetail",
    "getPVCList", "getPVCDetail",
    "getNetworkPolicyList", "getNetworkPolicyDetail",
    "getResourceQuotaList", "getResourceQuotaDetail",
    "getLimitRangeList", "getLimitRangeDetail",
    "getServiceAccountList", "getServiceAccountDetail",
  ] as const;
  const readers = {} as Record<(typeof readerNames)[number], ReturnType<typeof vi.fn>>;
  for (const name of readerNames) readers[name] = vi.fn();
  return readers;
});
vi.mock("@/datasource/cluster", () => datasource);
// mode 按模块 key 返回不同的值，借此确认每个工厂读的是自己那个模块的数据源配置
vi.mock("@/config/data-source", () => ({ getDataSourceMode: (moduleKey: string) => `mode-of-${moduleKey}` }));

import {
  deploymentDetailQuery, deploymentOverviewQuery, ingressDetailQuery, ingressOverviewQuery,
  namespaceDetailQuery, namespaceOverviewQuery, nodeDetailQuery, nodeOverviewQuery,
  podDetailQuery, podOverviewQuery, serviceDetailQuery, serviceOverviewQuery,
} from "./cluster";
import {
  limitRangeDetailQuery, limitRangeListQuery, networkPolicyDetailQuery, networkPolicyListQuery,
  resourceQuotaDetailQuery, resourceQuotaListQuery, serviceAccountDetailQuery, serviceAccountListQuery,
} from "./cluster-policy";
import { pvcDetailQuery, pvcListQuery, pvDetailQuery, pvListQuery } from "./cluster-storage";
import {
  cronJobDetailQuery, cronJobListQuery, daemonSetDetailQuery, daemonSetListQuery,
  jobDetailQuery, jobListQuery, statefulSetDetailQuery, statefulSetListQuery,
} from "./cluster-workload";
import { REFRESH } from "./refresh";

type ReaderName = keyof typeof datasource;

beforeEach(() => {
  for (const reader of Object.values(datasource)) reader.mockReset();
});

// ------------------------------------------------------------
// 列表
// ------------------------------------------------------------

const overviewParams = { ClusterID: "c1" };
const listParams = { cluster_id: "c1" };

const LIST_CASES: { kind: string; factory: (clusterId: string) => object; reader: ReaderName; params: object }[] = [
  { kind: "pod", factory: podOverviewQuery, reader: "getPodOverview", params: overviewParams },
  { kind: "node", factory: nodeOverviewQuery, reader: "getNodeOverview", params: overviewParams },
  { kind: "deployment", factory: deploymentOverviewQuery, reader: "getDeploymentOverview", params: overviewParams },
  { kind: "service", factory: serviceOverviewQuery, reader: "getServiceOverview", params: overviewParams },
  { kind: "namespace", factory: namespaceOverviewQuery, reader: "getNamespaceOverview", params: overviewParams },
  { kind: "ingress", factory: ingressOverviewQuery, reader: "getIngressOverview", params: overviewParams },
  { kind: "statefulset", factory: statefulSetListQuery, reader: "getStatefulSetList", params: listParams },
  { kind: "daemonset", factory: daemonSetListQuery, reader: "getDaemonSetList", params: listParams },
  { kind: "job", factory: jobListQuery, reader: "getJobList", params: listParams },
  { kind: "cronjob", factory: cronJobListQuery, reader: "getCronJobList", params: listParams },
  { kind: "pv", factory: pvListQuery, reader: "getPVList", params: listParams },
  { kind: "pvc", factory: pvcListQuery, reader: "getPVCList", params: listParams },
  { kind: "netpol", factory: networkPolicyListQuery, reader: "getNetworkPolicyList", params: listParams },
  { kind: "quota", factory: resourceQuotaListQuery, reader: "getResourceQuotaList", params: listParams },
  { kind: "limit", factory: limitRangeListQuery, reader: "getLimitRangeList", params: listParams },
  { kind: "sa", factory: serviceAccountListQuery, reader: "getServiceAccountList", params: listParams },
];

// 表里的工厂返回类型各不相同，测试只关心它们共有的这几项
interface FactoryResult {
  queryKey: readonly unknown[];
  enabled?: boolean;
  refetchInterval?: unknown;
}
const asFactoryResult = (options: object) => options as FactoryResult;
const fetchWith = (options: object) => new QueryClient().fetchQuery(options as Parameters<QueryClient["fetchQuery"]>[0]);

describe.each(LIST_CASES)("$kind 列表", ({ kind, factory, reader, params }) => {
  it("没有选定集群时不启用", () => {
    expect(asFactoryResult(factory("")).enabled).toBe(false);
    expect(asFactoryResult(factory("c1")).enabled).toBe(true);
  });

  it("键里带 clusterId 与本模块的 mode", () => {
    expect(asFactoryResult(factory("c1")).queryKey).toEqual([
      "cluster",
      `${kind}-list`,
      { clusterId: "c1", mode: `mode-of-${kind}` },
    ]);
  });

  it("按集群取数，把取消信号交给数据源，返回解包后的数据", async () => {
    const unwrapped = { rows: [] };
    datasource[reader].mockResolvedValue(unwrapped);

    const result = await fetchWith(factory("c1"));

    expect(result).toBe(unwrapped);
    expect(datasource[reader]).toHaveBeenCalledTimes(1);
    const [calledParams, signal] = datasource[reader].mock.calls[0];
    expect(calledParams).toEqual(params);
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("按默认档轮询", () => {
    expect(asFactoryResult(factory("c1")).refetchInterval).toBe(REFRESH.standard);
  });
});

// ------------------------------------------------------------
// 详情（带 namespace 的资源）
// ------------------------------------------------------------

const namespacedParams = { ClusterID: "c1", Namespace: "ns1", Name: "web-1" };

const NAMESPACED_DETAIL_CASES: {
  kind: string;
  factory: (clusterId: string, target: { namespace: string; name: string }) => object;
  reader: ReaderName;
  params: object;
}[] = [
  { kind: "pod", factory: podDetailQuery, reader: "getPodDetail", params: { ClusterID: "c1", Namespace: "ns1", PodName: "web-1" } },
  { kind: "deployment", factory: deploymentDetailQuery, reader: "getDeploymentDetail", params: namespacedParams },
  { kind: "service", factory: serviceDetailQuery, reader: "getServiceDetail", params: namespacedParams },
  { kind: "ingress", factory: ingressDetailQuery, reader: "getIngressDetail", params: namespacedParams },
  { kind: "statefulset", factory: statefulSetDetailQuery, reader: "getStatefulSetDetail", params: namespacedParams },
  { kind: "daemonset", factory: daemonSetDetailQuery, reader: "getDaemonSetDetail", params: namespacedParams },
  { kind: "job", factory: jobDetailQuery, reader: "getJobDetail", params: namespacedParams },
  { kind: "cronjob", factory: cronJobDetailQuery, reader: "getCronJobDetail", params: namespacedParams },
  { kind: "pvc", factory: pvcDetailQuery, reader: "getPVCDetail", params: namespacedParams },
  { kind: "netpol", factory: networkPolicyDetailQuery, reader: "getNetworkPolicyDetail", params: namespacedParams },
  { kind: "quota", factory: resourceQuotaDetailQuery, reader: "getResourceQuotaDetail", params: namespacedParams },
  { kind: "limit", factory: limitRangeDetailQuery, reader: "getLimitRangeDetail", params: namespacedParams },
  { kind: "sa", factory: serviceAccountDetailQuery, reader: "getServiceAccountDetail", params: namespacedParams },
];

describe.each(NAMESPACED_DETAIL_CASES)("$kind 详情", ({ kind, factory, reader, params }) => {
  const target = { namespace: "ns1", name: "web-1" };

  it("没有集群或目标不完整时不启用", () => {
    expect(asFactoryResult(factory("", target)).enabled).toBe(false);
    expect(asFactoryResult(factory("c1", { namespace: "ns1", name: "" })).enabled).toBe(false);
    expect(asFactoryResult(factory("c1", { namespace: "", name: "web-1" })).enabled).toBe(false);
    expect(asFactoryResult(factory("c1", target)).enabled).toBe(true);
  });

  it("键里带 clusterId、mode、namespace 与 name", () => {
    expect(asFactoryResult(factory("c1", target)).queryKey).toEqual([
      "cluster",
      `${kind}-detail`,
      { clusterId: "c1", mode: `mode-of-${kind}`, namespace: "ns1", name: "web-1" },
    ]);
  });

  it("取这一个资源，把取消信号交给数据源，返回解包后的数据", async () => {
    const detail = { name: "web-1" };
    datasource[reader].mockResolvedValue(detail);

    const result = await fetchWith(factory("c1", target));

    expect(result).toBe(detail);
    const [calledParams, signal] = datasource[reader].mock.calls[0];
    expect(calledParams).toEqual(params);
    expect(signal).toBeInstanceOf(AbortSignal);
  });

  it("不轮询（旧实现只在打开弹窗时取一次）", () => {
    expect(asFactoryResult(factory("c1", target)).refetchInterval).toBeUndefined();
  });
});

// ------------------------------------------------------------
// 详情（集群级资源：没有 namespace）
// ------------------------------------------------------------

const CLUSTER_SCOPED_DETAIL_CASES: {
  kind: string;
  factory: (clusterId: string, name: string) => object;
  reader: ReaderName;
  params: object;
}[] = [
  { kind: "node", factory: nodeDetailQuery, reader: "getNodeDetail", params: { ClusterID: "c1", NodeName: "node-1" } },
  { kind: "namespace", factory: namespaceDetailQuery, reader: "getNamespaceDetail", params: { ClusterID: "c1", Namespace: "node-1" } },
  { kind: "pv", factory: pvDetailQuery, reader: "getPVDetail", params: { ClusterID: "c1", Name: "node-1" } },
];

describe.each(CLUSTER_SCOPED_DETAIL_CASES)("$kind 详情", ({ kind, factory, reader, params }) => {
  it("没有集群或名字为空时不启用", () => {
    expect(asFactoryResult(factory("", "node-1")).enabled).toBe(false);
    expect(asFactoryResult(factory("c1", "")).enabled).toBe(false);
    expect(asFactoryResult(factory("c1", "node-1")).enabled).toBe(true);
  });

  it("键里带 clusterId、mode 与 name", () => {
    expect(asFactoryResult(factory("c1", "node-1")).queryKey).toEqual([
      "cluster",
      `${kind}-detail`,
      { clusterId: "c1", mode: `mode-of-${kind}`, namespace: "", name: "node-1" },
    ]);
  });

  it("取这一个资源，把取消信号交给数据源", async () => {
    const detail = { name: "node-1" };
    datasource[reader].mockResolvedValue(detail);

    const result = await fetchWith(factory("c1", "node-1"));

    expect(result).toBe(detail);
    const [calledParams, signal] = datasource[reader].mock.calls[0];
    expect(calledParams).toEqual(params);
    expect(signal).toBeInstanceOf(AbortSignal);
  });
});

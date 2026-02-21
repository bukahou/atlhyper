/**
 * Log Mock 静态数据 — 基于 ClickHouse 真实日志特征生成
 *
 * 6 个 geass 服务, ~1600 条日志
 * 参考: docs/design/active/clickhouse-otel-data-reference.md
 */

import type { LogEntry } from "@/types/model/log";

// ============================================================
// Deterministic random (same pattern as APM mock)
// ============================================================

function seededRandom(seed: string): () => number {
  let h = 0;
  for (let i = 0; i < seed.length; i++) {
    h = (Math.imul(31, h) + seed.charCodeAt(i)) | 0;
  }
  return () => {
    h = (Math.imul(h, 1103515245) + 12345) | 0;
    return ((h >>> 16) & 0x7fff) / 0x7fff;
  };
}

function hexId(rand: () => number, len: number): string {
  const hex = "0123456789abcdef";
  let result = "";
  for (let i = 0; i < len; i++) {
    result += hex[Math.floor(rand() * 16)];
  }
  return result;
}

// ============================================================
// Service definitions (aligned with ClickHouse data)
// ============================================================

interface ServiceDef {
  name: string;
  port: number;
  scopes: ScopeDef[];
  podSuffix: string;
}

interface ScopeDef {
  fullName: string;
  weight: number;  // relative frequency
}

const SERVICES: ServiceDef[] = [
  {
    name: "geass-gateway",
    port: 8080,
    scopes: [
      { fullName: "com.geass.gateway.common.LoggingFilter", weight: 70 },
      { fullName: "com.geass.gateway.util.RemoteCaller", weight: 20 },
      { fullName: "com.geass.gateway.filter.AuthVerifyFilter", weight: 10 },
    ],
    podSuffix: "ff5988887-fmztl",
  },
  {
    name: "geass-media",
    port: 8081,
    scopes: [
      { fullName: "com.geass.media.common.LoggingFilter", weight: 55 },
      { fullName: "com.geass.media.mapper.AnimeMapper", weight: 15 },
      { fullName: "com.geass.media.mapper.AvMapper", weight: 10 },
      { fullName: "com.geass.media.mapper.UraAnimeMapper", weight: 10 },
      { fullName: "com.geass.media.service.impl.MediaServiceImpl", weight: 10 },
    ],
    podSuffix: "6d8b9c7f5-k2m3n",
  },
  {
    name: "geass-auth",
    port: 8085,
    scopes: [
      { fullName: "com.geass.auth.common.LoggingFilter", weight: 90 },
      { fullName: "com.geass.auth.service.impl.TokenServiceImpl", weight: 10 },
    ],
    podSuffix: "4c5d6e7f8-p9q0r",
  },
  {
    name: "geass-favorites",
    port: 8082,
    scopes: [
      { fullName: "com.geass.favorites.common.LoggingFilter", weight: 85 },
      { fullName: "com.geass.favorites.service.impl.FavoritesServiceImpl", weight: 15 },
    ],
    podSuffix: "3b4c5d6e7-s1t2u",
  },
  {
    name: "geass-history",
    port: 8083,
    scopes: [
      { fullName: "com.geass.history.common.LoggingFilter", weight: 85 },
      { fullName: "com.geass.history.service.impl.HistoryServiceImpl", weight: 15 },
    ],
    podSuffix: "2a3b4c5d6-v3w4x",
  },
  {
    name: "geass-user",
    port: 8084,
    scopes: [
      { fullName: "com.geass.user.common.LoggingFilter", weight: 80 },
      { fullName: "com.geass.user.service.impl.UserServiceImpl", weight: 20 },
    ],
    podSuffix: "1z2a3b4c5-y5z6a",
  },
];

// ============================================================
// Endpoint definitions for log body generation
// ============================================================

interface EndpointDef {
  method: string;
  gatewayPath: string;
  downstreamService: string;
  downstreamPath: string;
  weight: number;
}

const ENDPOINTS: EndpointDef[] = [
  { method: "POST", gatewayPath: "/api/history/list", downstreamService: "geass-history", downstreamPath: "/history/v2/list", weight: 20 },
  { method: "POST", gatewayPath: "/api/media/search", downstreamService: "geass-media", downstreamPath: "/media/search", weight: 15 },
  { method: "POST", gatewayPath: "/api/favorites/list", downstreamService: "geass-favorites", downstreamPath: "/favorites/v2/list", weight: 15 },
  { method: "POST", gatewayPath: "/api/favorites/add", downstreamService: "geass-favorites", downstreamPath: "/favorites/add", weight: 10 },
  { method: "GET", gatewayPath: "/api/user/info", downstreamService: "geass-user", downstreamPath: "/user/profile", weight: 10 },
  { method: "POST", gatewayPath: "/api/ura-anime/sort/release", downstreamService: "geass-media", downstreamPath: "/ura-anime/sort/release", weight: 10 },
];

const CLIENT_IPS = [
  "10.42.2.123", "10.42.1.87", "10.42.3.201", "10.42.2.55", "10.42.1.142",
];

const SQL_TEMPLATES = [
  "SELECT * FROM anime WHERE id IN ( ? , ? , ? )",
  "SELECT * FROM av WHERE category = ? ORDER BY created_at DESC LIMIT ?",
  "SELECT * FROM ura_anime WHERE status = ? LIMIT ?",
  "SELECT count(*) FROM favorites WHERE user_id = ?",
  "SELECT * FROM history WHERE user_id = ? ORDER BY updated_at DESC LIMIT ?",
  "INSERT INTO history (user_id, anime_id, progress) VALUES (?, ?, ?)",
];

// ============================================================
// Generate logs for one trace/request
// ============================================================

function makeResource(svc: ServiceDef): Record<string, string> {
  return {
    "service.name": svc.name,
    "service.version": "0.0.1-SNAPSHOT",
    "k8s.pod.name": `${svc.name}-${svc.podSuffix}`,
    "k8s.namespace.name": "default",
  };
}

function pickWeighted<T extends { weight: number }>(items: T[], rand: () => number): T {
  const total = items.reduce((s, i) => s + i.weight, 0);
  let r = rand() * total;
  for (const item of items) {
    r -= item.weight;
    if (r <= 0) return item;
  }
  return items[items.length - 1];
}

function findService(name: string): ServiceDef {
  return SERVICES.find((s) => s.name === name)!;
}

function generateRequestLogs(
  rand: () => number,
  endpoint: EndpointDef,
  baseTime: Date,
  isError: boolean,
): LogEntry[] {
  const logs: LogEntry[] = [];
  const traceId = hexId(rand, 32);
  const clientIp = CLIENT_IPS[Math.floor(rand() * CLIENT_IPS.length)];
  const gateway = findService("geass-gateway");
  const downstream = findService(endpoint.downstreamService);

  let timeMs = baseTime.getTime();
  const latencyMs = Math.round(20 + rand() * 100);
  const dsLatencyMs = Math.round(10 + rand() * 40);

  // 1. Gateway inbound log
  logs.push({
    timestamp: new Date(timeMs).toISOString(),
    traceId,
    spanId: hexId(rand, 16),
    severity: "INFO",
    severityNum: 9,
    serviceName: gateway.name,
    body: `\u27A1\uFE0F [${endpoint.method}] ${endpoint.gatewayPath} from ${clientIp}`,
    scopeName: "com.geass.gateway.common.LoggingFilter",
    attributes: {},
    resource: makeResource(gateway),
  });
  timeMs += 1;

  // 2. Auth verify log (gateway)
  const userId = Math.floor(rand() * 5) + 1;
  const role = Math.floor(rand() * 3) + 1;
  logs.push({
    timestamp: new Date(timeMs).toISOString(),
    traceId,
    spanId: hexId(rand, 16),
    severity: "INFO",
    severityNum: 9,
    serviceName: gateway.name,
    body: `[Auth] userId=${userId}, role=${role}, uri=${endpoint.gatewayPath}`,
    scopeName: "com.geass.gateway.filter.AuthVerifyFilter",
    attributes: {},
    resource: makeResource(gateway),
  });
  timeMs += 1;

  // 3. RemoteCaller log (gateway → downstream)
  logs.push({
    timestamp: new Date(timeMs).toISOString(),
    traceId,
    spanId: hexId(rand, 16),
    severity: "INFO",
    severityNum: 9,
    serviceName: gateway.name,
    body: `[Downstream] ${endpoint.method} http://${endpoint.downstreamService}:${downstream.port}${endpoint.downstreamPath} -> ${isError ? 500 : 200} (${dsLatencyMs}ms)`,
    scopeName: "com.geass.gateway.util.RemoteCaller",
    attributes: {},
    resource: makeResource(gateway),
  });
  timeMs += 2;

  // 4. Downstream inbound log
  logs.push({
    timestamp: new Date(timeMs).toISOString(),
    traceId,
    spanId: hexId(rand, 16),
    severity: "INFO",
    severityNum: 9,
    serviceName: downstream.name,
    body: `\u27A1\uFE0F [${endpoint.method}] ${endpoint.downstreamPath} from ${gateway.name}`,
    scopeName: downstream.scopes[0].fullName, // LoggingFilter
    attributes: {},
    resource: makeResource(downstream),
  });
  timeMs += 1;

  // 5. If media service, add MyBatis SQL logs (DEBUG)
  if (downstream.name === "geass-media") {
    const sqlCount = 1 + Math.floor(rand() * 3);
    for (let i = 0; i < sqlCount; i++) {
      const sql = SQL_TEMPLATES[Math.floor(rand() * SQL_TEMPLATES.length)];
      const mapperScope = downstream.scopes[1 + Math.floor(rand() * 3)]; // one of the mapper scopes

      // Preparing
      logs.push({
        timestamp: new Date(timeMs).toISOString(),
        traceId,
        spanId: hexId(rand, 16),
        severity: "DEBUG",
        severityNum: 5,
        serviceName: downstream.name,
        body: `==>  Preparing: ${sql}`,
        scopeName: mapperScope.fullName,
        attributes: {},
        resource: makeResource(downstream),
      });
      timeMs += 1;

      // Parameters
      const paramCount = (sql.match(/\?/g) || []).length;
      const params = Array.from({ length: paramCount }, () => {
        const v = Math.floor(rand() * 50) + 1;
        return `${v}(Long)`;
      }).join(", ");
      logs.push({
        timestamp: new Date(timeMs).toISOString(),
        traceId,
        spanId: hexId(rand, 16),
        severity: "DEBUG",
        severityNum: 5,
        serviceName: downstream.name,
        body: `==> Parameters: ${params}`,
        scopeName: mapperScope.fullName,
        attributes: {},
        resource: makeResource(downstream),
      });
      timeMs += 1;

      // Total
      const total = Math.floor(rand() * 30) + 1;
      logs.push({
        timestamp: new Date(timeMs).toISOString(),
        traceId,
        spanId: hexId(rand, 16),
        severity: "DEBUG",
        severityNum: 5,
        serviceName: downstream.name,
        body: `<==      Total: ${total}`,
        scopeName: mapperScope.fullName,
        attributes: {},
        resource: makeResource(downstream),
      });
      timeMs += 1;
    }
  }

  // 6. Business log (occasional, from ServiceImpl scope)
  if (rand() < 0.15) {
    const implScope = downstream.scopes[downstream.scopes.length - 1];
    let businessBody: string;
    if (downstream.name === "geass-user") {
      businessBody = `[Login] Success: userId=${userId}, username=user${userId}`;
    } else if (downstream.name === "geass-favorites") {
      businessBody = `[Favorites] Added: userId=${userId}, animeId=${Math.floor(rand() * 50) + 1}`;
    } else if (downstream.name === "geass-history") {
      businessBody = `[History] Updated: userId=${userId}, animeId=${Math.floor(rand() * 50) + 1}, progress=${Math.floor(rand() * 24) + 1}`;
    } else {
      businessBody = `[Service] Processing request for userId=${userId}`;
    }
    logs.push({
      timestamp: new Date(timeMs).toISOString(),
      traceId,
      spanId: hexId(rand, 16),
      severity: "INFO",
      severityNum: 9,
      serviceName: downstream.name,
      body: businessBody,
      scopeName: implScope.fullName,
      attributes: {},
      resource: makeResource(downstream),
    });
    timeMs += 1;
  }

  // 7. Downstream outbound log
  logs.push({
    timestamp: new Date(timeMs).toISOString(),
    traceId,
    spanId: hexId(rand, 16),
    severity: isError ? "ERROR" : "INFO",
    severityNum: isError ? 17 : 9,
    serviceName: downstream.name,
    body: `\u2B05\uFE0F [${endpoint.method}] ${endpoint.downstreamPath} - ${dsLatencyMs}ms (status=${isError ? 500 : 200})`,
    scopeName: downstream.scopes[0].fullName,
    attributes: isError ? { "error.message": "Internal Server Error" } : {},
    resource: makeResource(downstream),
  });
  timeMs += 1;

  // 8. Gateway outbound log
  logs.push({
    timestamp: new Date(timeMs).toISOString(),
    traceId,
    spanId: hexId(rand, 16),
    severity: isError ? "ERROR" : "INFO",
    severityNum: isError ? 17 : 9,
    serviceName: gateway.name,
    body: `\u2B05\uFE0F [${endpoint.method}] ${endpoint.gatewayPath} - ${latencyMs}ms (status=${isError ? 500 : 200})`,
    scopeName: "com.geass.gateway.common.LoggingFilter",
    attributes: isError ? { "error.message": "Downstream service error" } : {},
    resource: makeResource(gateway),
  });

  return logs;
}

// ============================================================
// Build all mock data (~1600 logs)
// ============================================================

function buildMockLogs(): LogEntry[] {
  const rand = seededRandom("log-mock-v1-stable");
  const now = new Date();
  const allLogs: LogEntry[] = [];

  // Generate ~200 request traces → ~1600 logs (avg 8 logs per request)
  const REQUEST_COUNT = 200;

  for (let i = 0; i < REQUEST_COUNT; i++) {
    const endpoint = pickWeighted(ENDPOINTS, rand);

    // Spread over last 15 days
    const offsetMs = Math.floor(rand() * 15 * 24 * 3600 * 1000);
    const baseTime = new Date(now.getTime() - offsetMs);

    // ~2% WARN, ~1% ERROR
    const errorRoll = rand();
    const isError = errorRoll < 0.01;

    const requestLogs = generateRequestLogs(rand, endpoint, baseTime, isError);

    // Inject occasional WARN logs (~2%)
    if (errorRoll >= 0.01 && errorRoll < 0.03) {
      const warnSvc = SERVICES[Math.floor(rand() * SERVICES.length)];
      requestLogs.push({
        timestamp: new Date(baseTime.getTime() + Math.floor(rand() * 50)).toISOString(),
        traceId: requestLogs[0].traceId,
        spanId: hexId(rand, 16),
        severity: "WARN",
        severityNum: 13,
        serviceName: warnSvc.name,
        body: `[Warn] Slow response detected: ${Math.floor(200 + rand() * 300)}ms exceeds threshold`,
        scopeName: warnSvc.scopes[0].fullName,
        attributes: {},
        resource: makeResource(warnSvc),
      });
    }

    allLogs.push(...requestLogs);
  }

  // Sort by timestamp descending (newest first)
  allLogs.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  return allLogs;
}

export const mockLogEntries: LogEntry[] = buildMockLogs();

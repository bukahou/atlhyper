import { readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import ts from "typescript";
import { describe, expect, it } from "vitest";
import { LEGACY_FETCH_FILES } from "./legacy-fetch-files";

// 页面与组件的读请求一律经 queries/ 下的工厂 + useQuery：
// 同一份数据只请求一次、后发的结果生效、参数变化或卸载时取消。
// 自己在 useEffect 里调接口、自己起定时器轮询，这些保证一条都没有。
//
// 这里用语法树检查两条规则：
//   1. app/ 与 components/ 下不得「值导入」@/api/*（只导入类型不算）
//   2. 不得使用 useAutoRefresh
// 还没迁移的文件列在 legacy-fetch-files.ts 里，迁一个删一个；⛔ 不得往里加新文件。

const SRC_ROOT = path.resolve(__dirname, "..");
const CHECKED_DIRS = ["app", "components"];
const API_MODULE = /^@\/api(\/|$)/;
const POLLING_HOOK = "useAutoRefresh";

function sourceFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const full = path.join(dir, name);
    if (statSync(full).isDirectory()) return sourceFiles(full);
    return /\.(ts|tsx)$/.test(name) && !/\.test\.tsx?$/.test(name) ? [full] : [];
  });
}

/** 返回一段源码里违反规则的写法（空数组 = 合规） */
export function directFetchViolations(fileName: string, text: string): string[] {
  const source = ts.createSourceFile(fileName, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const violations: string[] = [];

  for (const statement of source.statements) {
    if (!ts.isImportDeclaration(statement) || !ts.isStringLiteral(statement.moduleSpecifier)) continue;
    const moduleName = statement.moduleSpecifier.text;
    const clause = statement.importClause;
    if (!clause || clause.isTypeOnly) continue;

    const bindings = clause.namedBindings;
    const namedImports = bindings && ts.isNamedImports(bindings) ? bindings.elements : [];
    const valueNames = namedImports.filter((element) => !element.isTypeOnly).map((element) => element.name.text);
    const importsWholeModule = clause.name !== undefined || (bindings !== undefined && ts.isNamespaceImport(bindings));

    if (API_MODULE.test(moduleName) && (importsWholeModule || valueNames.length > 0)) {
      violations.push(`值导入 ${moduleName}`);
    }
    if (valueNames.includes(POLLING_HOOK) || clause.name?.text === POLLING_HOOK) {
      violations.push(`使用 ${POLLING_HOOK}`);
    }
  }
  return violations;
}

function relativePath(file: string): string {
  return path.relative(SRC_ROOT, file).split(path.sep).join("/");
}

describe("检查器自身", () => {
  it("识别值导入与轮询 hook，放过类型导入", () => {
    expect(directFetchViolations("a.tsx", 'import { getPodList } from "@/api/pod";')).toEqual(["值导入 @/api/pod"]);
    expect(directFetchViolations("a.tsx", 'import * as api from "@/api/pod";')).toEqual(["值导入 @/api/pod"]);
    expect(directFetchViolations("a.tsx", 'import { type Pod, getPodList } from "@/api/pod";')).toEqual(["值导入 @/api/pod"]);
    expect(directFetchViolations("a.tsx", 'import { useAutoRefresh } from "@/hooks/useAutoRefresh";')).toEqual([
      "使用 useAutoRefresh",
    ]);
    expect(directFetchViolations("a.tsx", 'import type { Pod } from "@/api/pod";')).toEqual([]);
    expect(directFetchViolations("a.tsx", 'import { type Pod } from "@/api/pod";')).toEqual([]);
    expect(directFetchViolations("a.tsx", 'import { podListQuery } from "@/queries/cluster";')).toEqual([]);
  });
});

describe("页面与组件不绕过数据层", () => {
  const files = CHECKED_DIRS.flatMap((dir) => sourceFiles(path.join(SRC_ROOT, dir)));
  const legacy = new Set(LEGACY_FETCH_FILES);

  it("扫描到了文件（装置有效）", () => {
    expect(files.length).toBeGreaterThan(100);
  });

  it("已迁移的文件没有直接取数的写法", () => {
    const offenders = files
      .filter((file) => !legacy.has(relativePath(file)))
      .map((file) => ({ file: relativePath(file), violations: directFetchViolations(file, readFileSync(file, "utf8")) }))
      .filter((entry) => entry.violations.length > 0)
      .map((entry) => `${entry.file}: ${entry.violations.join("、")}`);
    expect(offenders).toEqual([]);
  });

  it("「尚未迁移」清单里没有已经迁完或已删除的文件", () => {
    const scanned = new Map(files.map((file) => [relativePath(file), file]));
    const stale = LEGACY_FETCH_FILES.filter((entry) => {
      const file = scanned.get(entry);
      return file === undefined || directFetchViolations(file, readFileSync(file, "utf8")).length === 0;
    });
    expect(stale).toEqual([]);
  });
});

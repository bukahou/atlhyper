import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTypescript from "eslint-config-next/typescript";
import pluginQuery from "@tanstack/eslint-plugin-query";
import { LEGACY_LINT_FILES } from "./eslint.legacy-files.mjs";

// 旧文件里暂时只报警告的规则（都是 react-hooks 的）。
// 迁到数据层时一并修掉，然后把文件从 eslint.legacy-files.mjs 里删除。
const LEGACY_DOWNGRADED_RULES = [
  "react-hooks/exhaustive-deps",
  "react-hooks/set-state-in-effect",
  "react-hooks/set-state-in-render",
  "react-hooks/rules-of-hooks",
  "react-hooks/static-components",
  "react-hooks/refs",
  "react-hooks/purity",
  "react-hooks/preserve-manual-memoization",
];

export default defineConfig([
  ...nextVitals,
  ...nextTypescript,
  ...pluginQuery.configs["flat/recommended"],
  {
    rules: {
      // 依赖数组写错会让闭包一直用旧值（集群 ID、筛选条件），必须由工具拦住
      "react-hooks/exhaustive-deps": "error",
    },
  },
  {
    files: LEGACY_LINT_FILES,
    rules: Object.fromEntries(LEGACY_DOWNGRADED_RULES.map((rule) => [rule, "warn"])),
  },
  globalIgnores([".next/**", "out/**", "build/**", "next-env.d.ts", "node_modules/**"]),
]);

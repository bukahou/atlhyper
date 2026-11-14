import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTypescript from "eslint-config-next/typescript";
import pluginQuery from "@tanstack/eslint-plugin-query";

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
  globalIgnores([".next/**", "out/**", "build/**", "next-env.d.ts", "node_modules/**"]),
]);

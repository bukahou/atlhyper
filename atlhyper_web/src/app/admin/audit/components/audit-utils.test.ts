import { describe, expect, it } from "vitest";
import { zh } from "@/i18n/locales/zh";
import { ja } from "@/i18n/locales/ja";
import { getActionLabel } from "./audit-utils";

// MCP 入口的每次工具调用记为 action=execute、resource=mcp_tool
describe("getActionLabel: MCP 工具调用", () => {
  it("中文", () => {
    expect(getActionLabel("execute", "mcp_tool", zh.audit)).toBe("执行 MCP 工具");
  });
  it("日文", () => {
    expect(getActionLabel("execute", "mcp_tool", ja.audit)).toBe("実行 MCP ツール");
  });
});

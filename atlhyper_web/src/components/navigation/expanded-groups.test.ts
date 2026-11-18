import { describe, expect, it } from "vitest";
import { withGroupsExpanded } from "./expanded-groups";

describe("withGroupsExpanded", () => {
  it("把还没展开的分组加进去，已展开的其余分组不动", () => {
    expect(withGroupsExpanded(["observe"], ["cluster"])).toEqual(["observe", "cluster"]);
    expect(withGroupsExpanded([], ["cluster", "admin"])).toEqual(["cluster", "admin"]);
  });

  it("已经展开的分组不重复", () => {
    expect(withGroupsExpanded(["observe", "cluster"], ["cluster", "admin"])).toEqual(["observe", "cluster", "admin"]);
  });

  it("没有新增时返回原数组，不产生新的状态", () => {
    const expanded = ["observe", "cluster"];
    expect(withGroupsExpanded(expanded, ["cluster"])).toBe(expanded);
    expect(withGroupsExpanded(expanded, [])).toBe(expanded);
  });
});

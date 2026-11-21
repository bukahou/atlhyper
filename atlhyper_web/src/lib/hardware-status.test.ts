import { describe, expect, it } from "vitest";
import { hardwareStatusChip, hardwareStatusText } from "./hardware-status";

describe("hardware status colors", () => {
  it("unknown 是灰色，不落成正常的绿色", () => {
    expect(hardwareStatusText("unknown")).toBe("text-muted");
    expect(hardwareStatusChip("unknown")).not.toContain("green");
  });
  it("crit / warn / good 各自的颜色", () => {
    expect(hardwareStatusText("crit")).toContain("red");
    expect(hardwareStatusText("warn")).toContain("yellow");
    expect(hardwareStatusText("good")).toContain("green");
  });
});

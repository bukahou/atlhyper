import { describe, expect, it } from "vitest";
import {
  newSilenceDraft,
  parseLocalInputValue,
  SILENCE_COMMENT_MAX_CHARS,
  silenceBodyFromDraft,
  silenceDraftProblems,
  toLocalInputValue,
  type NotifySilenceDraft,
} from "./notify-silence-input";

// 用本地时间构造，测试不依赖运行机器的时区
const NOW = new Date(2026, 10, 15, 20, 0, 0);

const validDraft: NotifySilenceDraft = {
  clusterId: "prod",
  namespace: "shop",
  subject: "",
  startsAtLocal: "2026-11-15T20:00",
  endsAtLocal: "2026-11-15T22:00",
  comment: "数据库升级",
};

describe("toLocalInputValue / parseLocalInputValue", () => {
  it("写成 datetime-local 认的形式：本地时间、精确到分钟、各段补零", () => {
    expect(toLocalInputValue(new Date(2026, 0, 5, 3, 7, 59))).toBe("2026-01-05T03:07");
  });

  it("写出去再读回来是同一分钟", () => {
    expect(parseLocalInputValue(toLocalInputValue(NOW))?.getTime()).toBe(NOW.getTime());
  });

  it.each(["", "2026-11-15", "20:00", "not a date", "2026-13-40T99:99"])("%j 读不出时刻", (localValue) => {
    expect(parseLocalInputValue(localValue)).toBeNull();
  });
});

describe("newSilenceDraft", () => {
  it("从现在开始、两小时后结束，范围不限，备注待填", () => {
    expect(newSilenceDraft(NOW)).toEqual({
      clusterId: "",
      namespace: "",
      subject: "",
      startsAtLocal: "2026-11-15T20:00",
      endsAtLocal: "2026-11-15T22:00",
      comment: "",
    });
  });
});

describe("silenceDraftProblems", () => {
  it("合规的表单没有问题", () => {
    expect(silenceDraftProblems(validDraft, NOW)).toEqual([]);
  });

  it("三个范围条件都留空也合规（= 全部告警）", () => {
    expect(silenceDraftProblems({ ...validDraft, clusterId: "", namespace: "", subject: "" }, NOW)).toEqual([]);
  });

  it("起止时间没填", () => {
    expect(silenceDraftProblems({ ...validDraft, startsAtLocal: "", endsAtLocal: "" }, NOW)).toEqual([
      "startInvalid",
      "endInvalid",
    ]);
  });

  it("结束必须晚于开始：相等也不行", () => {
    expect(silenceDraftProblems({ ...validDraft, endsAtLocal: "2026-11-15T20:00" }, NOW)).toContain(
      "endNotAfterStart",
    );
  });

  it("结束必须晚于现在：补录一段已经过去的静默不允许", () => {
    const pastDraft = { ...validDraft, startsAtLocal: "2026-11-15T10:00", endsAtLocal: "2026-11-15T12:00" };
    expect(silenceDraftProblems(pastDraft, NOW)).toEqual(["endInPast"]);
  });

  it("时长不超过 30 天：正好 30 天可以，多一分钟不行", () => {
    const thirtyDays = { ...validDraft, endsAtLocal: "2026-12-15T20:00" };
    expect(silenceDraftProblems(thirtyDays, NOW)).toEqual([]);
    expect(silenceDraftProblems({ ...thirtyDays, endsAtLocal: "2026-12-15T20:01" }, NOW)).toEqual(["tooLong"]);
  });

  it("备注必填；只有空白也算没填", () => {
    expect(silenceDraftProblems({ ...validDraft, comment: "  " }, NOW)).toEqual(["commentRequired"]);
  });

  it("备注不超过 256 字，按字符数而不是字节数计", () => {
    const atLimit = { ...validDraft, comment: "维".repeat(SILENCE_COMMENT_MAX_CHARS) };
    expect(silenceDraftProblems(atLimit, NOW)).toEqual([]);
    expect(silenceDraftProblems({ ...atLimit, comment: atLimit.comment + "护" }, NOW)).toEqual(["commentTooLong"]);
  });
});

describe("silenceBodyFromDraft", () => {
  it("起止时间转成 RFC 3339（UTC），文本去掉首尾空白", () => {
    const body = silenceBodyFromDraft({ ...validDraft, namespace: " shop ", comment: " 数据库升级 " });
    expect(body).toEqual({
      clusterId: "prod",
      namespace: "shop",
      subject: "",
      startsAt: new Date(2026, 10, 15, 20, 0).toISOString(),
      endsAt: new Date(2026, 10, 15, 22, 0).toISOString(),
      comment: "数据库升级",
    });
  });

  it("起止时间读不出来时不产生请求体", () => {
    expect(silenceBodyFromDraft({ ...validDraft, endsAtLocal: "" })).toBeNull();
  });
});

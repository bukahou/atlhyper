"use client";

import type { IntegerRange } from "@/lib/notify-policy-input";
import { FIELD_ERROR_CLASS, HINT_CLASS, INPUT_CLASS, LABEL_CLASS } from "../shared/form-styles";

interface PolicyNumberFieldProps {
  id: string;
  label: string;
  hint: string;
  /** 输入框里的文本（还没读成数字） */
  value: string;
  onChange: (value: string) => void;
  range: IntegerRange;
  /** 单位，显示在输入框后面；没有单位传空串 */
  unit: string;
  /** 这一项不合规时显示的提示；合规传空串 */
  error: string;
  disabled: boolean;
}

/** 策略里的一个整数项：输入框 + 单位 + 范围 + 说明 */
export function PolicyNumberField({
  id,
  label,
  hint,
  value,
  onChange,
  range,
  unit,
  error,
  disabled,
}: PolicyNumberFieldProps) {
  return (
    <div>
      <label htmlFor={id} className={LABEL_CLASS}>
        {label}
      </label>
      <div className="flex items-center gap-2">
        <input
          id={id}
          type="text"
          inputMode="numeric"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          disabled={disabled}
          autoComplete="off"
          aria-invalid={error !== ""}
          className={`${INPUT_CLASS} max-w-[10rem] tabular-nums`}
        />
        <span className="text-sm text-muted whitespace-nowrap">
          {unit}（{range.min}–{range.max}）
        </span>
      </div>
      {error !== "" ? <p className={FIELD_ERROR_CLASS}>{error}</p> : <p className={HINT_CLASS}>{hint}</p>}
    </div>
  );
}

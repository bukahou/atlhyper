"use client";

import { Switch } from "../shared/Switch";

interface PolicyToggleRowProps {
  label: string;
  hint: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled: boolean;
}

/** 策略里的一个开关项：左边名称与说明，右边开关 */
export function PolicyToggleRow({ label, hint, checked, onChange, disabled }: PolicyToggleRowProps) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <div className="text-sm font-medium text-default">{label}</div>
        <p className="text-xs text-muted mt-0.5">{hint}</p>
      </div>
      <Switch checked={checked} onChange={onChange} disabled={disabled} label={label} />
    </div>
  );
}

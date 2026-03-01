// SMTP 预设服务器配置

export interface SmtpPreset {
  value: string;
  label: string;
  port: number;
}

interface SmtpPresetLabels {
  qq: string;
  netease163: string;
  netease126: string;
  tencentEnterprise: string;
  aliEnterprise: string;
  feishu: string;
}

export function getSmtpPresets(presetLabels: SmtpPresetLabels): SmtpPreset[] {
  return [
    { value: "smtp.gmail.com", label: "Gmail", port: 587 },
    { value: "smtp.office365.com", label: "Outlook / Office 365", port: 587 },
    { value: "smtp.qq.com", label: presetLabels.qq, port: 587 },
    { value: "smtp.163.com", label: presetLabels.netease163, port: 465 },
    { value: "smtp.126.com", label: presetLabels.netease126, port: 465 },
    { value: "smtp.exmail.qq.com", label: presetLabels.tencentEnterprise, port: 465 },
    { value: "smtp.mxhichina.com", label: presetLabels.aliEnterprise, port: 465 },
    { value: "smtp.feishu.cn", label: presetLabels.feishu, port: 465 },
  ];
}

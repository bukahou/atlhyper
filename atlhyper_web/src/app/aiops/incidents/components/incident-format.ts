/** 事件页各组件共用的显示格式 */

/** 事件状态徽标的配色 */
export const INCIDENT_STATE_COLORS: Record<string, string> = {
  warning: "bg-yellow-500/15 text-yellow-600 dark:text-yellow-400",
  incident: "bg-red-500/15 text-red-600 dark:text-red-400",
  recovery: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  stable: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
};

/** 时长（秒）→「N 分钟」或「Xh Ym」。minuteLabel 是「分钟」的译名 */
export function formatDuration(seconds: number, minuteLabel: string): string {
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes} ${minuteLabel}`;
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}

function twoDigits(part: number): string {
  return part.toString().padStart(2, "0");
}

/** 列表里的紧凑时间：月/日 时:分（本地时间） */
export function formatShortTime(isoTimestamp: string): string {
  const moment = new Date(isoTimestamp);
  return `${moment.getMonth() + 1}/${moment.getDate()} ${twoDigits(moment.getHours())}:${twoDigits(moment.getMinutes())}`;
}

/** 通知设置页各表单共用的样式 */

export const INPUT_CLASS =
  "w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default text-sm focus:ring-2 focus:ring-primary outline-none disabled:opacity-60 disabled:cursor-not-allowed";

/** 下拉框：option 在深色主题下要单独给底色，否则是白底白字 */
export const SELECT_CLASS = `${INPUT_CLASS} [&>option]:bg-white [&>option]:text-gray-900 dark:[&>option]:bg-gray-800 dark:[&>option]:text-gray-100`;

export const LABEL_CLASS = "block text-sm font-medium text-muted mb-1";

export const HINT_CLASS = "text-xs text-muted mt-1";

export const FIELD_ERROR_CLASS = "text-xs text-red-500 mt-1";

export const PRIMARY_BUTTON_CLASS =
  "flex items-center gap-1.5 px-3 sm:px-4 py-2 bg-primary hover:bg-primary-hover text-white rounded-lg transition-colors text-sm disabled:opacity-40 disabled:cursor-not-allowed";

export const SECONDARY_BUTTON_CLASS =
  "px-4 py-2 border border-[var(--border-color)] rounded-lg hover-bg text-sm text-default disabled:opacity-40 disabled:cursor-not-allowed";

/** 表格行里的图标按钮 */
export const ICON_BUTTON_CLASS =
  "p-2 rounded-lg hover-bg text-muted hover:text-primary transition-colors disabled:opacity-40 disabled:cursor-not-allowed disabled:hover:text-muted";

export const CARD_CLASS = "bg-card rounded-xl border border-[var(--border-color)] overflow-hidden";

/** 徽标所在的单元格：列多、屏幕窄时不让徽标里的字被挤成一字一行 */
export const NOWRAP_CLASS = "whitespace-nowrap";

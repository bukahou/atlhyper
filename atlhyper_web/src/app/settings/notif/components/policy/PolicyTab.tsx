"use client";

import { useQuery } from "@tanstack/react-query";
import { QueryView } from "@/components/common";
import { useI18n } from "@/i18n/context";
import { MOCK_NOTIFY_POLICY, MOCK_NOTIFY_RECEIVERS } from "@/mock/notify";
import { notifyPolicyQuery } from "@/queries/notify";
import { notifyReceiversQuery } from "@/queries/notify-routing";
import type { NotifyPolicy, NotifyReceiver } from "@/types/notify";
import type { NotifyTabProps } from "../shared/format";
import { PolicyForm } from "./PolicyForm";

const NO_RECEIVERS: NotifyReceiver[] = [];

/** 「通知策略」页签：全局的通知开关、阈值、分组与重复的时间参数、默认接收人 */
export function PolicyTab({ isDemo }: NotifyTabProps) {
  const { t } = useI18n();
  const policyQuery = useQuery(notifyPolicyQuery(!isDemo));
  // 接收人只用来给「默认接收人」提供选项；取不到时策略的其余部分照常显示
  const receiversQuery = useQuery(notifyReceiversQuery(!isDemo));
  const receivers = isDemo ? MOCK_NOTIFY_RECEIVERS : (receiversQuery.data ?? NO_RECEIVERS);

  const renderPolicy = (policy: NotifyPolicy) => (
    <PolicyForm
      // 已保存的策略变了（保存成功、别处修改）就按新值重新填表单
      key={JSON.stringify(policy)}
      policy={policy}
      receivers={receivers}
      readOnly={isDemo}
      t={t.notifications}
      saveLabel={t.common.save}
    />
  );

  return isDemo ? renderPolicy(MOCK_NOTIFY_POLICY) : <QueryView query={policyQuery}>{renderPolicy}</QueryView>;
}

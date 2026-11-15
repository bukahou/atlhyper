"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BellRing } from "lucide-react";
import { QueryView } from "@/components/common";
import { useScopedState } from "@/hooks/useScopedState";
import { useI18n } from "@/i18n/context";
import { MOCK_NOTIFY_ALERTS } from "@/mock/notify";
import { notifyAlertsQuery } from "@/queries/notify";
import type { NotifyAlert, NotifyAlertStatus, NotifyPage } from "@/types/notify";
import { EmptyCard } from "../shared/EmptyCard";
import { demoPage, type NotifyTabProps } from "../shared/format";
import { CARD_CLASS } from "../shared/form-styles";
import { SegmentedFilter } from "../shared/SegmentedFilter";
import { ServerPager } from "../shared/ServerPager";
import { AlertsTable } from "./AlertsTable";

const PAGE_SIZE = 20;

/** 「当前告警」页签：按状态（持续中 / 已结束）看告警，以及它们有没有被抑制、被静默 */
export function AlertsTab({ isDemo }: NotifyTabProps) {
  const { t } = useI18n();
  const notifyT = t.notifications;
  const [status, setStatus] = useState<NotifyAlertStatus>("firing");
  // 页码依附于状态筛选：换状态回到第一页
  const [page, setPage] = useScopedState(status, 0);

  const alertsQuery = useQuery(notifyAlertsQuery({ status, limit: PAGE_SIZE, offset: page * PAGE_SIZE }, !isDemo));

  const renderAlertPage = (alertPage: NotifyPage<NotifyAlert>) =>
    alertPage.items.length === 0 ? (
      <EmptyCard
        icon={BellRing}
        title={status === "firing" ? notifyT.alerts.emptyFiring : notifyT.alerts.emptyResolved}
      />
    ) : (
      <div className={CARD_CLASS}>
        <AlertsTable alerts={alertPage.items} t={notifyT} />
        <ServerPager page={page} pageSize={PAGE_SIZE} total={alertPage.total} onPageChange={setPage} />
      </div>
    );

  return (
    <div className="space-y-4">
      <SegmentedFilter
        options={[
          { value: "firing", label: notifyT.alerts.statusFiring },
          { value: "resolved", label: notifyT.alerts.statusResolved },
        ]}
        value={status}
        onChange={setStatus}
      />

      {isDemo ? (
        renderAlertPage(
          demoPage(
            MOCK_NOTIFY_ALERTS.filter((alert) => alert.status === status),
            page,
            PAGE_SIZE,
          ),
        )
      ) : (
        <QueryView query={alertsQuery}>{renderAlertPage}</QueryView>
      )}
    </div>
  );
}

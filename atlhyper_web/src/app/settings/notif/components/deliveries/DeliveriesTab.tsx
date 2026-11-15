"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Send } from "lucide-react";
import { QueryView } from "@/components/common";
import { useScopedState } from "@/hooks/useScopedState";
import { useI18n } from "@/i18n/context";
import { MOCK_NOTIFY_DELIVERIES } from "@/mock/notify";
import { notifyDeliveriesQuery } from "@/queries/notify";
import { NOTIFY_DELIVERY_STATUSES, type NotifyDelivery, type NotifyDeliveryStatus, type NotifyPage } from "@/types/notify";
import { EmptyCard } from "../shared/EmptyCard";
import { demoPage, type NotifyTabProps } from "../shared/format";
import { CARD_CLASS } from "../shared/form-styles";
import { SegmentedFilter } from "../shared/SegmentedFilter";
import { ServerPager } from "../shared/ServerPager";
import { DeliveriesTable } from "./DeliveriesTable";

const PAGE_SIZE = 20;

/** 「送达记录」页签：每条通知发给了谁、成没成功。空串 = 不按状态筛选 */
export function DeliveriesTab({ isDemo }: NotifyTabProps) {
  const { t } = useI18n();
  const notifyT = t.notifications;
  const [status, setStatus] = useState<NotifyDeliveryStatus | "">("");
  // 页码依附于状态筛选：换状态回到第一页
  const [page, setPage] = useScopedState(status, 0);

  const deliveriesQuery = useQuery(
    notifyDeliveriesQuery({ status, limit: PAGE_SIZE, offset: page * PAGE_SIZE }, !isDemo),
  );

  const renderDeliveryPage = (deliveryPage: NotifyPage<NotifyDelivery>) =>
    deliveryPage.items.length === 0 ? (
      <EmptyCard icon={Send} title={notifyT.deliveries.empty} />
    ) : (
      <div className={CARD_CLASS}>
        <DeliveriesTable deliveries={deliveryPage.items} t={notifyT} />
        <ServerPager page={page} pageSize={PAGE_SIZE} total={deliveryPage.total} onPageChange={setPage} />
      </div>
    );

  return (
    <div className="space-y-4">
      <SegmentedFilter
        options={[
          { value: "", label: notifyT.deliveries.allStatuses },
          ...NOTIFY_DELIVERY_STATUSES.map((value) => ({ value, label: notifyT.deliveries.statusLabel[value] })),
        ]}
        value={status}
        onChange={setStatus}
      />

      {isDemo ? (
        renderDeliveryPage(
          demoPage(
            MOCK_NOTIFY_DELIVERIES.filter((delivery) => status === "" || delivery.status === status),
            page,
            PAGE_SIZE,
          ),
        )
      ) : (
        <QueryView query={deliveriesQuery}>{renderDeliveryPage}</QueryView>
      )}
    </div>
  );
}

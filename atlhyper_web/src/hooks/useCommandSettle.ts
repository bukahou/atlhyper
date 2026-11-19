/**
 * 写操作下发后，等指令到终态再让相关查询失效重取（见 lib/command-settle）。
 * 没有成功完成时弹出提示（失败原因 / 仍在执行），结果也返回给调用方。
 * 组件卸载时停止等待：页面都离开了，没有谁还需要那次刷新。
 */
import { useCallback, useEffect, useState } from "react";
import { useQueryClient, type QueryKey } from "@tanstack/react-query";
import { getCommandStatus } from "@/api/commands";
import { toast } from "@/components/common/Toast";
import { useI18n } from "@/i18n/context";
import { commandOutcomeMessage, waitForCommand, type CommandOutcome } from "@/lib/command-settle";

export function useCommandSettle() {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [controllers] = useState(() => new Set<AbortController>());

  useEffect(() => () => {
    for (const controller of controllers) controller.abort();
  }, [controllers]);

  return useCallback(
    async (commandId: string, queryKeys: readonly QueryKey[]): Promise<CommandOutcome | null> => {
      const controller = new AbortController();
      controllers.add(controller);
      try {
        const outcome = await waitForCommand(
          commandId,
          async (id, signal) => (await getCommandStatus(id, signal)).data,
          { signal: controller.signal },
        );
        for (const queryKey of queryKeys) void queryClient.invalidateQueries({ queryKey });
        const message = commandOutcomeMessage(outcome, t.commands);
        if (message) {
          console.error("指令没有成功完成", commandId, outcome);
          toast.error(message);
        }
        return outcome;
      } catch (err) {
        if (controller.signal.aborted) return null; // 组件已卸载
        throw err;
      } finally {
        controllers.delete(controller);
      }
    },
    [queryClient, controllers, t],
  );
}

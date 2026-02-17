"use client";

import { createContext, useContext } from "react";
import type { EntityDetailTarget } from "@/types/entity-detail";

interface EntityDetailContextValue {
  openEntityDetail: (target: EntityDetailTarget) => void;
  closeEntityDetail: () => void;
}

export const EntityDetailContext = createContext<
  EntityDetailContextValue | undefined
>(undefined);

export function useEntityDetail() {
  const ctx = useContext(EntityDetailContext);
  if (!ctx) {
    throw new Error("useEntityDetail must be used within EntityDetailProvider");
  }
  return ctx;
}

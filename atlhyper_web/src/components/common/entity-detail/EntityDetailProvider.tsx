"use client";

import { useState, useCallback } from "react";
import type { EntityDetailTarget } from "@/types/entity-detail";
import { EntityDetailContext } from "./context";
import { EntityDetailRenderer } from "./EntityDetailRenderer";

export function EntityDetailProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [target, setTarget] = useState<EntityDetailTarget | null>(null);

  const openEntityDetail = useCallback((t: EntityDetailTarget) => {
    setTarget(t);
  }, []);

  const closeEntityDetail = useCallback(() => {
    setTarget(null);
  }, []);

  return (
    <EntityDetailContext.Provider
      value={{ openEntityDetail, closeEntityDetail }}
    >
      {children}
      <EntityDetailRenderer target={target} onClose={closeEntityDetail} />
    </EntityDetailContext.Provider>
  );
}

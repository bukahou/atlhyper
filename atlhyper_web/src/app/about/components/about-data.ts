import type { LucideIcon } from "lucide-react";
import type { AboutTranslations } from "@/types/i18n";
import {
  Globe,
  Network,
  Search,
  FileText,
  Cpu,
  Box,
  Activity,
  Bot,
  ClipboardList,
  AlertTriangle,
  Server,
  Zap,
  Monitor,
} from "lucide-react";

// -- Layer definitions --

export interface LayerDef {
  level: string;
  titleKey: keyof AboutTranslations;
  descKey: keyof AboutTranslations;
  sourceKey: keyof AboutTranslations;
  metricsKey: keyof AboutTranslations;
  icon: LucideIcon;
  status: "done" | "planned";
  color: string;
  detailWhatKey: keyof AboutTranslations;
  detailRoleKey: keyof AboutTranslations;
  detailToolsKey: keyof AboutTranslations;
  detailAtlhyperKey: keyof AboutTranslations;
}

export const layers: LayerDef[] = [
  { level: "L1", titleKey: "layer1Title", descKey: "layer1Desc", sourceKey: "layer1Source", metricsKey: "layer1Metrics", icon: Globe, status: "done", color: "blue", detailWhatKey: "layer1DetailWhat", detailRoleKey: "layer1DetailRole", detailToolsKey: "layer1DetailTools", detailAtlhyperKey: "layer1DetailAtlhyper" },
  { level: "L2", titleKey: "layer2Title", descKey: "layer2Desc", sourceKey: "layer2Source", metricsKey: "layer2Metrics", icon: Network, status: "done", color: "violet", detailWhatKey: "layer2DetailWhat", detailRoleKey: "layer2DetailRole", detailToolsKey: "layer2DetailTools", detailAtlhyperKey: "layer2DetailAtlhyper" },
  { level: "L3", titleKey: "layer3Title", descKey: "layer3Desc", sourceKey: "layer3Source", metricsKey: "layer3Metrics", icon: Search, status: "planned", color: "amber", detailWhatKey: "layer3DetailWhat", detailRoleKey: "layer3DetailRole", detailToolsKey: "layer3DetailTools", detailAtlhyperKey: "layer3DetailAtlhyper" },
  { level: "L4", titleKey: "layer4Title", descKey: "layer4Desc", sourceKey: "layer4Source", metricsKey: "layer4Metrics", icon: FileText, status: "planned", color: "emerald", detailWhatKey: "layer4DetailWhat", detailRoleKey: "layer4DetailRole", detailToolsKey: "layer4DetailTools", detailAtlhyperKey: "layer4DetailAtlhyper" },
  { level: "L5", titleKey: "layer5Title", descKey: "layer5Desc", sourceKey: "layer5Source", metricsKey: "layer5Metrics", icon: Cpu, status: "done", color: "rose", detailWhatKey: "layer5DetailWhat", detailRoleKey: "layer5DetailRole", detailToolsKey: "layer5DetailTools", detailAtlhyperKey: "layer5DetailAtlhyper" },
];

export const drilldowns: { key: keyof AboutTranslations }[] = [
  { key: "drilldown12" },
  { key: "drilldown23" },
  { key: "drilldown34" },
  { key: "drilldown45" },
];

// -- Feature definitions --

export interface FeatureDef {
  icon: LucideIcon;
  titleKey: keyof AboutTranslations;
  descKey: keyof AboutTranslations;
  status: "done" | "planned";
}

export const featureModules: FeatureDef[] = [
  { icon: Box, titleKey: "featureClusterTitle", descKey: "featureClusterDesc", status: "done" },
  { icon: Activity, titleKey: "featureSloTitle", descKey: "featureSloDesc", status: "done" },
  { icon: Network, titleKey: "featureTopologyTitle", descKey: "featureTopologyDesc", status: "done" },
  { icon: Bot, titleKey: "featureAiTitle", descKey: "featureAiDesc", status: "done" },
  { icon: ClipboardList, titleKey: "featureCommandTitle", descKey: "featureCommandDesc", status: "done" },
  { icon: AlertTriangle, titleKey: "featureAlertTitle", descKey: "featureAlertDesc", status: "done" },
  { icon: Cpu, titleKey: "featureMetricsTitle", descKey: "featureMetricsDesc", status: "done" },
  { icon: Search, titleKey: "featureApmTitle", descKey: "featureApmDesc", status: "planned" },
  { icon: FileText, titleKey: "featureLogsTitle", descKey: "featureLogsDesc", status: "planned" },
];

// -- Tech stack definitions --

export interface TechDef {
  icon: LucideIcon;
  titleKey: keyof AboutTranslations;
  stackKey: keyof AboutTranslations;
  descKey: keyof AboutTranslations;
}

export const techStack: TechDef[] = [
  { icon: Server, titleKey: "techMasterTitle", stackKey: "techMasterStack", descKey: "techMasterDesc" },
  { icon: Zap, titleKey: "techAgentTitle", stackKey: "techAgentStack", descKey: "techAgentDesc" },
  { icon: Cpu, titleKey: "techMetricsTitle", stackKey: "techMetricsStack", descKey: "techMetricsDesc" },
  { icon: Monitor, titleKey: "techWebTitle", stackKey: "techWebStack", descKey: "techWebDesc" },
];

// -- Color map --

export const colorMap: Record<string, { bg: string; text: string; border: string; badge: string }> = {
  blue:    { bg: "bg-blue-500/10",    text: "text-blue-500",    border: "border-blue-500/20",    badge: "bg-blue-500/20 text-blue-400" },
  violet:  { bg: "bg-violet-500/10",  text: "text-violet-500",  border: "border-violet-500/20",  badge: "bg-violet-500/20 text-violet-400" },
  amber:   { bg: "bg-amber-500/10",   text: "text-amber-500",   border: "border-amber-500/20",   badge: "bg-amber-500/20 text-amber-400" },
  emerald: { bg: "bg-emerald-500/10", text: "text-emerald-500", border: "border-emerald-500/20", badge: "bg-emerald-500/20 text-emerald-400" },
  rose:    { bg: "bg-rose-500/10",    text: "text-rose-500",    border: "border-rose-500/20",    badge: "bg-rose-500/20 text-rose-400" },
};

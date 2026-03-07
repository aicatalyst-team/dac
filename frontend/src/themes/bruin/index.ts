import type { DashboardTemplate } from "../../types/template";
import { bruinLightTokens } from "./tokens";
import { BruinDashboardLayout } from "./DashboardLayout";
import { BruinDashboardListLayout } from "./DashboardListLayout";
import { BruinWidgetFrame } from "./WidgetFrame";
import { BruinFilterBar } from "./FilterBar";
import { BruinRow, BruinWidgetContainer } from "./Row";
import { MetricWidget } from "../../components/widgets/MetricWidget";
import { ChartWidget } from "../../components/widgets/ChartWidget";
import { TableWidget } from "../../components/widgets/TableWidget";
import { TextWidget } from "../../components/widgets/TextWidget";

export const bruinTemplate: DashboardTemplate = {
  name: "bruin",
  tokens: bruinLightTokens,
  components: {
    DashboardLayout: BruinDashboardLayout,
    DashboardListLayout: BruinDashboardListLayout,
    WidgetFrame: BruinWidgetFrame,
    FilterBar: BruinFilterBar,
    Row: BruinRow,
    WidgetContainer: BruinWidgetContainer,
    MetricWidget,
    ChartWidget,
    TableWidget,
    TextWidget,
  },
};

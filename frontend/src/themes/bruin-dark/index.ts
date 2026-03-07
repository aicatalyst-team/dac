import type { DashboardTemplate } from "../../types/template";
import { bruinDarkTokens } from "../bruin/tokens";
import { BruinDashboardLayout } from "../bruin/DashboardLayout";
import { BruinDashboardListLayout } from "../bruin/DashboardListLayout";
import { BruinWidgetFrame } from "../bruin/WidgetFrame";
import { BruinFilterBar } from "../bruin/FilterBar";
import { BruinRow, BruinWidgetContainer } from "../bruin/Row";
import { MetricWidget } from "../../components/widgets/MetricWidget";
import { ChartWidget } from "../../components/widgets/ChartWidget";
import { TableWidget } from "../../components/widgets/TableWidget";
import { TextWidget } from "../../components/widgets/TextWidget";

export const bruinDarkTemplate: DashboardTemplate = {
  name: "bruin-dark",
  tokens: bruinDarkTokens,
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

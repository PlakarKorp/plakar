import type { StoryObj } from "@storybook/react-vite";
import prettyBytes from "pretty-bytes";

import { BytesLineChart } from "./bytes-line-chart";
import { JobsBarChart } from "./jobs-bar-chart";
import { MultiAxisLineChart } from "./multi-axis-line-chart";
import {
  evenTicks,
  SERIES_ACCENT,
  SERIES_NEUTRAL,
} from "./multi-axis-line-chart.utils";
import { PieChart as PieChartComponent } from "./pie-chart";
import { RatioBar } from "./ratio-bar";
import { SparkBars } from "./spark-bars";
import { UsageBar } from "./usage-bar";

const meta = {
  title: "design-system/Charts",
};

export default meta;

const data = [
  {
    name: "16/01",
    processed: 27800000000,
    stored: 7500000000,
  },
  {
    name: "17/01",
    processed: 18900000000,
    stored: 6500000000,
  },
  {
    name: "18/01",
    processed: 17500000000,
    stored: 7000000000,
  },
  {
    name: "19/01",
    processed: 20000000000,
    stored: 5500000000,
  },
  {
    name: "20/01",
    processed: 16000000000,
    stored: 6000000000,
  },
  {
    name: "21/01",
    processed: 21000000000,
    stored: 4500000000,
  },
  {
    name: "22/01",
    processed: 19200000000,
    stored: 5000000000,
  },
  {
    name: "23/01",
    processed: 22500000000,
    stored: 4300000000,
  },
  {
    name: "24/01",
    processed: 20800000000,
    stored: 4800000000,
  },
  {
    name: "25/01",
    processed: 24000000000,
    stored: 3900000000,
  },
];

export const BytesChart: StoryObj<typeof BytesLineChart> = {
  args: {
    data: data,
  },
  render: (args) => (
    <div className="h-100 w-180">
      <BytesLineChart {...args} />
    </div>
  ),
};

export const Ratio: StoryObj<typeof RatioBar> = {
  args: {
    label: "Backup",
    completed: 80,
    failed: 20,
  },
  render: (args) => (
    <div className="w-120">
      <RatioBar {...args} />
    </div>
  ),
};

export const Usage: StoryObj<typeof UsageBar> = {
  args: {
    used: 1.4e12,
    limit: 2e12,
  },
  render: (args) => (
    <div className="flex w-120 flex-col gap-4">
      <UsageBar {...args} />
      <UsageBar used={3e12} limit={2e12} />
    </div>
  ),
};

const jobsData = [
  {
    name: "19/01",
    completed: 12,
    failed: 1,
  },
  {
    name: "20/01",
    completed: 16,
    failed: 6,
  },
  {
    name: "21/01",
    completed: 21,
    failed: 4,
  },
  {
    name: "22/01",
    completed: 19,
    failed: 5,
  },
  {
    name: "23/01",
    completed: 22,
    failed: 0,
  },
  {
    name: "24/01",
    completed: 20,
    failed: 0,
  },
  {
    name: "25/01",
    completed: 24,
    failed: 3,
  },
];

export const JobsChart: StoryObj<typeof JobsBarChart> = {
  args: {
    data: jobsData,
  },
  render: (args) => (
    <div className="h-80 w-120">
      <JobsBarChart {...args} />
    </div>
  ),
};

const pieData = [
  {
    name: "protected",
    value: 23,
    color: "var(--border-color-success)",
  },
  {
    name: "unprotected",
    value: 5,
    color: "var(--border-color-warning)",
  },
  {
    name: "excluded",
    value: 7,
    color: "var(--border-color-tertiary)",
  },
];

export const PieChart: StoryObj<typeof PieChartComponent> = {
  args: {
    data: pieData,
    label: "35",
  },
  render: (args) => (
    <div className="h-40">
      <PieChartComponent {...args} />
    </div>
  ),
};

const resourceData = [
  { t: 0, cpu: 12, memory: 52_428_800 },
  { t: 5, cpu: 78, memory: 89_128_960 },
  { t: 10, cpu: 91, memory: 134_217_728 },
  { t: 20, cpu: 85, memory: 178_257_920 },
  { t: 35, cpu: 67, memory: 210_763_776 },
  { t: 55, cpu: 72, memory: 230_686_720 },
  { t: 80, cpu: 60, memory: 241_172_480 },
  { t: 110, cpu: 55, memory: 248_512_512 },
  { t: 145, cpu: 48, memory: 251_658_240 },
  { t: 185, cpu: 41, memory: 252_706_816 },
  { t: 230, cpu: 35, memory: 253_755_392 },
  { t: 280, cpu: 29, memory: 254_279_680 },
  { t: 335, cpu: 22, memory: 254_541_824 },
  { t: 395, cpu: 18, memory: 254_803_968 },
  { t: 460, cpu: 14, memory: 255_066_112 },
];

export const MultiAxisChart: StoryObj<typeof MultiAxisLineChart> = {
  args: {
    label: "CPU and memory usage over time",
    times: resourceData.map((d) => d.t),
    left: {
      min: 0,
      max: 100,
      ticks: evenTicks(0, 25, 4),
      format: (v) => `${v}%`,
    },
    right: {
      min: 0,
      max: 320_000_000,
      ticks: evenTicks(0, 80_000_000, 4),
      format: (v) => (v === 0 ? "0" : prettyBytes(v)),
    },
    lines: [
      {
        name: "CPU",
        color: SERIES_ACCENT,
        axis: "left",
        hasArea: true,
        values: resourceData.map((d) => d.cpu),
        format: (v) => `${v.toFixed(1)}%`,
      },
      {
        name: "Memory",
        color: SERIES_NEUTRAL,
        axis: "right",
        values: resourceData.map((d) => d.memory),
        format: prettyBytes,
      },
    ],
  },
  render: (args) => (
    <div className="w-180">
      <MultiAxisLineChart {...args} />
    </div>
  ),
};

const dailyDays = ["11/01", "11/02", "11/03", "11/04", "11/05", "11/06"];

export const MultiAxisChartDaily: StoryObj<typeof MultiAxisLineChart> = {
  args: {
    label: "Storage size per day",
    times: dailyDays.map((_, i) => i),
    formatTime: (t) => dailyDays[t],
    timeTicks: [0, 2, 4],
    left: {
      min: 0,
      max: 4_000_000_000,
      ticks: evenTicks(0, 1_000_000_000, 4),
      format: (v) => (v === 0 ? "0" : prettyBytes(v)),
    },
    lines: [
      {
        name: "Storage size",
        color: SERIES_ACCENT,
        axis: "left",
        hasArea: true,
        values: [
          undefined,
          undefined,
          1_200_000_000,
          2_100_000_000,
          2_900_000_000,
          3_400_000_000,
        ],
        format: prettyBytes,
      },
    ],
  },
  render: (args) => (
    <div className="w-180">
      <MultiAxisLineChart {...args} />
    </div>
  ),
};

export const MultiAxisChartLonePoint: StoryObj<typeof MultiAxisLineChart> = {
  ...MultiAxisChartDaily,
  args: {
    ...MultiAxisChartDaily.args,
    label: "Storage size on the first day of history",
    lines: [
      {
        name: "Storage size",
        color: SERIES_ACCENT,
        axis: "left",
        hasArea: true,
        values: [
          undefined,
          undefined,
          undefined,
          undefined,
          undefined,
          3_400_000_000,
        ],
        format: prettyBytes,
      },
    ],
  },
};

export const Sparkline: StoryObj<typeof SparkBars> = {
  args: {
    data: [0, 2, 1, 0, 3, 4, 2, 0, 0, 5, 3, 2, 4, 6, 1, 0, 2, 3, 5, 4].map(
      (value, index) => ({ label: `Day ${index + 1}`, value })
    ),
  },
  render: (args) => (
    <div className="w-40">
      <SparkBars {...args} />
    </div>
  ),
};

export const SparklineOutcomes: StoryObj<typeof SparkBars> = {
  args: {
    data: [62, 70, 58, 12, 66, 71, 9, 64].map((value, index) => ({
      label: `Job ${index + 1}`,
      value,
      tone: value < 20 ? "error" : "success",
      tooltip: `Job ${index + 1} · ${value < 20 ? "Failed" : "Succeeded"} · ${value}s`,
    })),
  },
  render: (args) => (
    <div className="w-40">
      <SparkBars {...args} />
    </div>
  ),
};

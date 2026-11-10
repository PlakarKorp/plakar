import type { Meta, StoryObj } from "@storybook/react-vite";
import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from "@tanstack/react-query";
import {
  createColumnHelper,
  useTable as useReactTable,
} from "@tanstack/react-table";

import { PaginatedTable } from "./paginated-table";
import { AppTableFeatures, appTableFeatures } from "./table-features";

type Job = { id: string; day: string; time: string; task: string };

const jobs: Job[] = [
  { id: "1", day: "Today", time: "14:02", task: "Backup laptop.local" },
  { id: "2", day: "Today", time: "09:30", task: "Check S3 store" },
  { id: "3", day: "Yesterday", time: "22:15", task: "Sync to offsite store" },
  { id: "4", day: "Yesterday", time: "14:00", task: "Backup laptop.local" },
  { id: "5", day: "Monday, Nov 2, 2026", time: "14:00", task: "Restore" },
];

const columnHelper = createColumnHelper<AppTableFeatures, Job>();
const columns = columnHelper.columns([
  columnHelper.accessor("time", { header: () => "Time" }),
  columnHelper.accessor("task", { header: () => "Task" }),
]);
const queryClient = new QueryClient();

function GroupedTable() {
  const query = useQuery({
    queryKey: ["jobs"],
    queryFn: () => ({ items: jobs, total: jobs.length }),
  });

  const table = useReactTable({
    features: appTableFeatures,
    data: query.data?.items ?? [],
    columns,
    getRowId: (row) => row.id,
    manualPagination: true,
  });

  return (
    <PaginatedTable<Job>
      query={query}
      table={table}
      ariaLabel="Jobs"
      getGroup={(job) => ({ key: job.day, label: job.day })}
      disablePagination
    />
  );
}

const meta = {
  title: "components/Paginated table",
  component: GroupedTable,
  decorators: [
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <Story />
      </QueryClientProvider>
    ),
  ],
} satisfies Meta<typeof GroupedTable>;

export default meta;
type Story = StoryObj<typeof meta>;

export const GroupedRows: Story = {};

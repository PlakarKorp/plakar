import { IntegrationIcon } from "@plakar-ui/common/components";
import {
  Button,
  Checkbox,
  Icon,
  Link,
  Modal,
  Spinner,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";

import { useInstallIntegration } from "~/api/integrations/mutations";
import { listIntegrationsQueryOptions } from "~/api/integrations/queries";
import type { Integration, IntegrationType } from "~/api/integrations/schema";

const MAX_UPDATES = 100;

type UpdateState =
  | { status: "idle" }
  | { status: "pending" }
  | { status: "done" }
  | { status: "failed"; message: string };

type ReviewFilters = {
  search: string;
  types: IntegrationType[] | undefined;
};

type ReviewUpdatesControlsProps = ReviewFilters & {
  count: number;
  isDisabled: boolean;
};

export function ReviewUpdatesControls({
  count,
  isDisabled,
  search,
  types,
}: ReviewUpdatesControlsProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <Button
        variant="secondary"
        leftIcon="arrow-up"
        onPress={() => setIsOpen(true)}
        isDisabled={isDisabled}
      >
        Review updates
        <Tag
          variant="yellow_borderless"
          radius="sqrt"
          size="sm"
          className="ml-2"
        >
          {count}
        </Tag>
      </Button>
      {isOpen && (
        <ReviewUpdatesModal
          search={search}
          types={types}
          onClose={() => setIsOpen(false)}
        />
      )}
    </>
  );
}

function ReviewUpdatesModal({
  search,
  types,
  onClose,
}: ReviewFilters & { onClose: () => void }) {
  const query = useQuery({
    ...listIntegrationsQueryOptions({
      search,
      types,
      installationStatus: "upgradable",
      page: 0,
      perPage: MAX_UPDATES,
    }),
    refetchInterval: false,
  });
  const install = useInstallIntegration();

  const [reviewed, setReviewed] = useState<Integration[]>();
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [states, setStates] = useState<Record<string, UpdateState>>({});
  const [isRunning, setIsRunning] = useState(false);

  if (query.data && reviewed === undefined) {
    setReviewed(query.data.items);
    setSelected(new Set(query.data.items.map(({ id }) => id)));
  }

  const rows = reviewed ?? [];
  const hasRun = Object.keys(states).length > 0;
  const isAllSelected = rows.length > 0 && selected.size === rows.length;

  const toggle = (id: string, isSelected: boolean) =>
    setSelected((previous) => {
      const next = new Set(previous);
      if (isSelected) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });

  const failedIds = rows
    .filter(({ id }) => states[id]?.status === "failed")
    .map(({ id }) => id);

  const runUpdates = async (ids: string[]) => {
    setIsRunning(true);
    for (const integration of rows.filter(({ id }) => ids.includes(id))) {
      setStates((previous) => ({
        ...previous,
        [integration.id]: { status: "pending" },
      }));
      try {
        await install.mutateAsync({
          integrationId: integration.id,
          version: integration.latest_version,
        });
        setStates((previous) => ({
          ...previous,
          [integration.id]: { status: "done" },
        }));
      } catch (error) {
        setStates((previous) => ({
          ...previous,
          [integration.id]: {
            status: "failed",
            message: error instanceof Error ? error.message : String(error),
          },
        }));
      }
    }
    setIsRunning(false);
  };

  return (
    <Modal
      isOpen
      isDismissable={!isRunning}
      onOpenChange={(isOpen) => !isOpen && !isRunning && onClose()}
      aria-label="Review updates"
    >
      <Modal.Dialog className="max-w-2xl">
        <Modal.Header
          title="Review updates"
          description="Selected integrations update one after another."
        />
        <Modal.Content className="flex flex-col p-0!">
          {query.isPending ? (
            <div className="p-4">
              <Spinner />
            </div>
          ) : query.isError ? (
            <Typography size="sm" className="text-error p-4">
              {query.error.message}
            </Typography>
          ) : (
            <>
              <div className="border-secondary bg-secondary flex items-center gap-3 border-b px-4 py-2">
                <Checkbox
                  size="sm"
                  aria-label="Select all updates"
                  isSelected={isAllSelected}
                  isIndeterminate={selected.size > 0 && !isAllSelected}
                  isDisabled={isRunning || hasRun}
                  onChange={(isSelected) =>
                    setSelected(
                      isSelected ? new Set(rows.map(({ id }) => id)) : new Set()
                    )
                  }
                />
                <Typography
                  as="span"
                  size="xs"
                  weight="medium"
                  className="text-tertiary"
                >
                  {selected.size} of {rows.length} selected
                </Typography>
              </div>
              <ul className="max-h-[24rem] overflow-y-auto">
                {rows.map((integration) => (
                  <UpdateRow
                    key={integration.id}
                    integration={integration}
                    isSelected={selected.has(integration.id)}
                    onToggle={(isSelected) =>
                      toggle(integration.id, isSelected)
                    }
                    isLocked={isRunning || hasRun}
                    state={states[integration.id] ?? { status: "idle" }}
                  />
                ))}
              </ul>
              {query.data.total > rows.length && (
                <Typography
                  size="xs"
                  className="text-tertiary border-secondary border-t px-4 py-2"
                >
                  Showing the first {rows.length} of {query.data.total} updates.
                </Typography>
              )}
            </>
          )}
        </Modal.Content>
        <Modal.Footer className="flex flex-wrap justify-end gap-2">
          {hasRun && !isRunning && failedIds.length === 0 ? (
            <Button onPress={onClose}>Done</Button>
          ) : hasRun && !isRunning ? (
            <>
              <Button variant="secondary" onPress={onClose}>
                Close
              </Button>
              <Button
                leftIcon="arrows-rotate"
                onPress={() => runUpdates(failedIds)}
              >
                {`Retry ${failedIds.length} failed`}
              </Button>
            </>
          ) : (
            <>
              <Button
                variant="secondary"
                onPress={onClose}
                isDisabled={isRunning}
              >
                Cancel
              </Button>
              <Button
                leftIcon="arrow-up"
                onPress={() => runUpdates([...selected])}
                isPending={isRunning}
                isDisabled={selected.size === 0 || isRunning}
              >
                {`Update ${selected.size} ${selected.size === 1 ? "integration" : "integrations"}`}
              </Button>
            </>
          )}
        </Modal.Footer>
      </Modal.Dialog>
    </Modal>
  );
}

type UpdateRowProps = {
  integration: Integration;
  isSelected: boolean;
  onToggle: (isSelected: boolean) => void;
  isLocked: boolean;
  state: UpdateState;
};

function UpdateRow({
  integration,
  isSelected,
  onToggle,
  isLocked,
  state,
}: UpdateRowProps) {
  return (
    <li className="border-secondary flex flex-col gap-1 border-b px-4 py-2.5 last:border-b-0">
      <div className="flex items-center gap-3">
        <Checkbox
          size="sm"
          aria-label={`Update ${integration.display_name}`}
          isSelected={isSelected}
          isDisabled={isLocked}
          onChange={onToggle}
        />
        <span className="bg-tertiary flex size-7 shrink-0 items-center justify-center overflow-hidden rounded-md p-1">
          <IntegrationIcon
            name={integration.name}
            icon={integration.icon}
            size="md"
          />
        </span>
        <Typography
          as="span"
          size="sm"
          weight="semibold"
          className="text-primary min-w-0 flex-1 truncate"
        >
          {integration.display_name}
        </Typography>
        <Typography
          as="span"
          size="xs"
          monospace
          className="text-tertiary shrink-0"
        >
          {integration.installation.version} →{" "}
          <span className="text-primary">{integration.latest_version}</span>
        </Typography>
        <UpdateStatus state={state} />
        {integration.homepage && (
          <Link
            href={integration.homepage}
            target="_blank"
            className="inline-flex shrink-0 items-center gap-1 text-xs font-medium"
          >
            Homepage
            <Icon type="arrow-up-right" size="xs" />
          </Link>
        )}
      </div>
      {state.status === "failed" && (
        <Typography size="xs" className="text-error pl-7">
          {state.message}
        </Typography>
      )}
    </li>
  );
}

function UpdateStatus({ state }: { state: UpdateState }) {
  switch (state.status) {
    case "pending":
      return <Spinner size="sm" />;
    case "done":
      return (
        <Icon
          type="check"
          size="sm"
          className="text-success"
          aria-label="Updated"
        />
      );
    case "failed":
      return (
        <Icon
          type="close"
          size="sm"
          className="text-error"
          aria-label="Update failed"
        />
      );
    default:
      return null;
  }
}

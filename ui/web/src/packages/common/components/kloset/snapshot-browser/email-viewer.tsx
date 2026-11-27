import {
  Alert,
  Avatar,
  Button,
  DescriptionList,
  Disclosure,
  Icon,
  SegmentedControl,
  Spinner,
  Tag,
  Typography,
} from "@plakar-ui/ui/components";
import { useQuery } from "@tanstack/react-query";
import clsx from "clsx";
import { useEffect, useMemo, useState } from "react";

import { Size } from "@plakar-ui/common/api/enrich";
import { triggerBlobDownload } from "@plakar-ui/common/utils";

import {
  bodyDocument,
  EmailAddress,
  emailFlags,
  EmailAttachment,
  parseEmail,
  ParsedEmail,
} from "./email-message";

type EmailViewerProps = {
  variant: "panel" | "stage";
  name: string;
  url: string;
  onError: () => void;
  onExpand?: () => void;
};

type BodyView = "html" | "text" | "source";

const PANEL_RECIPIENTS = 2;

export function EmailViewer({
  variant,
  name,
  url,
  onError,
  onExpand,
}: EmailViewerProps) {
  const { data: email, isError } = useQuery({
    queryKey: ["email", url],
    queryFn: async () => {
      const res = await fetch(url);
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      return parseEmail(await res.arrayBuffer());
    },
    staleTime: Infinity,
    // A parse error fails again, so retrying only downloads the file again.
    retry: false,
    // A parsed message holds its attachments in memory. Keep it long enough
    // for the panel to lightbox switch, not for every message browsed.
    gcTime: 30_000,
  });
  useEffect(() => {
    if (isError) onError();
  }, [isError, onError]);

  if (!email) {
    return <Spinner />;
  }
  // Keyed so that the view and "Load images" reset on the next message.
  return (
    <EmailMessage
      key={url}
      variant={variant}
      name={name}
      email={email}
      onExpand={onExpand}
    />
  );
}

function EmailMessage({
  variant,
  name,
  email,
  onExpand,
}: {
  variant: EmailViewerProps["variant"];
  name: string;
  email: ParsedEmail;
  onExpand?: () => void;
}) {
  const isStage = variant === "stage";
  const views = [
    email.html && { id: "html", label: isStage ? "Rendered" : "HTML" },
    email.text && { id: "text", label: "Plain text" },
    isStage && { id: "source", label: "Source" },
  ].filter((view) => !!view);
  const [view, setView] = useState<BodyView>(email.html ? "html" : "text");
  const [remoteContent, setRemoteContent] = useState(false);
  const showRemoteNotice =
    view === "html" && !remoteContent && email.hasRemoteContent;

  return (
    <div
      className={clsx(
        "flex min-h-0 flex-1 flex-col overflow-hidden",
        isStage ? "gap-3 p-6" : "gap-3 p-4"
      )}
    >
      <EmailHeader
        variant={variant}
        name={name}
        email={email}
        onExpand={onExpand}
      />
      {email.attachments.length > 0 && (
        <Attachments variant={variant} attachments={email.attachments} />
      )}
      {showRemoteNotice && (
        <Alert
          size="sm"
          variant="warning"
          title="Remote images are blocked"
          actions={
            <Button
              size="xs"
              variant="secondary"
              onPress={() => setRemoteContent(true)}
            >
              Load images
            </Button>
          }
        >
          They could tell the sender that this message was opened.
        </Alert>
      )}
      {views.length > 1 && (
        <SegmentedControl
          size="sm"
          ariaLabel="Message view"
          items={views}
          selectedId={view}
          onSelect={(id) => setView(id as BodyView)}
          className="self-start"
        />
      )}
      <EmailBody view={view} email={email} remoteContent={remoteContent} />
    </div>
  );
}

function EmailHeader({
  variant,
  name,
  email,
  onExpand,
}: {
  variant: EmailViewerProps["variant"];
  name: string;
  email: ParsedEmail;
  onExpand?: () => void;
}) {
  const isStage = variant === "stage";
  const flags = emailFlags(name);
  const sender = email.from?.name || email.from?.address || "Unknown sender";

  return (
    <header className="flex flex-col gap-3">
      <div className="flex flex-wrap items-start gap-2">
        <Typography
          variant="heading"
          size={isStage ? "lg" : "md"}
          weight="semibold"
          className="min-w-0 flex-1 break-words"
        >
          {email.subject || "(no subject)"}
        </Typography>
        {flags && !flags.read && (
          <Tag size="xs" variant="blue">
            Unread
          </Tag>
        )}
        {flags?.flagged && (
          <Tag size="xs" variant="yellow">
            Flagged
          </Tag>
        )}
        {!isStage && onExpand && (
          <Button
            size="xs"
            variant="tertiary"
            leftIcon="arrow-up-right"
            onPress={onExpand}
          >
            Open in viewer
          </Button>
        )}
      </div>
      <div className="flex min-w-0 items-center gap-3">
        <Avatar text={sender} size={isStage ? "md" : "sm"} />
        <div className="min-w-0 flex-1">
          <Typography size="sm" weight="semibold" className="truncate">
            {sender}
          </Typography>
          {email.from?.name && (
            <Typography size="xs" className="text-tertiary truncate">
              {email.from.address}
            </Typography>
          )}
        </div>
        {email.date && (
          <Typography size="xs" className="text-tertiary shrink-0">
            {email.date.toLocaleString()}
          </Typography>
        )}
      </div>
      <DescriptionList direction="row" singleColumn size="xs">
        <Recipients label="To" addresses={email.to} isStage={isStage} />
        <Recipients label="Cc" addresses={email.cc} isStage={isStage} />
      </DescriptionList>
      {isStage && (
        <Disclosure title="Details">
          <DescriptionList direction="row" singleColumn size="xs">
            {email.date && (
              <DescriptionList.Item
                title="Date"
                description={email.date.toString()}
              />
            )}
            {email.messageId && (
              <DescriptionList.Item
                title="Message-ID"
                description={email.messageId}
              />
            )}
            {flags && (
              <DescriptionList.Item
                title="Flags"
                description={
                  [flags.read && "Read", flags.flagged && "Flagged"]
                    .filter(Boolean)
                    .join(", ") || "None"
                }
              />
            )}
          </DescriptionList>
        </Disclosure>
      )}
    </header>
  );
}

function Recipients({
  label,
  addresses,
  isStage,
}: {
  label: string;
  addresses: EmailAddress[];
  isStage: boolean;
}) {
  const [isExpanded, setExpanded] = useState(false);
  if (addresses.length === 0) {
    return null;
  }
  const shown =
    isStage || isExpanded ? addresses : addresses.slice(0, PANEL_RECIPIENTS);
  const hidden = addresses.length - shown.length;
  return (
    <DescriptionList.Item
      title={label}
      description={
        <>
          {shown
            .map((a) => (a.name ? `${a.name} <${a.address}>` : a.address))
            .join(", ")}
          {hidden > 0 && (
            <button
              type="button"
              className="text-tertiary ml-1 underline"
              onClick={() => setExpanded(true)}
            >
              +{hidden} more
            </button>
          )}
        </>
      }
    />
  );
}

function Attachments({
  variant,
  attachments,
}: {
  variant: EmailViewerProps["variant"];
  attachments: EmailAttachment[];
}) {
  return (
    <ul
      className={clsx(
        "flex gap-2",
        variant === "stage" ? "flex-wrap" : "overflow-x-auto"
      )}
    >
      {attachments.map((attachment, i) => (
        <li key={i} className="shrink-0">
          <button
            type="button"
            title={`Download ${attachment.filename}`}
            onClick={() =>
              triggerBlobDownload({
                blob: new Blob([attachment.content], {
                  type: attachment.mimeType,
                }),
                filename: attachment.filename,
              })
            }
            className={clsx(
              "border-secondary hover:bg-primary-hover flex items-center gap-2 rounded-md border text-left",
              variant === "stage" ? "px-3 py-2" : "px-2 py-1"
            )}
          >
            <Icon type="download" size="sm" />
            <span className="max-w-48 truncate text-xs">
              {attachment.filename}
            </span>
            <span className="text-tertiary text-xs">
              {Size(attachment.content.byteLength).Pretty}
            </span>
          </button>
        </li>
      ))}
    </ul>
  );
}

function EmailBody({
  view,
  email,
  remoteContent,
}: {
  view: BodyView;
  email: ParsedEmail;
  remoteContent: boolean;
}) {
  const srcDoc = useMemo(
    () => email.html && bodyDocument(email.html, { remoteContent }),
    [email.html, remoteContent]
  );
  const source = useMemo(
    () => (view === "source" ? new TextDecoder().decode(email.bytes) : ""),
    [view, email.bytes]
  );
  if (view === "html" && srcDoc) {
    return (
      <iframe
        title="Email body"
        srcDoc={srcDoc}
        sandbox="allow-popups allow-popups-to-escape-sandbox"
        className="border-secondary min-h-0 w-full flex-1 rounded-md border bg-white"
        style={{ colorScheme: "light" }}
      />
    );
  }
  const text = view === "source" ? source : email.text;
  if (!text) {
    return (
      <Typography size="sm" className="text-tertiary">
        This message has no body.
      </Typography>
    );
  }
  return (
    <pre
      className={clsx(
        "border-secondary min-h-0 flex-1 overflow-auto rounded-md border p-4 text-xs whitespace-pre-wrap",
        view === "source" && "font-mono"
      )}
    >
      {text}
    </pre>
  );
}

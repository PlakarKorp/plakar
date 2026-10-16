import clsx from "clsx";
import { PropsWithChildren, ReactNode } from "react";
import { MdClose, MdFullscreen, MdRemove } from "react-icons/md";

import { CopyButton } from "..";

type TerminalProps = PropsWithChildren<{
  className?: string;
  title?: string;
}>;

function TerminalRoot({ className, children, title }: TerminalProps) {
  return (
    <div className={clsx("rounded-b-lg", className)}>
      <Header title={title} />
      <Content>{children}</Content>
    </div>
  );
}

type HeaderProps = {
  title?: string;
};

function Header({ title }: HeaderProps) {
  return (
    <div className="flex items-center gap-2 rounded-t-md border-b border-b-neutral-400 bg-neutral-200 p-1 pl-2">
      <div className="flex gap-2">
        <MdClose className="rounded-full border border-red-600 bg-red-500 text-red-700" />
        <MdRemove className="rounded-full border border-yellow-600 bg-yellow-500 text-yellow-700" />
        <MdFullscreen className="rounded-full border border-green-600 bg-green-500 text-green-700" />
      </div>

      {title && <span className="font-semibold text-neutral-600">{title}</span>}
    </div>
  );
}

function Content({ children }: PropsWithChildren) {
  return (
    <div className="rounded-b-md border-b border-transparent bg-black p-2 text-sm text-neutral-300">
      {children}
    </div>
  );
}

type CommandProps = PropsWithChildren<{
  prompt: ReactNode;
  command?: string;
  cursor?: boolean;
}>;

function Command({ prompt, command, children, cursor }: CommandProps) {
  return (
    <div>
      {prompt}
      {command && (
        <span className="group/command">
          {command}
          <span className="-my-2 ml-1 hidden align-middle group-hover/command:inline-block">
            <CopyButton content={command} tooltip="Copy to clipboard" />
          </span>
        </span>
      )}
      {children}
      {cursor && <Cursor />}
    </div>
  );
}

type CommandOutputLineProps = {
  children: ReactNode;
  cursor?: boolean;
};

function CommandOutputLine({ children, cursor }: CommandOutputLineProps) {
  return (
    <div className="break-all whitespace-pre-wrap">
      {children}
      {cursor && <Cursor />}
    </div>
  );
}

type PromptProps = {
  username: string;
  host: string;
  path: string;
};

function Prompt({ username, host, path }: PromptProps) {
  return (
    <span className="mr-1 font-semibold">
      <span className="text-cyan-200">
        {username}@{host}:
      </span>
      <span className="text-red-400">{path}</span>
    </span>
  );
}

function Cursor() {
  return <span className="bg-neutral-300 text-transparent">x</span>;
}

export const Terminal = Object.assign(TerminalRoot, {
  Command,
  CommandOutputLine,
  Prompt,
});

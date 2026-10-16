import { json } from "@codemirror/lang-json";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { unifiedMergeView } from "@codemirror/merge";
import { tags } from "@lezer/highlight";
import CodeMirror, { EditorView } from "@uiw/react-codemirror";
import clsx from "clsx";
import { ComponentProps } from "react";

import { useTheme } from "../../../common/hooks/use-theme";
import { FieldLabel } from "../field/field-label";
import { FieldSupportText } from "../field/field-support-text";

// Token colors for the dark theme. The default highlight style CodeMirror
// applies through basicSetup is drawn for a light background, so the dark side
// needs light tokens of its own. The light side keeps the default.
const darkHighlightStyle = HighlightStyle.define([
  { tag: tags.propertyName, color: "var(--color-sky-300)" },
  {
    tag: [tags.string, tags.special(tags.string)],
    color: "var(--color-emerald-300)",
  },
  { tag: tags.number, color: "var(--color-amber-300)" },
  {
    tag: [tags.bool, tags.null, tags.keyword, tags.atom],
    color: "var(--color-violet-300)",
  },
  { tag: tags.punctuation, color: "var(--text-color-tertiary)" },
  {
    tag: tags.comment,
    color: "var(--text-color-tertiary)",
    fontStyle: "italic",
  },
  { tag: tags.invalid, color: "var(--text-color-error)" },
]);

export type CodeEditorProps = ComponentProps<typeof CodeMirror> & {
  label?: string;
  supportText?: string;
  isRequired?: boolean;
  isInvalid?: boolean;
  suffix?: React.ReactNode;
  beforeLabelAction?: React.ReactNode;
  labelAction?: React.ReactNode;
  rows?: number;

  /**
   * Document to compare `value` against. When given, the editor shows one
   * unified diff instead of the document alone: removals appear above the line
   * that replaced them, and long unchanged stretches collapse.
   */
  original?: string;
};

export function CodeEditor({
  label,
  supportText,
  isRequired = false,
  isInvalid = false,
  suffix,
  beforeLabelAction,
  labelAction,
  rows,
  original,
  ...props
}: CodeEditorProps) {
  const theme = useTheme();

  // Inspired by https://github.com/codemirror/theme-one-dark/blob/main/src/one-dark.ts
  // Removed most of the unused features (for tooltips, tabs, etc) to keep it simple.
  const myTheme = EditorView.theme(
    {
      "&": {
        color: "var(--text-color-primary)",
        backgroundColor: "var(--background-color-primary)",
      },
      ".cm-scroller": rows ? { minHeight: `calc(${rows}lh + 8px)` } : {},
      // Editor content area
      ".cm-content": {
        // I couldn't find how this color is used. Apparently it's for the
        // cursor, so let's keep it in doubt.
        caretColor: "var(--text-color-placeholder)",
      },
      // Cursor color
      ".cm-cursor, .cm-dropCursor": {
        borderLeftColor: "var(--text-color-placeholder)",
      },
      // Selection background color
      "&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection":
        {
          backgroundColor:
            theme.value === "dark"
              ? "var(--blue-brand-300)"
              : "var(--blue-brand-50)",
        },
      // Current line
      ".cm-activeLine": {
        backgroundColor: "transparent",
      },
      // Matching bracket highlight background
      "&.cm-focused .cm-matchingBracket, &.cm-focused .cm-nonmatchingBracket": {
        backgroundColor:
          theme.value === "dark"
            ? "var(--blue-brand-900)"
            : "var(--blue-brand-100)",
      },
      // Gutters, containing line numbers
      ".cm-gutters": {
        backgroundColor: "var(--background-color-tertiary)",
        color: "var(--text-color-tertiary)",
        border: "none",
      },
      // Gutters for the current line
      ".cm-activeLineGutter": {
        backgroundColor:
          theme.value === "dark"
            ? "var(--blue-brand-900)"
            : "var(--blue-brand-100)",
      },

      // Diff, when `original` is given. The editor holds the new document, so
      // it is the merge view's "b" side; removals are widgets it renders above
      // the line. The palette tokens already flip with the theme, so these
      // rules need no light/dark branch of their own.
      "&.cm-merge-b .cm-changedLine, .cm-inlineChangedLine": {
        backgroundColor: "var(--background-color-success-tertiary)",
      },
      "&.cm-merge-b .cm-changedText": {
        background: "var(--background-color-success-secondary)",
      },
      "&.cm-merge-b .cm-changedLineGutter": {
        background: "var(--border-color-success)",
      },
      ".cm-deletedChunk": {
        backgroundColor: "var(--background-color-error-tertiary)",
      },
      ".cm-deletedChunk .cm-deletedText": {
        background: "var(--background-color-error-secondary)",
      },
      ".cm-deletedLineGutter": {
        background: "var(--border-color-error)",
      },
      ".cm-collapsedLines": {
        color: "var(--text-color-tertiary)",
        background: "var(--background-color-tertiary)",
      },
    },
    { dark: theme.value === "dark" }
  );

  return (
    <div
      className="group relative flex h-full flex-col gap-1"
      data-invalid={isInvalid ? true : undefined}
    >
      {(label || beforeLabelAction || labelAction) && (
        <FieldLabel
          isRequired={isRequired}
          beforeAction={beforeLabelAction}
          action={labelAction}
        >
          {label}
        </FieldLabel>
      )}
      <CodeMirror
        className={clsx(
          "flex-1 overflow-hidden rounded-lg border",
          isInvalid ? "border-error" : "border-primary"
        )}
        height="100%"
        theme={myTheme}
        extensions={[
          json(),
          ...(theme.value === "dark"
            ? [syntaxHighlighting(darkHighlightStyle)]
            : []),
          ...(original === undefined
            ? []
            : unifiedMergeView({
                original,
                // A viewer, not a merge tool: there is nothing to accept into.
                mergeControls: false,
                gutter: true,
                collapseUnchanged: { margin: 2 },
              })),
        ]}
        basicSetup={{
          tabSize: 2,
          indentOnInput: true,
        }}
        {...props}
      />
      {supportText && <FieldSupportText>{supportText}</FieldSupportText>}

      <div className="absolute top-0 right-0">{suffix}</div>
    </div>
  );
}

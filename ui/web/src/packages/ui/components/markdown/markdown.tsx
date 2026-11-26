import clsx from "clsx";
import ReactMarkdown from "react-markdown";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { dracula as dark } from "react-syntax-highlighter/dist/esm/styles/prism";
import remarkGfm from "remark-gfm";

import { Typography } from "..";

type MarkdownProps = {
  content: string;

  // urlTransform is used to transform the URL of images in the markdown.
  // It is useful when the images are stored in a different location than the markdown.
  // For example, if the images are stored in a .ptar file, the urlTransform
  // function can be used to resolve the relative path to an absolute URL
  // pointing to the API serving the .ptar content.
  urlImageTransform?: (url: string) => string;

  // sm renders smaller headings and text, for content shown inside a modal
  // or a card.
  size?: "md" | "sm";
};

export function Markdown({
  content,
  urlImageTransform = (url) => url,
  size = "md",
}: MarkdownProps) {
  const compact = size === "sm";

  return (
    <div
      className={clsx(
        "text-primary flex flex-col",
        compact ? "gap-3" : "gap-4"
      )}
    >
      <ReactMarkdown
        skipHtml={false}
        remarkPlugins={[remarkGfm]}
        urlTransform={(url, _key, node) => {
          if (node.tagName === "img") {
            return urlImageTransform(url);
          }
          return url;
        }}
        components={{
          h1({ children }) {
            return (
              <Typography
                as="h1"
                variant="heading"
                size={compact ? "sm" : "md"}
                weight="medium"
                children={children}
              />
            );
          },
          h2({ children }) {
            return (
              <Typography
                as="h2"
                variant="heading"
                size={compact ? "xs" : "md"}
                weight="medium"
                children={children}
              />
            );
          },
          h3({ children }) {
            return (
              <Typography
                as="h3"
                variant={compact ? "body" : "heading"}
                size={compact ? "sm" : "md"}
                weight="medium"
                children={children}
              />
            );
          },
          h4({ children }) {
            return (
              <Typography
                as="h4"
                variant={compact ? "body" : "heading"}
                size={compact ? "sm" : "md"}
                weight="medium"
                children={children}
              />
            );
          },
          h5({ children }) {
            return (
              <Typography
                as="h5"
                variant={compact ? "body" : "heading"}
                size={compact ? "sm" : "md"}
                children={children}
              />
            );
          },
          h6({ children }) {
            return (
              <Typography
                as="h6"
                variant={compact ? "body" : "heading"}
                size={compact ? "sm" : "md"}
                children={children}
              />
            );
          },
          a({ href, children }) {
            return (
              <Typography
                as="a"
                className="text-accent hover:text-accent-hover"
                href={href}
                children={children}
              />
            );
          },
          em({ children }) {
            return <Typography as="i" className="italic" children={children} />;
          },
          code({ className, children }) {
            const match = /language-(\w+)/.exec(className ?? "");

            return match ? (
              <SyntaxHighlighter
                wrapLines
                wrapLongLines
                // .replace to remove the trailing newlines
                children={String(children).replace(/\n+$/, "")}
                language={match[1]}
                style={dark}
              />
            ) : (
              <Typography
                as="code"
                monospace
                size={compact ? "xs" : "md"}
                className={clsx(
                  "text-primary",
                  compact && "bg-tertiary rounded px-1 py-0.5"
                )}
              >
                {children}
              </Typography>
            );
          },
          ul({ children }) {
            return (
              <ul
                className={clsx(
                  "text-secondary list-disc",
                  compact ? "list-outside space-y-1 pl-5" : "list-inside"
                )}
                children={children}
              />
            );
          },
          ol({ children }) {
            return (
              <ol
                className={clsx(
                  "list-decimal",
                  compact ? "list-outside space-y-1 pl-5" : "list-inside"
                )}
                children={children}
              />
            );
          },
          li({ children }) {
            return (
              <Typography
                as="li"
                size={compact ? "sm" : "md"}
                children={children}
              />
            );
          },
          p({ children }) {
            return (
              <Typography
                as="p"
                variant="body"
                size={compact ? "sm" : "md"}
                children={children}
                className={clsx("text-secondary", !compact && "inline")}
              />
            );
          },
          strong({ children }) {
            return (
              <Typography
                as="strong"
                children={children}
                weight="semibold"
                className="text-primary"
              />
            );
          },
          img({ src, alt }) {
            return (
              // The typography is rendered when the image fails to load.
              <Typography
                as="img"
                className="border-secondary text-tertiary max-w-full rounded-lg border text-center"
                src={src}
                alt={alt}
              />
            );
          },
          blockquote({ children }) {
            return (
              <Typography
                as="blockquote"
                className="border-accent ml-2 border-l-4 px-4 py-2"
                children={children}
              />
            );
          },
          hr() {
            return <hr className="border-secondary" />;
          },
          del({ children }) {
            return <Typography as="del" children={children} />;
          },
          // XXX: uncomment and use the new Checkbox component if props.type === 'checkbox'
          // input({ node: _node, ...props }) {
          //   return <input {...props} />;
          // },
          th({ children }) {
            return (
              <Typography
                as="th"
                className="border-secondary border p-2"
                weight="medium"
                children={children}
              />
            );
          },
          td({ children }) {
            return (
              <Typography
                as="td"
                className="border-secondary border p-2"
                children={children}
              />
            );
          },
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

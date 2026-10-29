import { expect, test } from "vitest";

import { ServerSentEventsParser } from "./sse";

test("reads events however the chunks are cut", () => {
  const parser = new ServerSentEventsParser();
  const events = [
    ...parser.push("event: text\nda"),
    ...parser.push('ta: {"text":"hel'),
    ...parser.push('lo"}\n\nevent: done\ndata: {}\n'),
    ...parser.push("\n"),
  ];
  expect(events).toEqual([
    { name: "text", data: '{"text":"hello"}' },
    { name: "done", data: "{}" },
  ]);
});

test("joins multi-line data, accepts CRLF and skips comments", () => {
  const parser = new ServerSentEventsParser();
  expect(
    parser.push(": keep-alive\r\nevent: text\r\ndata: a\r\ndata: b\r\n\r\n")
  ).toEqual([{ name: "text", data: "a\nb" }]);
});

test("defaults the name and drops an event without data", () => {
  const parser = new ServerSentEventsParser();
  expect(parser.push("data: x\n\nevent: empty\n\n")).toEqual([
    { name: "message", data: "x" },
  ]);
});

test("flushes an event the stream ended without closing", () => {
  const parser = new ServerSentEventsParser();
  expect(parser.push("event: done\ndata: {}\n")).toEqual([]);
  expect(parser.flush()).toEqual({ name: "done", data: "{}" });
  expect(parser.flush()).toBeUndefined();
});

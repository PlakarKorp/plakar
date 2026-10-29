export type ServerSentEvent = {
  name: string;
  data: string;
};

/**
 * Splits a text/event-stream into events as its chunks arrive, however the
 * network cut them. Only the event and data fields are read; comments and the
 * other fields are skipped.
 */
export class ServerSentEventsParser {
  private buffer = "";
  private name = "";
  private data: string[] = [];

  /** Feeds a chunk of the stream and returns the events it completed. */
  push(chunk: string): ServerSentEvent[] {
    this.buffer += chunk;
    const events: ServerSentEvent[] = [];

    for (
      let index = this.buffer.indexOf("\n");
      index !== -1;
      index = this.buffer.indexOf("\n")
    ) {
      const line = this.buffer.slice(0, index).replace(/\r$/, "");
      this.buffer = this.buffer.slice(index + 1);

      if (line === "") {
        const event = this.flush();
        if (event) {
          events.push(event);
        }
        continue;
      }
      if (line.startsWith(":")) {
        continue;
      }

      const colon = line.indexOf(":");
      const field = colon === -1 ? line : line.slice(0, colon);
      let value = colon === -1 ? "" : line.slice(colon + 1);
      if (value.startsWith(" ")) {
        value = value.slice(1);
      }

      if (field === "event") {
        this.name = value;
      } else if (field === "data") {
        this.data.push(value);
      }
    }

    return events;
  }

  /**
   * Returns the event whose fields were read but not yet closed by a blank
   * line, and forgets it. An event without data is dropped, as the standard
   * says.
   */
  flush(): ServerSentEvent | undefined {
    const name = this.name || "message";
    const data = this.data;
    this.name = "";
    this.data = [];

    if (data.length === 0) {
      return undefined;
    }
    return { name, data: data.join("\n") };
  }
}

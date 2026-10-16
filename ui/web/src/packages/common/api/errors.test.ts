import { expect, test } from "vitest";

import { APIResponseBadRequest } from "./errors";

test("uses the detail field from a JSON response body as the message", () => {
  const error = new APIResponseBadRequest(
    "POST",
    "/foo",
    JSON.stringify({ detail: "field 'name' is required" })
  );
  expect(error.message).toBe("field 'name' is required");
});

test("falls back to a generic message when the body isn't JSON", () => {
  const error = new APIResponseBadRequest("POST", "/foo", "not json");
  expect(error.message).toBe("BadRequest");
});

test("falls back to a generic message when the body has no detail field", () => {
  const error = new APIResponseBadRequest(
    "POST",
    "/foo",
    JSON.stringify({ title: "invalid_configuration" })
  );
  expect(error.message).toBe("BadRequest");
});

test("falls back to a generic message when detail isn't a string", () => {
  const error = new APIResponseBadRequest(
    "POST",
    "/foo",
    JSON.stringify({ detail: 42 })
  );
  expect(error.message).toBe("BadRequest");
});

test("keeps the raw response payload available", () => {
  const payload = JSON.stringify({ detail: "field 'name' is required" });
  const error = new APIResponseBadRequest("POST", "/foo", payload);
  expect(error.responsePayload).toBe(payload);
});

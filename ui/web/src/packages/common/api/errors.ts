import { z } from "zod";

export class APIError extends Error {
  public readonly method: string;
  public readonly url: string;

  constructor(method: string, url: string, message: string) {
    super(message);
    this.name = "APIError";
    this.method = method;
    this.url = url;
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

export class APIAbortError extends Error {
  constructor() {
    super("Request aborted");
    this.name = "APIAbortError";
    Object.setPrototypeOf(this, new.target.prototype);
  }
}

export class APINetworkError extends APIError {
  public readonly originalError: Error;

  constructor(
    method: string,
    url: string,
    message: string,
    originalError: Error
  ) {
    super(method, url, message);
    this.originalError = originalError;
  }

  toString() {
    return `${this.method} ${this.url}: ${this.originalError.message}`;
  }
}

export class APIResponseError extends APIError {
  public readonly code: number;

  constructor(method: string, url: string, message: string, code: number) {
    super(method, url, message);
    this.code = code;
    this.name = "APIResponseError";
  }

  toString() {
    return `${this.method} ${this.url}: HTTP/${this.code} ${this.message}`;
  }
}

const badRequestPayloadSchema = z.object({ detail: z.string() }).partial();

function badRequestMessage(responsePayload: string): string {
  let json: unknown;
  try {
    json = JSON.parse(responsePayload);
  } catch {
    return "BadRequest";
  }
  const parsed = badRequestPayloadSchema.safeParse(json);
  return parsed.success && parsed.data.detail
    ? parsed.data.detail
    : "BadRequest";
}

export class APIResponseBadRequest extends APIResponseError {
  public responsePayload: string;

  constructor(method: string, url: string, responsePayload: string) {
    super(method, url, badRequestMessage(responsePayload), 400);
    this.name = "BadRequestError";
    this.responsePayload = responsePayload;
  }
}

export class APIResponseUnauthorizedError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Unauthorized", 401);
    this.name = "APIResponseUnauthorizedError";
  }
}

export class APIResponseForbiddenError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Forbidden", 403);
    this.name = "APIResponseForbiddenError";
  }
}

export class APIResponseNotFoundError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Not found", 404);
    this.name = "APIResponseNotFoundError";
  }
}

export class APIResponseConflictError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Conflict", 409);
    this.name = "APIResponseConflictError";
  }
}

export class APIResponseTooManyRequestsError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Too many requests", 429);
    this.name = "APIResponseTooManyRequestsError";
  }
}

export class APIResponseInternalError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Internal server error", 500);
    this.name = "APIResponseInternalError";
  }
}

export class APIResponseNotImplementedError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Not implemented", 501);
    this.name = "APIResponseNotImplementedError";
  }
}

export class APIResponseServiceUnavailableError extends APIResponseError {
  constructor(method: string, url: string) {
    super(method, url, "Service Unavailable", 503);
    this.name = "APIResponseServiceUnavailableError";
  }
}

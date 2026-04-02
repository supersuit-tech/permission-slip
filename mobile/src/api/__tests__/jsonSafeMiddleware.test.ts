import { jsonSafeMiddleware } from "../client";

/** Helper to invoke the onResponse callback with a minimal context. */
async function runMiddleware(response: Response): Promise<Response> {
  const result = await jsonSafeMiddleware.onResponse!({
    response,
    request: new Request("https://example.com/v1/approvals"),
    schemaPath: "/v1/approvals",
    params: {},
    id: "test-id",
    options: {} as never,
  });
  return result ?? response;
}

describe("jsonSafeMiddleware", () => {
  it("passes JSON responses through unchanged", async () => {
    const body = JSON.stringify({ data: [1, 2, 3] });
    const original = new Response(body, {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });

    const result = await runMiddleware(original);
    expect(result).toBe(original);
  });

  it("passes JSON responses with charset through unchanged", async () => {
    const body = JSON.stringify({ ok: true });
    const original = new Response(body, {
      status: 200,
      headers: { "Content-Type": "application/json; charset=utf-8" },
    });

    const result = await runMiddleware(original);
    expect(result).toBe(original);
  });

  it("converts HTML 200 response to 502 JSON error", async () => {
    const original = new Response("<!DOCTYPE html><html><body>App</body></html>", {
      status: 200,
      headers: { "Content-Type": "text/html" },
    });

    const result = await runMiddleware(original);
    expect(result.status).toBe(502);
    expect(result.headers.get("content-type")).toBe("application/json");

    const parsed = await result.json();
    expect(parsed.error.code).toBe("non_json_response");
    expect(parsed.error.message).toMatch(/unable to reach the server/i);
    expect(parsed.error.retryable).toBe(true);
  });

  it("preserves 4xx/5xx status for non-JSON error responses", async () => {
    const original = new Response("<html>503 Service Unavailable</html>", {
      status: 503,
      statusText: "Service Unavailable",
      headers: { "Content-Type": "text/html" },
    });

    const result = await runMiddleware(original);
    expect(result.status).toBe(503);
    expect(result.statusText).toBe("Service Unavailable");

    const parsed = await result.json();
    expect(parsed.error.code).toBe("non_json_response");
  });

  it("converts text/plain responses to JSON error", async () => {
    const original = new Response("CORS origin not allowed\n", {
      status: 403,
      headers: { "Content-Type": "text/plain; charset=utf-8" },
    });

    const result = await runMiddleware(original);
    expect(result.status).toBe(403);

    const parsed = await result.json();
    expect(parsed.error.code).toBe("non_json_response");
  });

  it("passes 204 No Content through unchanged", async () => {
    const original = new Response(null, { status: 204 });

    const result = await runMiddleware(original);
    expect(result).toBe(original);
  });

  it("passes 304 Not Modified through unchanged", async () => {
    const original = new Response(null, { status: 304 });

    const result = await runMiddleware(original);
    expect(result).toBe(original);
  });
});

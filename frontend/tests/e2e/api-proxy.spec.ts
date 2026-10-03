import { expect, test } from "@playwright/test";

import {
    apiTimeout,
    apiUnavailable,
    forwardJSON,
    isJSONRequest,
    proxyToAPI,
} from "../../src/lib/apiProxy";

/**
 * The route handlers forward the status of the Go API. These tests need no
 * browser: they feed forwardJSON the responses the API can give, including
 * those without a JSON body, which used to be reported as a 502.
 */
test.describe("API proxy", () => {
    test("forwards a JSON response with its status", async () => {
        const forwarded = await forwardJSON(
            Response.json([{ id: 1 }], { status: 200 }),
        );

        expect(forwarded.status).toBe(200);
        expect(await forwarded.json()).toEqual([{ id: 1 }]);
    });

    test("forwards a JSON error with its status", async () => {
        const forwarded = await forwardJSON(
            Response.json({ error: "forbidden" }, { status: 403 }),
        );

        expect(forwarded.status).toBe(403);
        expect(await forwarded.json()).toEqual({ error: "forbidden" });
    });

    test("keeps the status of an error without a JSON body", async () => {
        const responses = [
            // The Go router for a route the deployed API does not have.
            new Response("404 page not found\n", {
                status: 404,
                headers: { "Content-Type": "text/plain; charset=utf-8" },
            }),
            // The Clerk middleware for a session token it cannot verify.
            new Response(null, { status: 401 }),
            new Response("<html>Internal Server Error</html>", { status: 500 }),
        ];

        for (const response of responses) {
            const status = response.status;

            const forwarded = await forwardJSON(response);

            expect(forwarded.status).toBe(status);
            expect(await forwarded.json()).toEqual({
                error: `appointment API responded with status ${status}`,
            });
        }
    });

    test("answers 502 to a success without a JSON body", async () => {
        const forwarded = await forwardJSON(
            new Response("<html>Not the API</html>", { status: 200 }),
        );

        expect(forwarded.status).toBe(502);
    });

    test("answers 502 when the API cannot be reached", async () => {
        const response = apiUnavailable();

        expect(response.status).toBe(502);
        expect(await response.json()).toEqual({
            error: "failed to connect to appointment API",
        });
    });

    test("keeps Retry-After and forbids caching", async () => {
        const forwarded = await forwardJSON(
            Response.json(
                { error: "rate limit exceeded" },
                { status: 429, headers: { "Retry-After": "42" } },
            ),
        );

        expect(forwarded.status).toBe(429);
        expect(forwarded.headers.get("Retry-After")).toBe("42");
        expect(forwarded.headers.get("Cache-Control")).toBe("no-store");
    });

    test("answers 504 when the API does not respond in time", async () => {
        const response = apiTimeout();

        expect(response.status).toBe(504);
        expect(await response.json()).toEqual({
            error: "appointment API did not respond in time",
        });
    });

    test("answers 502 without details when nothing listens at API_URL", async () => {
        const original = process.env.API_URL;

        // Port 1 is reserved and never has a server, so the connection fails.
        process.env.API_URL = "http://127.0.0.1:1";

        try {
            const response = await proxyToAPI("/appointments/availability?date=2026-09-28");

            expect(response.status).toBe(502);
            expect(await response.json()).toEqual({
                error: "failed to connect to appointment API",
            });
        } finally {
            // Assigning undefined would store the string "undefined".
            if (original === undefined) {
                delete process.env.API_URL;
            } else {
                process.env.API_URL = original;
            }
        }
    });

    test("forwards a 204 without a body", async () => {
        const forwarded = await withFetch(
            async () => new Response(null, { status: 204 }),
            () => proxyToAPI("/admin/appointments/1", { method: "DELETE" }),
        );

        expect(forwarded.status).toBe(204);
        expect(await forwarded.text()).toBe("");
    });

    test("accepts only JSON request bodies", () => {
        const withType = (contentType?: string) =>
            new Request("http://localhost/api/appointments", {
                method: "POST",
                headers: contentType ? { "Content-Type": contentType } : {},
            });

        expect(isJSONRequest(withType("application/json"))).toBe(true);
        expect(isJSONRequest(withType("application/json; charset=utf-8"))).toBe(true);
        expect(isJSONRequest(withType("text/plain;charset=UTF-8"))).toBe(false);
        expect(isJSONRequest(withType("application/x-www-form-urlencoded"))).toBe(false);
        expect(isJSONRequest(withType())).toBe(false);
    });
});

/**
 * Runs fn with fetch replaced by fakeFetch, so proxyToAPI can be tested
 * against responses no running API gives on demand.
 */
async function withFetch<T>(
    fakeFetch: typeof fetch,
    fn: () => Promise<T>,
): Promise<T> {
    const original = globalThis.fetch;

    globalThis.fetch = fakeFetch;

    try {
        return await fn();
    } finally {
        globalThis.fetch = original;
    }
}

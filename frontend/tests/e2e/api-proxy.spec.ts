import { expect, test } from "@playwright/test";

import { apiUnavailable, forwardJSON } from "../../src/lib/apiProxy";

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
});

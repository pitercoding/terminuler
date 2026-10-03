import { expect, test } from "@playwright/test";

/**
 * Production hardening checks against the running Next.js server: the
 * security headers, the admin protection that no longer relies on the Clerk
 * middleware matching paths, and the rejection of cross-site bookings.
 */
test.describe("Security", () => {
    test("pages send the security headers", async ({ request }) => {
        const response = await request.get("/");

        expect(response.ok()).toBe(true);

        const headers = response.headers();

        expect(headers["content-security-policy"]).toContain("frame-ancestors 'none'");
        expect(headers["x-frame-options"]).toBe("DENY");
        expect(headers["x-content-type-options"]).toBe("nosniff");
        expect(headers["referrer-policy"]).toBe("strict-origin-when-cross-origin");
        expect(headers["x-powered-by"]).toBeUndefined();
    });

    test("the admin dashboard is not rendered without a session", async ({
        request,
    }) => {
        const response = await request.get("/admin", { maxRedirects: 0 });

        // auth.protect() redirects a signed-out visitor to the Clerk sign-in.
        expect([302, 303, 307, 308]).toContain(response.status());
        expect(await response.text()).not.toContain("Admin Dashboard");
    });

    test("the admin API rejects requests without a session", async ({
        request,
    }) => {
        const list = await request.get("/api/admin/appointments");

        expect(list.status()).toBe(401);
        expect(await list.json()).toEqual({ error: "unauthorized" });

        const cancel = await request.delete("/api/admin/appointments/1");

        expect(cancel.status()).toBe(401);
    });

    test("a booking that is not JSON is rejected", async ({ request }) => {
        // What a form or fetch on another site can send without a preflight.
        const response = await request.post("/api/appointments", {
            headers: { "Content-Type": "text/plain;charset=UTF-8" },
            data: JSON.stringify({
                appointment_date: "2026-09-28",
                start_time: "10:00",
                end_time: "11:00",
                customer_name: "Cross Site",
                customer_phone: "+49 151 33333333",
                customer_email: "cross-site@example.com",
            }),
        });

        expect(response.status()).toBe(415);
        expect(await response.json()).toEqual({
            error: "content type must be application/json",
        });
    });

    test("invalid JSON reaches the API and gets its 400", async ({ request }) => {
        const response = await request.post("/api/appointments", {
            headers: { "Content-Type": "application/json" },
            data: "{not json",
        });

        expect(response.status()).toBe(400);
        expect(await response.json()).toEqual({ error: "invalid request body" });
    });
});

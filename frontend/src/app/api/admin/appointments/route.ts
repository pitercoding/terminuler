import { auth } from "@clerk/nextjs/server";

import { proxyToAPI, unauthorized } from "@/lib/apiProxy";

/**
 * Forwards the request to the admin API with the Clerk session token of the
 * signed-in user. The Go API verifies the token and that the user is the
 * admin, so this route only has to reject requests without a session.
 */
export async function GET() {
    const { getToken } = await auth();

    const token = await getToken();

    if (!token) {
        return unauthorized();
    }

    return proxyToAPI("/admin/appointments", {
        headers: { Authorization: `Bearer ${token}` },
    });
}

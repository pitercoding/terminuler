import { auth } from "@clerk/nextjs/server";
import { NextResponse } from "next/server";

import { proxyToAPI } from "@/lib/apiProxy";

/**
 * Forwards the request to the admin API with the Clerk session token of the
 * signed-in user. The Go API verifies the token and that the user is the
 * admin, so this route only has to reject requests without a session.
 */
export async function GET() {
    const { getToken } = await auth();

    const token = await getToken();

    if (!token) {
        return NextResponse.json(
            { error: "unauthorized" },
            { status: 401 },
        );
    }

    return proxyToAPI("/admin/appointments", {
        headers: { Authorization: `Bearer ${token}` },
    });
}

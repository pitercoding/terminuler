import { auth } from "@clerk/nextjs/server";

import { proxyToAPI, unauthorized } from "@/lib/apiProxy";

/**
 * Cancels an appointment through the admin API with the Clerk session token
 * of the signed-in user. The Go API verifies the token and that the user is
 * the admin, so this route only has to reject requests without a session.
 */
export async function DELETE(
    _request: Request,
    { params }: { params: Promise<{ id: string }> },
) {
    const { getToken } = await auth();

    const token = await getToken();

    if (!token) {
        return unauthorized();
    }

    const { id } = await params;

    return proxyToAPI(`/admin/appointments/${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${token}` },
    });
}

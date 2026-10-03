import { auth } from "@clerk/nextjs/server";

/**
 * Protects every page under /admin on the server: a signed-out visitor is
 * redirected to the Clerk sign-in before any admin page is rendered. The
 * page itself is a Client Component, so the check lives here. It only
 * requires a session; whether that user is the admin is decided by the Go
 * API, which answers 403 to anyone else.
 */
export default async function AdminLayout({
    children,
}: LayoutProps<"/admin">) {
    await auth.protect();

    return children;
}

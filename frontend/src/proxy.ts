import { clerkMiddleware } from "@clerk/nextjs/server";

/**
 * Makes the Clerk session available to auth() in pages and route handlers.
 * It does not protect any route: each protected resource checks the session
 * itself (src/app/admin/layout.tsx and the /api/admin route handlers), as
 * Clerk recommends, so protection does not depend on path matching that
 * could diverge from how Next.js routes the request. The Go API still
 * verifies the token and the admin user on every admin request.
 */
export default clerkMiddleware();

export const config = {
    matcher: [
        "/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)",
        "/(api|trpc)(.*)",
    ],
};

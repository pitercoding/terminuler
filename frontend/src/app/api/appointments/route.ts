import { NextRequest } from "next/server";

import {
    isJSONRequest,
    proxyToAPI,
    unsupportedMediaType,
} from "@/lib/apiProxy";

/**
 * Returns the IP of the client that called this proxy, or null when it is
 * unknown. The last X-Forwarded-For entry is the one added by the closest
 * hop (the hosting platform, a reverse proxy or Next.js itself), while
 * earlier entries can be sent by the client and are not trusted.
 */
function clientIP(request: NextRequest): string | null {
    const forwardedFor = request.headers.get("x-forwarded-for");

    const ip = forwardedFor?.split(",").at(-1)?.trim();

    return ip || null;
}

/**
 * Forwards the booking to the Go API, which validates the body. The client
 * IP goes in X-Real-IP for the rate limit; the API only believes it when the
 * request comes from TRUSTED_PROXIES or carries API_PROXY_SECRET, a
 * server-only variable that never reaches the browser.
 */
export async function POST(request: NextRequest) {
    if (!isJSONRequest(request)) {
        return unsupportedMediaType();
    }

    const headers = new Headers({ "Content-Type": "application/json" });

    const ip = clientIP(request);

    if (ip) {
        headers.set("X-Real-IP", ip);
    }

    const proxySecret = process.env.API_PROXY_SECRET;

    if (proxySecret) {
        headers.set("X-Proxy-Secret", proxySecret);
    }

    // Forwarded as received: the API rejects invalid JSON with a 400, which
    // re-encoding it here would turn into a proxy error.
    const body = await request.text();

    return proxyToAPI("/appointments", {
        method: "POST",
        headers,
        body,
    });
}

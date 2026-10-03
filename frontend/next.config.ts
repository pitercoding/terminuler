import type { NextConfig } from "next";

/**
 * Security headers for every page and route handler.
 *
 * The CSP only sets directives that cannot break Next.js or Clerk: no other
 * site may frame the pages (clickjacking on the admin dashboard), <base> and
 * plugins are blocked. script-src is left out on purpose: Next.js inline
 * scripts and the Clerk scripts loaded from its Frontend API would need
 * nonces, which force every page to render dynamically.
 *
 * Strict-Transport-Security is not set here because Vercel already sends it
 * for its domains.
 */
const securityHeaders = [
  {
    key: "Content-Security-Policy",
    value: "frame-ancestors 'none'; base-uri 'self'; object-src 'none'",
  },
  // Same as frame-ancestors, for browsers that ignore CSP.
  { key: "X-Frame-Options", value: "DENY" },
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
  // Features the app never uses. WebAuthn is left allowed for Clerk passkeys.
  {
    key: "Permissions-Policy",
    value: "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
  },
];

const nextConfig: NextConfig = {
  poweredByHeader: false,

  async headers() {
    return [
      {
        source: "/:path*",
        headers: securityHeaders,
      },
    ];
  },
};

export default nextConfig;

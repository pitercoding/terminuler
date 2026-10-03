package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
	// Embeds the timezone database so APP_TIMEZONE works on hosts
	// without one (Windows, minimal Docker images).
	_ "time/tzdata"

	"github.com/joho/godotenv"
)

const (
	defaultPort     = "8080"
	defaultTimezone = "UTC"

	defaultResendFromEmail = "onboarding@resend.dev"

	defaultAppointmentRateLimit = 5
)

// Email providers accepted in EMAIL_PROVIDER.
const (
	// EmailProviderResend sends confirmation emails through Resend.
	EmailProviderResend = "resend"

	// EmailProviderLog only logs confirmation emails, for local testing
	// (such as the E2E tests) without sending real emails.
	EmailProviderLog = "log"
)

// envFiles are the locations checked for a .env file, relative to the
// working directory: the current directory and the repository root when
// running from backend/.
var envFiles = []string{
	".env",
	"../.env",
}

// Load reads the first .env file found. The file is optional so the
// application can run with environment variables provided by the host
// (Docker, CI, production). Variables already set in the environment
// always take precedence over values from the file.
func Load() error {
	for _, path := range envFiles {
		err := godotenv.Load(path)

		if err == nil {
			return nil
		}

		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}
	}

	return nil
}

func DatabaseURL() (string, error) {
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		return "", fmt.Errorf("DATABASE_URL is not set")
	}

	return databaseURL, nil
}

// EmailProvider returns how confirmation emails are delivered
// (EMAIL_PROVIDER), defaulting to Resend so a deployment never skips real
// emails unless it explicitly opts out.
func EmailProvider() (string, error) {
	provider := os.Getenv("EMAIL_PROVIDER")

	switch provider {
	case "":
		return EmailProviderResend, nil
	case EmailProviderResend, EmailProviderLog:
		return provider, nil
	default:
		return "", fmt.Errorf(
			"invalid EMAIL_PROVIDER %q: must be %q or %q",
			provider,
			EmailProviderResend,
			EmailProviderLog,
		)
	}
}

// ResendAPIKey returns the API key used to send confirmation emails
// (RESEND_API_KEY). It is required so a misconfigured deployment fails at
// startup instead of silently skipping every confirmation email.
func ResendAPIKey() (string, error) {
	apiKey := os.Getenv("RESEND_API_KEY")

	if apiKey == "" {
		return "", fmt.Errorf("RESEND_API_KEY is not set")
	}

	return apiKey, nil
}

// ResendFromEmail returns the sender address used for confirmation emails
// (RESEND_FROM_EMAIL), defaulting to Resend's testing address, which can
// only deliver to the email of the Resend account owner.
func ResendFromEmail() string {
	if from := os.Getenv("RESEND_FROM_EMAIL"); from != "" {
		return from
	}

	return defaultResendFromEmail
}

// Port returns the HTTP port (PORT), defaulting to 8080.
func Port() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}

	return defaultPort
}

// AppointmentRateLimit returns how many appointments a single client may
// create per minute (APPOINTMENT_RATE_LIMIT), defaulting to 5.
func AppointmentRateLimit() (int, error) {
	value := os.Getenv("APPOINTMENT_RATE_LIMIT")

	if value == "" {
		return defaultAppointmentRateLimit, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 {
		return 0, fmt.Errorf(
			"invalid APPOINTMENT_RATE_LIMIT %q: must be a positive integer",
			value,
		)
	}

	return limit, nil
}

// TrustedProxies returns the networks allowed to report the client IP in
// the X-Real-IP header (TRUSTED_PROXIES), as a comma-separated list of
// CIDRs or single IPs. It is empty by default, so the header is ignored
// unless the proxy in front of the API is explicitly trusted.
func TrustedProxies() ([]netip.Prefix, error) {
	value := os.Getenv("TRUSTED_PROXIES")

	if value == "" {
		return nil, nil
	}

	var prefixes []netip.Prefix

	for entry := range strings.SplitSeq(value, ",") {
		entry = strings.TrimSpace(entry)

		if entry == "" {
			continue
		}

		prefix, err := parsePrefix(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid TRUSTED_PROXIES entry %q: %w", entry, err)
		}

		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}

// minProxySecretLength keeps API_PROXY_SECRET long enough that it cannot be
// guessed: openssl rand -hex 32 produces 64 characters.
const minProxySecretLength = 32

// ProxySecret returns the secret the Next.js proxy sends in the
// X-Proxy-Secret header to be allowed to report the client IP
// (API_PROXY_SECRET). In production the proxy runs on Vercel, whose
// outgoing IPs are not fixed and cannot be listed in TRUSTED_PROXIES, so the
// secret is how the API tells the proxy from anyone else on the internet.
// It is optional: when empty, only TRUSTED_PROXIES is used.
func ProxySecret() (string, error) {
	secret := os.Getenv("API_PROXY_SECRET")

	if secret != "" && len(secret) < minProxySecretLength {
		return "", fmt.Errorf(
			"API_PROXY_SECRET must have at least %d characters",
			minProxySecretLength,
		)
	}

	return secret, nil
}

// parsePrefix accepts a CIDR or a single IP, which becomes a prefix that
// matches only that address.
func parsePrefix(value string) (netip.Prefix, error) {
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return netip.Prefix{}, err
		}

		return prefix.Masked(), nil
	}

	addr, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, err
	}

	return netip.PrefixFrom(addr, addr.BitLen()), nil
}

// Location returns the timezone used for business hours (APP_TIMEZONE),
// defaulting to UTC when it is not set.
func Location() (*time.Location, error) {
	timezone := os.Getenv("APP_TIMEZONE")

	if timezone == "" {
		timezone = defaultTimezone
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid APP_TIMEZONE %q: %w", timezone, err)
	}

	return location, nil
}

// ClerkSecretKey returns the Clerk secret key used to verify
// authenticated requests.
func ClerkSecretKey() (string, error) {
	secretKey := os.Getenv("CLERK_SECRET_KEY")

	if secretKey == "" {
		return "", fmt.Errorf("CLERK_SECRET_KEY is not set")
	}

	return secretKey, nil
}

// AdminClerkUserID returns the Clerk user ID allowed to access
// administrative endpoints.
func AdminClerkUserID() (string, error) {
	userID := os.Getenv("ADMIN_CLERK_USER_ID")

	if userID == "" {
		return "", fmt.Errorf("ADMIN_CLERK_USER_ID is not set")
	}

	return userID, nil
}

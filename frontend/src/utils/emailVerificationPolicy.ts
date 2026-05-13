const GMAIL_LIKE_DOMAINS = new Set(['gmail.com', 'googlemail.com'])

function extractEmailDomain(email: string): string {
  const value = String(email || '').trim().toLowerCase()
  const atIndex = value.lastIndexOf('@')
  if (atIndex <= 0 || atIndex >= value.length - 1) {
    return ''
  }
  return value.slice(atIndex + 1)
}

export function shouldSkipRegistrationEmailVerification(
  email: string,
  gmailVerificationBypassEnabled: boolean,
): boolean {
  if (!gmailVerificationBypassEnabled) {
    return false
  }
  return GMAIL_LIKE_DOMAINS.has(extractEmailDomain(email))
}

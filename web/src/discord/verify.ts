// discord request signature verification
import { verifyKey } from "discord-interactions";

/**
 * verifies that a request came from Discord by checking the signature
 * @param signature - The X-Signature-Ed25519 header
 * @param timestamp - The X-Signature-Timestamp header
 * @param body - The raw request body (must be string or Buffer)
 * @param publicKey - Discord application public key
 * @returns true if signature is valid, false otherwise
 */
export async function verifyDiscordSignature(
  signature: string,
  timestamp: string,
  body: string | Buffer,
  publicKey: string,
): Promise<boolean> {
  try {
    return await verifyKey(body, signature, timestamp, publicKey);
  } catch (error) {
    console.error("Discord signature verification failed:", error);
    return false;
  }
}

/**
 * extracts signature and timestamp from request headers
 * returns null if headers are missing
 */
export function extractDiscordHeaders(
  headers: Record<string, string | string[] | undefined>,
): { signature: string; timestamp: string } | null {
  const signature = Array.isArray(headers["x-signature-ed25519"])
    ? headers["x-signature-ed25519"][0]
    : headers["x-signature-ed25519"];

  const timestamp = Array.isArray(headers["x-signature-timestamp"])
    ? headers["x-signature-timestamp"][0]
    : headers["x-signature-timestamp"];

  if (!signature || !timestamp) {
    return null;
  }

  return { signature, timestamp };
}

const SOCKS_HOST = process.env.NEXT_PUBLIC_SOCKS_HOST || "72.62.0.219";
const SOCKS_PORT = process.env.NEXT_PUBLIC_SOCKS_PORT || "1081";

export function generateSocksUrl(username: string, password: string): string {
  const encoded = btoa(`${username}:${password}`);
  return `socks://${encoded}@${SOCKS_HOST}:${SOCKS_PORT}#${encodeURIComponent(`IR-VPN-${username}`)}`;
}

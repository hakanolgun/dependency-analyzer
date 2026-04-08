export function normalizeRepoUrl(url: string): string {
  if (!url) return url;

  let normalized = url;

  // Remove git+ prefix
  if (normalized.startsWith("git+")) {
    normalized = normalized.substring(4);
  }

  // Remove .git suffix
  if (normalized.endsWith(".git")) {
    normalized = normalized.substring(0, normalized.length - 4);
  }

  // Handle ssh:// protocols
  if (normalized.startsWith("ssh://")) {
    normalized = normalized.substring(6);
  }

  // Handle git@github.com:owner/repo or git@github.com/owner/repo
  const sshMatch = normalized.match(/^git@([^:]+):(.+)$/);
  if (sshMatch) {
    const host = sshMatch[1];
    const path = sshMatch[2];
    normalized = `https://${host}/${path}`;
  } else if (normalized.startsWith("git@")) {
    // Handle git@host/path
    normalized = "https://" + normalized.substring(4).replace(":", "/");
  }

  // Ensure https for git://
  if (normalized.startsWith("git://")) {
    normalized = "https" + normalized.substring(3);
  }

  return normalized;
}

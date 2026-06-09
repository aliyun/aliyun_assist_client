import { highlight, supportsLanguage } from "cli-highlight";

/**
 * Produce an ANSI-colored string for a code block.
 *
 * If a language is specified and supported, uses it for highlighting;
 * otherwise falls back to auto-detection. On any failure returns the
 * plain code unchanged.
 */
export function highlightCode(code: string, language?: string): string {
  try {
    const options: { language?: string; ignoreIllegals: boolean } = {
      ignoreIllegals: true,
    };
    if (language && supportsLanguage(language)) {
      options.language = language;
    }
    return highlight(code, options);
  } catch {
    // Graceful fallback: return plain text on any highlighting failure
    return code;
  }
}

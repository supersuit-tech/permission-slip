/**
 * Shared validation constants loaded from the single source of truth at
 * `shared/validation.json` in the repository root. The Go backend embeds the
 * same file at compile time, so both sides stay in sync.
 */
import config from "../../../shared/validation.json";

export default config;

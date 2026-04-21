import * as readline from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";
import type { AvailableConnectorInstance } from "./connectorInstance.js";

/**
 * Prompts for a connector instance when the API returns connector_instance_required.
 * Returns the selected UUID string, or undefined if the user cancels / invalid input.
 */
export async function promptConnectorInstanceChoice(
  instances: AvailableConnectorInstance[],
): Promise<string | undefined> {
  if (instances.length === 0) {
    return undefined;
  }
  const rl = readline.createInterface({ input, output });
  try {
    for (let i = 0; i < instances.length; i++) {
      const inst = instances[i]!;
      const label =
        inst.display_name !== undefined && inst.display_name !== ""
          ? `${inst.display_name} (${inst.id})`
          : inst.id;
      output.write(`${i + 1}. ${label}\n`);
    }
    const line = await rl.question(
      `Select connector instance (1-${instances.length}): `,
    );
    const n = parseInt(line.trim(), 10);
    if (!Number.isFinite(n) || n < 1 || n > instances.length) {
      return undefined;
    }
    return instances[n - 1]!.id;
  } finally {
    rl.close();
  }
}
